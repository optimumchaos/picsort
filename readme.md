# Purpose

Picsort is a command-line utility for sorting incoming pictures and videos into a destination photo library.  It sorts media by date, using the format `yyyy/yyyy-mm-dd/yyyy-mm-dd_hh-mm-ss-original-filename`, making reasonable attempts to exclude duplicate files.  It also has limited support for the Google Photo JSON metadata format, for getting dates and excluding "trashed" files.

Picsort uses [goexif](http://github.com/rwcarlsen/goexif/exif) to extract EXIF metadata, and therefore supports file formats recognized by [goexif](http://github.com/rwcarlsen/goexif/exif).

# Usage
## Getting
```
go get github.com/optimumchaos/picsort
```
## Running
```
picsort -incomingdir ~/incoming -libdir ~/Pictures -rejectdir ~/rejects
```
This will recursively scan all files in "~/incoming" for files with exif dates or Google metadata (file with the same name with the ".json" extension.)  It will create a directory structure in "~/Pictures" based on the dates within the incoming media, and move/rename them accordingly.  It will move any duplicates, unrecognized files, or files marked as "trashed" to subdirectories of "~/rejects", retaining the original directory structure from "~/incoming".  Finally, it cleans up the empty "incoming" directory and writes a script to undo everything.

There are a few options:
* `-dedupe lazy|eager`: By default, Picsort lazily deduplicates prior to moving each incoming file, scanning the destination directory.  This will be effective as long as your entire library is in the Picsort format.  It can also eagerly deduplicate, scanning the entire library upfront.  This will be effective regardless of the library format, but will take more time.
* `-dryrun`: Do not actually move any files.
* `-undofile`: The name of the undo script to write.  Defaults to "undo.sh".

To see all options:
```
picsort
```

Use this carefully, at your own risk.  Back up your files.

# Background and Motivations

I wrote this when Google turned off their function to sync between Google Photo and Google Drive, to help me keep control of my photo library.  My family uploads photos to Google Photo via smart phones and cameras, and until recently, I maintained library organization via a Google Script, and I sync'd them all down to my home server on a nightly basis.  With Drive sync turned off, I am now using the Google "Download Data" feature to get an archive every two months, and *Picsort* to sort them into my library.  I currently run this on my Synology DiskStation *experimentally*.

I could have borrowed one of the bash scripts available on the interwebs to do this sorting, but I wrote my own in Go so that I could add de-duplication.

While this seems to work, it is not done.  As of now, I'm still running this *very carefully*, with backups, and reviewing mysterious edge cases.  (I have files come down from Google that have the same name or that seem corrupt.  And it doesn't seem to extract dates from HEIC and HEIF files, relying on the Google metadata for that.)

I've tested this on Mac OSX (which is case insensitive) and Synology's Linux (which is case sensitive).  I have not tried it on Windows.

# Credits

Exif metadata handling by [goexif](http://github.com/rwcarlsen/goexif), which is licensed under BSD 2-clause license.  Refer to *goexif* for details.