package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNextDupPath(t *testing.T) {
	chdirTemp(t)

	writeFile(t, "file.txt", "")
	writeFile(t, "file copy.txt", "")
	writeFile(t, "dir/file.txt", "")

	rule := mustCompileDupRule(t, "%b copy%e", "%b copy %n%e")

	tests := []struct {
		name    string
		src     string
		want    string
		wantErr error
	}{
		{
			name: "FirstDuplicateExists",
			src:  "file.txt",
			want: "file copy 2.txt",
		},
		{
			name: "NestedFile",
			src:  "dir/file.txt",
			want: "dir/file copy.txt",
		},
		{
			name: "TrailingSlashDirectory",
			src:  "dir/",
			want: "dir copy",
		},
		{
			name:    "MissingSource",
			src:     "missing.txt",
			wantErr: os.ErrNotExist,
		},
		{
			name:    "Dot",
			src:     ".",
			wantErr: errDotPath,
		},
		{
			name:    "DotDot",
			src:     "..",
			wantErr: errDotPath,
		},
		{
			name:    "NestedDot",
			src:     "dir/.",
			wantErr: errDotPath,
		},
		{
			name:    "NestedDotDot",
			src:     "dir/..",
			wantErr: errDotPath,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			src := filepath.FromSlash(test.src)
			want := filepath.FromSlash(test.want)

			got, err := nextDupPath(src, rule)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("nextDupPath(%q) error = %v, want %v", src, err, test.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got != want {
				t.Errorf("nextDupPath(%q) = %q, want %q", src, got, want)
			}
		})
	}
}

func TestCopyPath(t *testing.T) {
	const src, dst = "src", "dst"

	t.Run("RegularFile", func(t *testing.T) {
		chdirTemp(t)

		writeFile(t, src, "hello")

		if err := (copier{}).copyPath(src, dst); err != nil {
			t.Fatalf("copyPath(%q, %q) = %v, want nil", src, dst, err)
		}
		if got := readFile(t, dst); got != "hello" {
			t.Errorf("ReadFile(%q) = %q, want %q", dst, got, "hello")
		}
	})

	t.Run("Directory", func(t *testing.T) {
		chdirTemp(t)

		writeFile(t, "src/a.txt", "a")
		writeFile(t, "src/sub/b.txt", "b")

		if err := (copier{}).copyPath(src, dst); err != nil {
			t.Fatalf("copyPath(%q, %q) = %v, want nil", src, dst, err)
		}

		files := []struct {
			path string
			want string
		}{
			{path: "dst/a.txt", want: "a"},
			{path: "dst/sub/b.txt", want: "b"},
		}

		for _, file := range files {
			if got := readFile(t, file.path); got != file.want {
				t.Errorf("ReadFile(%q) = %q, want %q", file.path, got, file.want)
			}
		}
	})

	t.Run("Symlink", func(t *testing.T) {
		chdirTemp(t)

		writeFile(t, "target", "")
		if err := os.Symlink("target", src); err != nil {
			t.Skipf("Symlink: %v", err)
		}

		if err := (copier{}).copyPath(src, dst); err != nil {
			t.Fatalf("copyPath(%q, %q) = %v, want nil", src, dst, err)
		}

		got, err := os.Readlink(dst)
		if err != nil {
			t.Fatal(err)
		}
		if got != "target" {
			t.Errorf("Readlink(%q) = %q, want %q", dst, got, "target")
		}
	})

	t.Run("ExistingDestination", func(t *testing.T) {
		chdirTemp(t)

		writeFile(t, src, "src")
		writeFile(t, dst, "dst")

		err := (copier{}).copyPath(src, dst)
		if !errors.Is(err, os.ErrExist) {
			t.Fatalf("copyPath(%q, %q) = %v, want %v", src, dst, err, os.ErrExist)
		}
		if got := readFile(t, dst); got != "dst" {
			t.Errorf("ReadFile(%q) = %q, want %q", dst, got, "dst")
		}
	})

	t.Run("DryRun", func(t *testing.T) {
		chdirTemp(t)

		writeFile(t, src, "")

		var out strings.Builder
		cp := copier{
			dryRun: true,
			output: &out,
		}

		if err := cp.copyPath(src, dst); err != nil {
			t.Fatalf("copyPath(%q, %q) = %v, want nil", src, dst, err)
		}
		if _, err := os.Lstat(dst); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Lstat(%q) error = %v, want %v", dst, err, os.ErrNotExist)
		}

		want := fmt.Sprintln(src, "->", dst)
		if got := out.String(); got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})
}

func chdirTemp(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
}

func writeFile(t *testing.T, name, contents string) {
	t.Helper()

	name = filepath.FromSlash(name)
	if err := os.MkdirAll(filepath.Dir(name), 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(contents), 0o666); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.FromSlash(name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
