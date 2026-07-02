package main

import "testing"

func TestCompileDupRule(t *testing.T) {
	tests := []struct {
		name     string
		first    string
		numbered string
		wantErr  bool
	}{
		{
			name:     "StemExtStyle",
			first:    "%b copy%e",
			numbered: "%b copy %n%e",
		},
		{
			name:     "FullNameStyle",
			first:    "%f.~1~",
			numbered: "%f.~%n~",
		},
		{
			name:     "StyleMismatch",
			first:    "%b copy%e",
			numbered: "%f copy %n",
			wantErr:  true,
		},
		{
			name:     "FirstNoExt",
			first:    "%b copy",
			numbered: "%b copy %n%e",
			wantErr:  true,
		},
		{
			name:     "NumberedNoExt",
			first:    "%b copy%e",
			numbered: "%b copy %n",
			wantErr:  true,
		},
		{
			name:     "FirstNoText",
			first:    "%b%e",
			numbered: "%b copy %n%e",
			wantErr:  true,
		},
		{
			name:     "FirstHasNumber",
			first:    "%b copy %n%e",
			numbered: "%b copy %n%e",
			wantErr:  true,
		},
		{
			name:     "FirstBadPlaceholder",
			first:    "%b copy %x%e",
			numbered: "%b copy %n%e",
			wantErr:  true,
		},
		{
			name:     "NumberedNoNumber",
			first:    "%b copy%e",
			numbered: "%b copy%e",
			wantErr:  true,
		},
		{
			name:     "NumberedTwoNumbers",
			first:    "%b copy%e",
			numbered: "%b copy %n %n%e",
			wantErr:  true,
		},
		{
			name:     "NumberedNoText",
			first:    "%b copy%e",
			numbered: "%b%n%e",
			wantErr:  true,
		},
		{
			name:     "NumberedBadPlaceholder",
			first:    "%b copy%e",
			numbered: "%b copy %n %x%e",
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := compileDupRule(test.first, test.numbered)
			if (err != nil) != test.wantErr {
				t.Errorf("compileDupRule(%q, %q) error = %v, want error presence = %t", test.first, test.numbered, err, test.wantErr)
			}
		})
	}
}

func TestParseAndFormat(t *testing.T) {
	next := func(r dupRule, name string) string {
		stem, ext, seq := r.parse(name)
		return r.format(stem, ext, seq+1)
	}

	finder := mustCompileDupRule(t, "%b copy%e", "%b copy %n%e")
	nautilus := mustCompileDupRule(t, "%b (Copy)%e", "%b (Copy %n)%e")
	explorer := mustCompileDupRule(t, "%b - Copy%e", "%b - Copy (%n)%e")
	emacs := mustCompileDupRule(t, "%f.~1~", "%f.~%n~")

	tests := []struct {
		rule dupRule
		name string
		want string
	}{
		{finder, "file.txt", "file copy.txt"},
		{finder, "file copy.txt", "file copy 2.txt"},
		{finder, "file copy 1.txt", "file copy 2.txt"}, // edge case, matches original behaviour
		{finder, "file copy 2.txt", "file copy 3.txt"},

		{nautilus, "file.txt", "file (Copy).txt"},
		{nautilus, "file (Copy).txt", "file (Copy 2).txt"},
		{nautilus, "file (Copy 1).txt", "file (Copy 2).txt"}, // edge case, matches original behaviour
		{nautilus, "file (Copy 2).txt", "file (Copy 3).txt"},

		// TODO: optional, lazy evaluation to better match the original behaviour
		{explorer, "file.txt", "file - Copy.txt"},
		{explorer, "file - Copy.txt", "file - Copy (2).txt"},
		{explorer, "file - Copy (2).txt", "file - Copy (3).txt"},

		{emacs, "file.txt", "file.txt.~1~"},
		{emacs, "file.txt.~1~", "file.txt.~2~"},
		{emacs, "file.txt.~2~", "file.txt.~3~"},

		{finder, "file copy 12.txt", "file copy 13.txt"},
		{finder, "file copy 02.txt", "file copy 02 copy.txt"},
		{finder, "track01.mp3", "track01 copy.mp3"},
		{finder, ".bashrc", ".bashrc copy"},
		{finder, ".bashrc copy", ".bashrc copy 2"},
		{finder, "archive.tar.gz", "archive copy.tar.gz"},
		{finder, "archive copy 3.tar.gz", "archive copy 4.tar.gz"},
	}

	for _, test := range tests {
		if got := next(test.rule, test.name); got != test.want {
			t.Errorf("next(%q) = %q, want %q", test.name, got, test.want)
		}
	}
}

func mustCompileDupRule(t *testing.T, first, numbered string) dupRule {
	t.Helper()

	r, err := compileDupRule(first, numbered)
	if err != nil {
		t.Fatalf("compileDupRule(%q, %q): %v", first, numbered, err)
	}
	return r
}
