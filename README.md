# PBibFix

A Pocketbook Application designed to fix entries in PocketBook's Library (Bibliothek) database.

## Rationale

PocketBook is very picky about metadata found in epubs.

* Infos about book series can only be found when they are in a specific order.
* Author names can only be found if the XML namespace is exactly `opf`

These are the first two issues this application is going to fix.

Please note: The application will **not** fix the meta data inside the box. It will only fix the entries in pocketbook's database.

## Compatibility

I only have a Touch HD3 with firmware U632.6.1.900 so this is the only Version where I can test.
Please feel free to report about other devices or firmware versions where you ran it successfully.

## Installation

Copy PBibFix.app to your pocketbook's applications directory.
Use it each time you copied new books onto the device or at least each time
you found a mistake in the library.

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