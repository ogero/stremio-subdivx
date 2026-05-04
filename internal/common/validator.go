package common

import (
	"errors"
	"regexp"
)

var imdbTitleIDRE = regexp.MustCompile(`^tt\d+$`)
var subxSubtitleIDRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

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

// ValidateSubXSubtitleID checks if the given SubX subtitle ID is valid.
func ValidateSubXSubtitleID(id string) error {
	if !subxSubtitleIDRE.MatchString(id) {
		return errors.New("invalid SubX subtitle id")
	}

	return nil
}
