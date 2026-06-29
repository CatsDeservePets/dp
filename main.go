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

var (
	dryRun  = flag.Bool("n", false, "do not duplicate files, but show what would be done instead; implies -v")
	verbose = flag.Bool("v", false, "cause dp to be verbose, showing files as they are duplicated")
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: dp [-n] [-v] source ...")
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
	cp := copier{
		dryRun:  *dryRun,
		verbose: *verbose,
		output:  os.Stdout,
	}

	exitCode := 0
	for _, src := range flag.Args() {
		if err := duplicate(src, rule, cp); err != nil {
			log.Print(err)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func duplicate(src string, rule dupRule, cp copier) error {
	dst, err := nextDupPath(src, rule)
	if err != nil {
		return err
	}
	return cp.copyPath(src, dst)
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

// A copier copies source paths to destination paths.
type copier struct {
	dryRun  bool      // report copies without creating them
	verbose bool      // report copies as they are made
	output  io.Writer // destination for reports
}

// copyPath recursively copies src to dst. If an error occurs, copyPath stops
// and leaves any partial copy in place.
//
// Note: cp -R variants differ in how umask and special bits affect regular
// files and directories. For now, dp intentionally keeps this simple.
func (c copier) copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	mode := info.Mode()
	switch mode & os.ModeType {
	case os.ModeDir:
		// Add owner access while copying into the directory.
		if err := c.mkdir(dst, mode&os.ModePerm|0o700); err != nil {
			return err
		}
		c.report(src, dst)

		ents, err := os.ReadDir(src)
		if err == nil {
			for _, e := range ents {
				if err = c.copyPath(
					filepath.Join(src, e.Name()),
					filepath.Join(dst, e.Name()),
				); err != nil {
					break
				}
			}
		}
		// Restore the mode after trying to copy the contents.
		if err2 := c.chmod(dst, mode&(os.ModePerm|preservedDirBits)); err == nil {
			err = err2
		}
		return err
	case os.ModeSymlink:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := c.symlink(target, dst); err != nil {
			return err
		}
	case 0:
		if err := c.copyFile(src, dst, mode&os.ModePerm); err != nil {
			return err
		}
	default:
		return &os.PathError{Op: "copy", Path: src, Err: os.ErrInvalid}
	}

	c.report(src, dst)
	return nil
}

func (c copier) copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if c.dryRun {
		return nil
	}
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

func (c copier) mkdir(name string, perm os.FileMode) error {
	if c.dryRun {
		return nil
	}
	return os.Mkdir(name, perm)
}

func (c copier) chmod(name string, perm os.FileMode) error {
	if c.dryRun {
		return nil
	}
	return os.Chmod(name, perm)
}

func (c copier) symlink(oldname, newname string) error {
	if c.dryRun {
		return nil
	}
	return os.Symlink(oldname, newname)
}

func (c copier) report(src, dst string) {
	if (!c.verbose && !c.dryRun) || c.output == nil {
		return
	}
	fmt.Fprintln(c.output, src, "->", dst)
}
