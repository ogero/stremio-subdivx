package common

import (
	"errors"
	"regexp"
	"strconv"
)

var imdbTitleIDRE = regexp.MustCompile(`^tt\d+$`)

// ValidateIMDBTitleID checks if the given IMDB title ID is valid.
// It ensures the title starts with 'tt' followed by a numeric suffix.
func ValidateIMDBTitleID(ID string) error {

	if !imdbTitleIDRE.MatchString(ID) {
		return errors.New("invalid IMDB title")
	}

	return nil
}

// ValidateSubtitleType checks if the subtitle type is valid.
// It expects 'movie' and 'series' as valid types.
func ValidateSubtitleType(t string) error {
	if t != "movie" && t != "series" {
		return errors.New("invalid subtitle type, only movie and series are supported")
	}

	return nil
}

// ValidateSubdivxSubtitleID checks if the given Subdivx subtitle ID is valid.
// It ensures the ID is a numeric value.
func ValidateSubdivxSubtitleID(id string) error {
	v, err := strconv.Atoi(id)
	if err != nil {
		return errors.New("invalid Subdivx subtitle id, not a number")
	}

	if v <= 0 {
		return errors.New("invalid Subdivx subtitle id, less than or equal to 0")
	}

	return nil
}
