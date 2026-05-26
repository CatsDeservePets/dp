package main

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
)

const (
	defaultFirst    = "%b copy%e"
	defaultNumbered = "%b copy %n%e"
)

var verbose = flag.Bool("v", false, "cause dp to be verbose, showing files as they are duplicated")

func usage() {
	fmt.Fprintln(os.Stderr, "usage: dp [-v] source ...")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("dp: ")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
	}

	rule, err := compileDupRule(
		cmp.Or(os.Getenv("DUPFMT_FIRST"), defaultFirst),
		cmp.Or(os.Getenv("DUPFMT_NUMBERED"), defaultNumbered),
	)
	if err != nil {
		log.Fatalf("bad duplicate format: %v", err)
	}

	exitCode := 0
	for _, src := range flag.Args() {
		if err := duplicate(src, rule, *verbose); err != nil {
			log.Print(err)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func duplicate(src string, rule dupRule, verbose bool) error {
	dst, err := nextDupPath(src, rule)
	if err != nil {
		return err
	}
	return copyPath(src, dst, verbose)
}

var (
	errDotPath   = errors.New(`"." and ".." may not be duplicated`)
	errNoDupName = errors.New("no available duplicate name")
)

func nextDupPath(src string, rule dupRule) (string, error) {
	// Check the final element before [filepath.Clean] rewrites the path.
	switch filepath.Base(src) {
	case ".", "..":
		return "", errDotPath
	}
	if _, err := os.Lstat(src); err != nil {
		return "", err
	}

	// Avoid [filepath.Split]. For "dir/", it returns "dir/" and "",
	// not the cleaned parent "." and final element "dir".
	src = filepath.Clean(src)
	dir := filepath.Dir(src)
	name := filepath.Base(src)
	stem, ext, seq := rule.parse(name)

	for seq < math.MaxInt {
		seq++
		dst := filepath.Join(dir, rule.format(stem, ext, seq))

		_, err := os.Lstat(dst)
		if errors.Is(err, os.ErrNotExist) {
			return dst, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", errNoDupName
}

const preservedDirBits = os.ModeSetuid | os.ModeSetgid | os.ModeSticky

// copyPath recursively copies src to dst. If verbose is true, copied paths
// are reported to stdout. copyPath stops at the first error and returns it.
//
// Note: cp -R variants differ in how umask and special bits affect regular
// files and directories. For now, dp intentionally keeps this simple.
func copyPath(src, dst string, verbose bool) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	mode := info.Mode()
	switch mode & os.ModeType {
	case os.ModeDir:
		// Add owner access while copying into the directory.
		if err := os.Mkdir(dst, mode&os.ModePerm|0o700); err != nil {
			return err
		}
		if verbose {
			fmt.Printf("%s -> %s\n", src, dst)
		}
		ents, err := os.ReadDir(src)
		if err == nil {
			for _, e := range ents {
				if err = copyPath(
					filepath.Join(src, e.Name()),
					filepath.Join(dst, e.Name()),
					verbose,
				); err != nil {
					break
				}
			}
		}
		// Restore the mode after trying to copy the contents.
		if err2 := os.Chmod(dst, mode&(os.ModePerm|preservedDirBits)); err == nil {
			err = err2
		}
		return err
	case os.ModeSymlink:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.Symlink(target, dst); err != nil {
			return err
		}
	case 0:
		if err := copyFile(src, dst, mode&os.ModePerm); err != nil {
			return err
		}
	default:
		return &os.PathError{Op: "copy", Path: src, Err: os.ErrInvalid}
	}

	if verbose {
		fmt.Printf("%s -> %s\n", src, dst)
	}
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return &os.PathError{Op: "copy", Path: dst, Err: err}
	}
	return out.Close()
}
