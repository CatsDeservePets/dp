package main

import (
	"errors"
	"strconv"
	"strings"
)

// A dupRule describes how duplicate file names are parsed and formatted.
type dupRule struct {
	stemBased bool   // false for %f..., true for %b...%e
	first     string // marker for sequence 1
	numPre    string // marker before %n
	numSuf    string // marker after %n
}

// compileDupRule compiles first and numbered into a [dupRule].
//
// first is used for sequence 1, numbered for sequence 2 and later.
// Both use %f for the full file name or %b and %e for the stem and extension.
// numbered must contain exactly one %n for the duplicate sequence.
func compileDupRule(first, numbered string) (dupRule, error) {
	cutFormat := func(format string) (body string, stemBased bool, ok bool) {
		if body, ok := strings.CutPrefix(format, "%f"); ok {
			return body, false, true
		}
		if body, ok := strings.CutPrefix(format, "%b"); ok {
			body, ok = strings.CutSuffix(body, "%e")
			return body, true, ok
		}
		return "", false, false
	}

	first, stemBased, ok := cutFormat(first)
	numbered, stemBased2, ok2 := cutFormat(numbered)
	if !ok || !ok2 || stemBased != stemBased2 {
		return dupRule{}, errors.New("formats must both match %f... or %b...%e")
	}

	if first == "" {
		return dupRule{}, errors.New("first format must add text")
	}
	if strings.Contains(first, "%n") {
		return dupRule{}, errors.New("first format must not contain %n")
	}
	if strings.Contains(first, "%") {
		return dupRule{}, errors.New("first format contains unsupported placeholder")
	}

	numPre, numSuf, ok := strings.Cut(numbered, "%n")
	if !ok || strings.Contains(numSuf, "%n") {
		return dupRule{}, errors.New("numbered format must contain exactly one %n")
	}
	// Avoid treating trailing digits in file names as duplicate sequences.
	if numPre == "" {
		return dupRule{}, errors.New("numbered format must add text before %n")
	}
	if strings.Contains(numPre, "%") || strings.Contains(numSuf, "%") {
		return dupRule{}, errors.New("numbered format contains unsupported placeholder")
	}

	return dupRule{
		stemBased: stemBased,
		first:     first,
		numPre:    numPre,
		numSuf:    numSuf,
	}, nil
}

// parse returns the original stem, extension, and duplicate sequence for name.
// If name is not recognised as a duplicate, seq is 0.
func (r dupRule) parse(name string) (stem, ext string, seq int) {
	stem = name
	if r.stemBased {
		stem, ext = splitName(name)
	}
	if orig, seq := r.parseNumbered(stem); seq > 0 {
		return orig, ext, seq
	}
	if orig, ok := strings.CutSuffix(stem, r.first); ok && orig != "" {
		return orig, ext, 1
	}
	return stem, ext, 0
}

func (r dupRule) parseNumbered(s string) (string, int) {
	s, ok := strings.CutSuffix(s, r.numSuf)
	if !ok {
		return "", 0
	}
	s, digits, ok := cutTrailingDigits(s)
	if !ok {
		return "", 0
	}
	orig, ok := strings.CutSuffix(s, r.numPre)
	if !ok || orig == "" {
		return "", 0
	}
	seq, err := strconv.Atoi(digits)
	// Numbered starts at sequence 2 when generated,
	// but may match sequence 1 when parsed.
	if err != nil || seq < 1 {
		return "", 0
	}
	return orig, seq
}

// format returns the canonical file name for sequence seq.
func (r dupRule) format(stem, ext string, seq int) string {
	switch seq {
	case 0:
		return stem + ext
	case 1:
		return stem + r.first + ext
	}
	return stem + r.numPre + strconv.Itoa(seq) + r.numSuf + ext
}

// cutTrailingDigits cuts a trailing digit suffix from s.
// It rejects suffixes that start with zero.
func cutTrailingDigits(s string) (before, digits string, ok bool) {
	i := len(s)
	for i > 0 && '0' <= s[i-1] && s[i-1] <= '9' {
		i--
	}
	if i == len(s) || s[i] == '0' {
		return "", "", false
	}
	return s[:i], s[i:], true
}

// splitName splits name into a stem and extension.
// Unlike [filepath.Ext], the extension is the suffix beginning at the first
// dot following any leading dots; it is empty if there is no such dot.
func splitName(name string) (stem, ext string) {
	i := 0
	for i < len(name) && name[i] == '.' {
		i++
	}
	for i < len(name) && name[i] != '.' {
		i++
	}
	return name[:i], name[i:]
}
