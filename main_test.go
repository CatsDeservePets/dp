package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNextDupPath(t *testing.T) {
	writeFile := func(name string) {
		t.Helper()

		name = filepath.FromSlash(name)
		if err := os.MkdirAll(filepath.Dir(name), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, nil, 0o666); err != nil {
			t.Fatal(err)
		}
	}

	t.Chdir(t.TempDir())
	writeFile("file.txt")
	writeFile("file copy.txt")
	writeFile("dir/file.txt")

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
