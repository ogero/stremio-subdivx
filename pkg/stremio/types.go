package stremio

// Manifest represents a Stremio addon manifest
type Manifest struct {
	ID            string        `json:"id"`
	Version       string        `json:"version"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Types         []string      `json:"types"`
	IDPrefixes    []string      `json:"idPrefixes"`
	Catalogs      []CatalogItem `json:"catalogs"`
	Resources     []string      `json:"resources"`
	BehaviorHints BehaviorHints `json:"behaviorHints"`
}

// CatalogItem represents a Stremio manifest catalog item
type CatalogItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// BehaviorHints represents Stremio manifest behavior hints
type BehaviorHints struct {
	Configurable          bool `json:"configurable"`
	ConfigurationRequired bool `json:"configurationRequired"`
}

// Subtitle represents a Stremio subtitle
type Subtitle struct {
	ID   string `json:"id"`
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

// Subtitles represents a collection of subtitle entries.
// Each entry contains details about the subtitle such as ID, language, and URL.
type Subtitles struct {
	Subtitles []Subtitle `json:"subtitles"`
}
