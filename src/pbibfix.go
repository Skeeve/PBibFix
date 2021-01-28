package main

import (
	"container/list"
	"database/sql"
	"github.com/Skeeve/PBibFix/pocketbook"
	"github.com/Skeeve/epub"
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const pocketbookFs = "/mnt/ext1"
const explorerFile = pocketbookFs + "/system/explorer-3/explorer-3.db"
const logfile = pocketbookFs + "/PBibFix.log"
const getEpubs = `
SELECT name AS path, filename, B.id, firstauthor, series, numinseries, sort_title
FROM books_impl B
JOIN files F
ON B.id = F.book_id
JOIN folders P
ON P.id = F.folder_id
WHERE ext = 'epub'
AND name !=''
;
`
const updateFirstAuthor = `
UPDATE books_impl
SET firstauthor = ?2
  , first_author_letter = SUBSTR(?2,1,1)
WHERE id = ?1
;
`
const updateSeries = `
UPDATE books_impl
SET series = ?2
  , numinseries = ?3
WHERE id = ?1
;
`

type ePubEntry struct {
	Path        string
	Filename    string
	ID          int32
	Firstauthor string
	Series      string
	Numinseries int32
	SortTitle   string
}

func main() {

	// open the logfile or die
	f, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(pocketbook.Fatal(err.Error()))
	}
	defer f.Close()
	logger := log.New(f, "", log.LstdFlags)

	// open book database
	db, err := sql.Open("sqlite3", explorerFile)
	if err != nil {
		logger.Fatal(pocketbook.Fatal("While opening the database\n" + err.Error()))
	}
	defer db.Close()

	// get all ePubs which have a storage location
	// on the device into a list
	epubs := list.New()
	rows, err := db.Query(getEpubs)
	if err != nil {
		logger.Fatal(pocketbook.Fatal("While running a database query\n" + err.Error()))
	}
	for rows.Next() {
		var entry ePubEntry
		if err := rows.Scan(
			&entry.Path,
			&entry.Filename,
			&entry.ID,
			&entry.Firstauthor,
			&entry.Series,
			&entry.Numinseries,
			&entry.SortTitle,
		); err != nil {
			rows.Close()
			logger.Fatal(pocketbook.Fatal("While getting a book entry from the database\n" + err.Error()))
		}
		epubs.PushFront(entry)
	}
	rows.Close()

	var authorsFixed int64
	var seriesFixed int64
	var errorCounter int64
	// iterate the ePubs
	for element := epubs.Front(); element != nil; element = element.Next() {

		entry := element.Value.(ePubEntry)

		// Does the book exist?
		filename := filepath.Join(entry.Path, entry.Filename)
		if _, err := os.Stat(filename); err != nil {
			continue // no. So skip
		}

		// read the books data
		book, err := epub.Open(filename)
		if err != nil {
			logger.Println(filename + ": " + err.Error())
			errorCounter++
			continue
		}
		defer book.Close()

	FileAs: // fix an empty "file-as" for the author
		for book.Opf.Metadata.Creator[0].FileAs == "" {

			// get the firstauthor's id
			id := "#" + book.Opf.Metadata.Creator[0].ID
			if id == "#" {
				// no id -> ignore this
				break FileAs
			}

			// try to find a refines for that id
			for _, meta := range book.Opf.Metadata.Meta {
				if meta.Refine != id {
					// not the correct id
					continue
				}
				if meta.Property != "file-as" {
					// not the required property
					continue
				}
				if entry.Firstauthor == meta.Data {
					// data is identical. No fix required
					continue
				}

				// Update the database
				logger.Println("Fixing Author book id", entry.ID, "filename", filename)

				_, err := db.Exec(updateFirstAuthor, entry.ID, meta.Data)
				if err != nil {
					logger.Fatal(pocketbook.Fatal("While fixing firstauthor for " + filename + "\n" + err.Error()))
				}
				authorsFixed++

			}
			break FileAs
		}
	Series:
		for _, meta := range book.Opf.Metadata.Meta {
			// find a belongs-to-collection
			var id string
			if meta.Property == "belongs-to-collection" {
				id = "#" + meta.ID
				if id == "#" {
					// no id -> ignore this
					continue Series
				}
			}
			seriesTitle := meta.Data
			isSeries := false
			var numInSeries int32
			// try to find a refines for that id
		Meta:
			for _, meta := range book.Opf.Metadata.Meta {
				if meta.Refine != id {
					// not the correct id
					continue Meta
				}
				switch meta.Property {
				case "collection-type":
					if meta.Data == "series" {
						isSeries = true
					}
				case "group-position":
					i, _ := strconv.ParseInt(meta.Data, 10, 32)
					numInSeries = int32(i)
				}
			}
			if !isSeries || numInSeries < 1 {
				continue Series
			}
			if entry.Series == seriesTitle && entry.Numinseries == numInSeries {
				continue Series
			}
			logger.Println("Fixing Series book id", entry.ID, "filename", filename)

			_, err := db.Exec(updateSeries, entry.ID, seriesTitle, numInSeries)
			if err != nil {
				logger.Fatal(pocketbook.Fatal("While fixing series for " + filename + "\n" + err.Error()))
			}
			seriesFixed++
		}

	}
	if authorsFixed == 0 && seriesFixed == 0 && errorCounter == 0 {
		pocketbook.Dialog(pocketbook.Info, "Nothing had to be fixed", "OK")
		os.Remove(logfile)
	} else {
		logger.Println("Done.")
		logger.Println("Authors fixed:", authorsFixed)
		logger.Println("Series fixed:", seriesFixed)
		logger.Println("Errors encountered:", errorCounter)
		if pocketbook.Dialog(pocketbook.Info,
			"Authors fixed: "+strconv.FormatInt(authorsFixed, 10)+
				"\nSeries fixed: "+strconv.FormatInt(seriesFixed, 10)+
				"\nErrors encountered: "+strconv.FormatInt(errorCounter, 10),
			"Delete Log", "OK") == 1 {
			os.Remove(logfile)
		}
	}
}

/*
	s, _ := json.MarshalIndent(meta, "", "\t")
	log.Println(string(s))
*/
