package main

import (
	"container/list"
	"database/sql"
	"fmt"
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
const getDeletedBooks = `
SELECT B.id
     , B.title
FROM books_impl B
LEFT OUTER JOIN files F
ON B.id = F.book_id
WHERE filename is null
;
`
const removeDeletedBooks = `
DELETE FROM books_impl
WHERE id in (
	SELECT B.id
	FROM books_impl B
	LEFT OUTER JOIN files F
	ON B.id = F.book_id
	WHERE filename is null
)
;
`
const cleanTable = `
DELETE FROM %s WHERE %s NOT IN (
	SELECT id FROM books_impl
)
;
`

var tablesToClean = map[string]string{
	"books_settings":   "bookid",
	"books_uids":       "book_id",
	"bookshelfs_books": "bookid",
	"booktogenre":      "bookid",
	"social":           "bookid",
}

type ePubEntry struct {
	Path        string
	Filename    string
	ID          int32
	Firstauthor string
	Series      string
	Numinseries int32
	SortTitle   string
}

type bookTitle struct {
	ID    int32
	Title string
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

	// get all book entries not having any filename
	// These are thos books which once were on the device
	// but are now missing from there as well as from the cloud
	deletedBooks := list.New()
	rows, err := db.Query(getDeletedBooks)
	if err != nil {
		logger.Fatal(pocketbook.Fatal("While running a database query\n" + err.Error()))
	}
	for rows.Next() {
		var entry bookTitle
		if err := rows.Scan(&entry.ID, &entry.Title); err != nil {
			rows.Close()
			logger.Fatal(pocketbook.Fatal("While getting deleted books from the database\n" + err.Error()))
		}
		deletedBooks.PushFront(entry)
		logger.Println("Found deleted book", entry.ID, entry.Title)
	}
	rows.Close()

	var numDeleted int64 = int64(deletedBooks.Len())
	// Ask whether or not to remove the entries
	if numDeleted > 0 {
		if 1 == pocketbook.Dialog(pocketbook.Question,
			"Number of books deleted from Device and Cloud: "+strconv.FormatInt(numDeleted, 20)+
				"\n\nRemove them from the database?",
			"Yes", /*1*/
			"No",  /*2*/
		) {
			tx, err := db.Begin()
			if err != nil {
				logger.Fatal(pocketbook.Fatal("While starting a transaction\n" + err.Error()))
			}
			// Remove from books table
			res, err := tx.Exec(removeDeletedBooks)
			if err != nil {
				logger.Fatal(pocketbook.Fatal("While removing deleted books\n" + err.Error()))
			}
			numDeleted, _ = res.RowsAffected()
			logger.Println("Book entries removed:", numDeleted)
			// clean associated tables
			for table, column := range tablesToClean {
				command := fmt.Sprintf(cleanTable, table, column)
				res, err := tx.Exec(command)
				if err != nil {
					logger.Fatal(pocketbook.Fatal("While removing entries from " + table + "\n" + err.Error()))
				}
				entriesDeleted, _ := res.RowsAffected()
				logger.Println("Entries removed from "+table+":", entriesDeleted)
			}
			tx.Commit()
		} else {
			numDeleted = 0
		}
	}

	// get all ePubs which have a storage location
	// on the device into a list
	epubs := list.New()
	rows, err = db.Query(getEpubs)
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
	if authorsFixed == 0 && seriesFixed == 0 && numDeleted == 0 && errorCounter == 0 {
		pocketbook.Dialog(pocketbook.Info, "Nothing had to be fixed", "OK")
		os.Remove(logfile)
	} else {
		logger.Println("Done.")
		logger.Println("Authors fixed:", authorsFixed)
		logger.Println("Series fixed:", seriesFixed)
		logger.Println("Books cleaned from DB:", numDeleted)
		logger.Println("Errors encountered:", errorCounter)
		if pocketbook.Dialog(pocketbook.Info,
			"Authors fixed: "+strconv.FormatInt(authorsFixed, 10)+
				"\nSeries fixed: "+strconv.FormatInt(seriesFixed, 10)+
				"\nBooks cleaned from DB: "+strconv.FormatInt(numDeleted, 10)+
				"\nErrors encountered: "+strconv.FormatInt(errorCounter, 10),
			"Delete Log", "OK") == 1 {
			os.Remove(logfile)
		}
	}
}

/*
	s, _ := json.MarshalIndent(meta, "", "\t")
	logger.Println(string(s))
*/
