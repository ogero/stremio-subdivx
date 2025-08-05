package subdivx

import (
	"strings"
	"unicode"
)

// alphaNumericDistinctLowercaseWords processes a string, extracts alphanumeric words, converts them to lowercase, and returns a slice of unique words in the order they appear.
func alphaNumericDistinctLowercaseWords(s string) []string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') ||
			b == ' ' {
			result.WriteByte(byte(unicode.ToLower(rune(b))))
		} else {
			result.WriteByte(' ')
		}
	}
	m := make(map[string]struct{})
	fields := strings.Fields(result.String())
	j := 0
	for i := 0; i < len(fields); i++ {
		if _, ok := m[fields[i]]; !ok {
			m[fields[i]] = struct{}{}
			fields[j] = fields[i]
			j++
		}
	}
	return fields[:j]
}

// Score calculates a match score between a given string and the subtitle's description words.
func (f *Subtitle) Score(s string) int {
	var score int
	inputWords := alphaNumericDistinctLowercaseWords(s)
	for _, word := range inputWords {
		for _, descriptionWord := range f.DescriptionWords {
			if word == descriptionWord {
				score++
			}
		}
	}
	return score
}
