package model

// For return of tags
type TaggingResponse struct {
	Tags        string `json:"tags"`
	Latitude    string `json:"latitude"`
	Longitude   string `json:"longitude"`
	Used_stored bool   `json:"used_stored"`
}

// For return of tagging w/ location filter
type TaggingMultipleResponse []struct {
	ReverseGeocodeResult string  `json:"reverseGeocodeResult,omitempty"`
	Name                 string  `json:"name,omitempty"`
	Website              string  `json:"website,omitempty"`
	Lon                  float64 `json:"lon,omitempty"`
	Lat                  float64 `json:"lat,omitempty"`
	Tags                 string  `json:"tags,omitempty"`
}
