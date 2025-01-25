package model

// For getting place id
type GoogleAPIFindPlaces struct {
	Candidates []struct {
		PlaceID string `json:"place_id"`
	} `json:"candidates"`
	Status string `json:"status"`
}

// For getting reviews
type GoogleAPIPlacesDetails struct {
	HTMLAttributions []string `json:"html_attributions"`
	Result           struct {
		Reviews []struct {
			AuthorName              string `json:"author_name"`
			AuthorURL               string `json:"author_url"`
			Language                string `json:"language"`
			OriginalLanguage        string `json:"original_language"`
			ProfilePhotoURL         string `json:"profile_photo_url"`
			Rating                  int    `json:"rating"`
			RelativeTimeDescription string `json:"relative_time_description"`
			Text                    string `json:"text"`
			Time                    int    `json:"time"`
			Translated              bool   `json:"translated"`
		} `json:"reviews"`
	} `json:"result"`
	Status string `json:"status"`
}
