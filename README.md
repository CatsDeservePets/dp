# dp

`dp` generates predictable duplicate names, like duplicating an item in a file manager.

It can duplicate files and directories itself, or provide the destination for use with another tool.

## Installation

```shell
go install github.com/CatsDeservePets/dp@latest
```

## Usage

```
usage: dp [-n] [-v] source ...
       dp -g source

  -g	generate a duplicate pathname and write it to standard output
  -n	do not duplicate files, but show what would be done instead; implies -v
  -v	cause dp to be verbose, showing files as they are duplicated
```

## Semantics

By default, `dp` uses Finder-like duplicate names:

```
file.txt        -> file copy.txt
file copy.txt   -> file copy 2.txt
file copy 2.txt -> file copy 3.txt
```

Path contents are duplicated roughly like `cp -R`: directories are traversed recursively, symlinks are not followed, and regular files are copied. For files, the source mode is modified by the process umask. Created directories have the same mode as their source after their contents have been copied. Ownership, timestamps, ACLs, and extended attributes are not preserved.

Each command-line argument is handled independently. If a source cannot be duplicated, `dp` reports the error and continues with the next argument. An error during copying may leave a partially copied duplicate in place.

For more control over copying, use `-g` and pass the resulting pathname to another tool:

```shell
src="$1"
dst=$(dp -g "$src") || exit
cp -Rp "$src" "$dst"
```

## Naming

Duplicate names can be configured using the environment variables `$DUPFMT_FIRST` and `$DUPFMT_NUMBERED`.

The following placeholders are supported:

```
%f    file name, including extension
%b    file stem, i.e. file name without extension
%e    extension, including the dot
%n    duplicate number
```

`$DUPFMT_FIRST` describes the first duplicate name and must not contain `%n`. `$DUPFMT_NUMBERED` describes the numbered names that follow and must contain exactly one `%n`, preceded by fixed text.

Both formats must use the same shape: either `%f` for the whole file name, or `%b` and `%e` for stem and extension.

When splitting a file name, the extension starts at the first dot after any leading dots:

```
archive.tar.gz -> archive copy.tar.gz
.bashrc        -> .bashrc copy
```

### Example formats

Finder-like (default):

```shell
DUPFMT_FIRST="%b copy%e"
DUPFMT_NUMBERED="%b copy %n%e"
```

```
file.txt -> file copy.txt -> file copy 2.txt
```

Explorer-like:

```shell
DUPFMT_FIRST="%b - Copy%e"
DUPFMT_NUMBERED="%b - Copy (%n)%e"
```

```
file.txt -> file - Copy.txt -> file - Copy (2).txt
```

Nautilus-like:

```shell
DUPFMT_FIRST="%b (Copy)%e"
DUPFMT_NUMBERED="%b (Copy %n)%e"
```

```
file.txt -> file (Copy).txt -> file (Copy 2).txt
```

Emacs-like:

```shell
DUPFMT_FIRST="%f.~1~"
DUPFMT_NUMBERED="%f.~%n~"
```

```
file.txt -> file.txt.~1~ -> file.txt.~2~
```
