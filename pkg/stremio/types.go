package stremio

// Manifest represents a Stremio addon manifest
type Manifest struct {
	ID          string        `json:"id"`
	Version     string        `json:"version"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Types       []string      `json:"types"`
	IDPrefixes  []string      `json:"idPrefixes"`
	Catalogs    []CatalogItem `json:"catalogs"`
	Resources   []string      `json:"resources"`
}

// CatalogItem represents a Stremio manifest catalog item
type CatalogItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Subtitle represents a Stremio subtitle
type Subtitle struct {
	ID   string `json:"id"`
	Lang string `json:"lang"`
	URL  string `json:"url"`
}
