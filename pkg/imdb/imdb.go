package imdb

import "context"

// Title represents a movie or series title with its name and release year.
type Title struct {
	Name string
	Year int
}

// IMDB defines the methods to interact with the IMDB service.
type IMDB interface {
	// GetTitle gets a Title by its ID.
	GetTitle(ctx context.Context, imdbID string) (*Title, error)
}
