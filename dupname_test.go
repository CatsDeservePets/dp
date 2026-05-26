package main

import "testing"

func TestCompileDupRule(t *testing.T) {
	tests := []struct {
		first    string
		numbered string
		wantErr  bool
	}{
		{"%b copy%e", "%b copy %n%e", false}, // stem + extension
		{"%f.~1~", "%f.~%n~", false},         // full name

		{"%b copy%e", "%f copy %n", true},      // mixed styles
		{"%b copy", "%b copy %n%e", true},      // missing %e in first
		{"%b copy%e", "%b copy %n", true},      // missing %e in numbered
		{"%b%e", "%b copy %n%e", true},         // no text in first
		{"%b copy %n%e", "%b copy %n%e", true}, // %n in first
		{"%b copy %x%e", "%b copy %n%e", true}, // bad placeholder in first
		{"%b copy%e", "%b copy%e", true},       // no %n in numbered
		{"%b copy%e", "%b copy %n %n%e", true}, // two %n in numbered
		{"%b copy%e", "%b%n%e", true},          // no text before %n
		{"%b copy%e", "%b copy %n %x%e", true}, // bad placeholder in numbered
	}

	for _, test := range tests {
		_, err := compileDupRule(test.first, test.numbered)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Errorf("compileDupRule(%q, %q) got error %v; want error %v", test.first, test.numbered, gotErr, test.wantErr)
		}
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
			t.Errorf("next(%q) = %q; want %q", test.name, got, test.want)
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
