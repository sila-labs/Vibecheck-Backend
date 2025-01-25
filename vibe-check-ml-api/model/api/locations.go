package model

type LocationsAPIGetLocationsRequest struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

type LocationsAPIGetLocationsResponse []struct {
	ReverseGeocodeResult string  `json:"reverseGeocodeResult,omitempty"`
	Name                 string  `json:"name,omitempty"`
	Website              string  `json:"website,omitempty"`
	Lon                  float64 `json:"lon,omitempty"`
	Lat                  float64 `json:"lat,omitempty"`
}
