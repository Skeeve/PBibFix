# NOTE

Currently this code is unusable.

I tried to fix it but it seems the database structure of PocketBooks has changed in the last years.

I do not think it's worth updating, convince me otherwise by opening a new issue.

Or check this repository with a similar and, I think, more advanced tool: https://git.rustysoft.de/martin/PbDbFixer

# PBibFix

A Pocketbook Application designed to fix entries in PocketBook's Library (Bibliothek) database.

## Rationale

PocketBook is very picky about metadata found in epubs.

* Infos about book series can only be found when they are in a specific order.
* Author names can only be found if the XML namespace is exactly `opf`.
* (Not sure whether or not this is an issue) Books which were deleted from  the device and from the cloud still are present in the database.

These are the first two issues this application is going to fix.

Please note: The application will **not** fix the meta data inside the books. It will only fix the entries in pocketbook's database.

## Compatibility

I only have a Touch HD3 with firmware U632.6.1.900 so this is the only Version where I can test.
Please feel free to report about other devices or firmware versions where you ran it successfully.

## Installation

Copy PBibFix.app to your pocketbook's applications directory.
Use it each time you copied new books onto the device or at least each time
you found a mistake in the library.

## Usage

When you start the application it will first ask you, whether or not to remove any books from the database which are no longer present on the device and also are missing in the cloud, provided there are any.

It will then clean up those entries, if you wanted to and then fix other book entries.

At the end you will get a list of changes done and maybe errors encountered.

When running, the application creates a log file `PBibFix.log`.

This file is mainly for cases when the application encountered errors and you can delete it.

But in case you are interested what the script did, you can keep it.

Copy it over to your computer and view it with any text editor.

As the file extension is `.log` it's not easily possible to view the file directly on the device.

## Errors

Please find a list of possible error messages below.

* While opening the database
* While running a database query
* While getting deleted books from the database
* While starting a transaction
* While removing deleted books
* While removing entries from *table*
* While running a database query
* While getting a book entry from the database
* While fixing firstauthor for *filename*
* While fixing series for *filename*

All these messages are followed by some more text.

These issues might not be easy to solve, so please report them on [github](https://github.com/Skeeve/PBibFix/issues).

Describe the issues in German or English, please.

* *filename*: *errormessage*

This error can only be found in the log file.

The book with the *filename* did not parse correctly.

This usually happens with badly created ePub files.

To fix those issues, check that file using [calibre](https://calibre-ebook.com/) or [pagina's EPUB-checker](https://www.pagina.gmbh/produkte/epub-checker/).

For help with those issues, please go to [e-reader-forum](https://www.e-reader-forum.de/f/epub.197/).

Describe the issue well and it's most probable that someone there can assist you.

## Build

Building PBibFix for your PocketBook device can easily be done if you have [Docker](https://www.docker.com/) installed.
Simply execute:

```bash
docker-compose run make
cp src/PBibFix PBibFix.app
```

## Development

For testing purposes, without the requirement to run the tests on the device, the `dev` service is provided.

First make a backup of your PocketBook's filesystem and place it in the directory `ext1`.
On a Mac this can be done with (e.g.)

```bash
rsync -rav /Volumes/PB632/* ext1/
# or
mkdir ext1 ; cp -r /Volumes/PB632/* ext1/
```

Simply start the `dev` service

```bash
docker-compose run dev
```

and you will find yourself in a shell inside the `src` directory.
You will find under `/mnt/ext1` your PocketBook's backup you created earlier.

Under `/ebrmain/bin` you will find a small bash `dialog` script which (kind of) "mimics" the dialog program on the PocketBook.

From here you can do `go build` and run the resulting binary with `./PBibFix`.

## Software used

### https://github.com/dennwc/inkview

Denys provided a go pocketbook SDK.
I updated it to a more recent version of go and use it for cross-compiling to ARM7 for PocketBook devices.

### https://github.com/n3integration/epub

I use a fork of this epub module to extract a book's metadata.

### github.com/mattn/go-sqlite3

The "standard" go module for accessing sqlite.

### https://vscodium.com/

My current IDE of choice.
