package tagging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"vibe/api"
	mAPI "vibe/model/api"
	"vibe/store"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var APP_ENV string
var OPEN_API_KEY string
var GOOGLE_API_KEY string
var LOCATIONS_API_PORT string
var LOCATIONS_API_URL string
var NUM_WORKERS int
var ENV_PROD = "prod"

var prompt_raw = "You are a sentiment analysis model that will use the following list of emotional tags (with helpful context) to classify text. Only respond with the top associated tag by itself."
var tags_raw = "chill (cool, relaxed, mellow), wild (loud, energetic, exciting)"

var jobs chan Job
var jobIDs map[string]struct{}
var ctx context.Context
var cancel context.CancelFunc

// Response -> response for the util scope
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

// Job details
type Job struct {
	ID            string // Technically the loc_hash
	Lat           string
	Lon           string
	Location_name string
}

var client = &http.Client{
	Timeout: time.Second * 10,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     time.Second * 30,
	},
}

// Runs on startup
func Setup() {
	LoadEnvValues()
	SetupJobChannel()
}

// Loads in the needed enviroment variables
func LoadEnvValues() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env: ", err)
	}

	APP_ENV = os.Getenv("APP_ENV") // This is error checking in main.go

	OPEN_API_KEY = os.Getenv("OPEN_API_KEY")
	if OPEN_API_KEY == "" {
		log.Fatal("OPEN_API_KEY is not set")
	}

	GOOGLE_API_KEY = os.Getenv("GOOGLE_API_KEY")
	if GOOGLE_API_KEY == "" {
		log.Fatal("GOOGLE_API_KEY is not set")
	}

	LOCATIONS_API_PORT = os.Getenv("LOCATIONS_API_PORT")
	if LOCATIONS_API_PORT == "" {
		log.Fatal("LOCATIONS_API_PORT is not set")
	}

	LOCATIONS_API_URL = os.Getenv("LOCATIONS_API_URL")
	if LOCATIONS_API_URL == "" {
		log.Fatal("LOCATIONS_API_URL is not set")
	}

	num_workers, ok := os.LookupEnv("NUM_WORKERS")
	if !ok {
		log.Warning("No NUM_WORKERS specified, defaulting to 1")
		NUM_WORKERS = 1
	} else {
		i, err := strconv.Atoi(num_workers)
		if err != nil {
			log.Warning("Couldn't convert NUM_WORKERS to int, defaulting to 1")
			NUM_WORKERS = 1
		}
		NUM_WORKERS = i
	}
}

// Setups up the job channel to process predictions
func SetupJobChannel() {
	// Setup context for force close workers if needed
	ctx, cancel = context.WithCancel(context.Background())

	// Initialize jobs channel
	jobs = make(chan Job)
	jobIDs = make(map[string]struct{})
	log.Info("Setup job channel")

	// Setup wait group and start go routines (workers)
	var wg sync.WaitGroup
	wg.Add(NUM_WORKERS)
	log.Trace("Setting up the workers for job channel")
	for i := 0; i < NUM_WORKERS; i++ {
		go worker(i, &wg)
		log.Trace("    Worker ", i, " setup")
	}
	log.Info("All workers setup for job channel")
}

// Proccesses jobs in the channel
func worker(worker_id int, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Trace("W", worker_id, ": Stopping...")
			return
		case job, ok := <-jobs:
			if !ok {
				log.Trace("W", worker_id, ": Channel closed")
				return
			}
			log.Trace("W", worker_id, ": Processing job ", job.ID, " | ", job.Location_name)

			log.Info("W", worker_id, ": Getting predictions")
			predictions, err := GetPreditionsFromAPI(worker_id, prompt_raw, tags_raw, job.Lat, job.Lon, job.Location_name)
			if err != nil {
				log.Error("W", worker_id, ": There was an error getting tags from API: ", err)
				delete(jobIDs, job.ID)
				log.Trace("W", worker_id, ": Removed job ", job.ID, " due to error")
			}

			// Store tags into DB
			err = StoreTagsInDB(worker_id, predictions, job.Lat, job.Lon, job.Location_name, job.ID)
			if err != nil {
				log.Error("W", worker_id, ": There was an error storing tags into DB: ", err)
				delete(jobIDs, job.ID)
				log.Trace("W", worker_id, ": Removed job ", job.ID, " due to error")
			}

			delete(jobIDs, job.ID) // Remove job from current job list
			log.Trace("W", worker_id, ": Removed job ", job.ID)
		}
	}
}

// Adds a prediction job to the channel
func AddPredictionJob(job Job) {
	// Check if job isnt already in queue
	if _, ok := jobIDs[job.ID]; !ok {
		jobIDs[job.ID] = struct{}{}
		jobs <- job
		log.Info("Added job to queue: ", job.ID, " | ", job.Location_name)
	}
}

// Generates a location hash based on name, lat, and long
func GenerateLocationHashString(name string, lat string, long string) string {
	log.WithFields(log.Fields{
		"name": name,
		"lat":  lat,
		"long": long,
	}).Trace("Generating location hash")

	hash32 := fnv.New32a()
	hash32.Write([]byte(name + lat + long))
	hashString := strconv.Itoa(int(hash32.Sum32()))

	log.WithFields(log.Fields{
		"hashString": hashString,
	}).Trace("Location hash generated")

	return hashString
}

// Limits the string if input is longer then length. Wont cut words in half.
func LimitString(input string, maxLength int) string {
	if len(input) > maxLength {
		input = input[:maxLength]
		input = strings.TrimSuffix(input, " ")
	}
	return input
}

// Grabs tags from db, if they dont exists, adds a new job to get the predictions
func GetTags(w http.ResponseWriter, r *http.Request) {
	log.Info("Starting get tag logic")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// Read in form values
	location_name := r.FormValue("location_name")
	lat_raw := r.FormValue("lat")
	lon_raw := r.FormValue("lon")
	log.WithFields(log.Fields{
		"location_name": location_name,
		"lat":           lat_raw,
		"long":          lon_raw,
	}).Trace("Raw form values")
	log.Info("Processed form values")

	// Error check each form value to make sure they exist
	if location_name == "" {
		response.Message = "Missing required query parameter: location_name"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	if lat_raw == "" {
		response.Message = "Missing required query parameter: lat"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	if lon_raw == "" {
		response.Message = "Missing required query parameter: lon"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// Convert lat and lon to floats
	lat_float, err := strconv.ParseFloat(lat_raw, 64)
	if err != nil {
		response.Message = "Invalid lat query parameter: " + err.Error()
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	lon_float, err := strconv.ParseFloat(lon_raw, 64)
	if err != nil {
		response.Message = "Invalid lon query parameter: " + err.Error()
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// Expand float to have 9 decimal points as is the the db table criteria(may change)
	lat_formatted := fmt.Sprintf("%.9f", lat_float)
	lon_formatted := fmt.Sprintf("%.9f", lon_float)
	location_hash := GenerateLocationHashString(location_name, lat_formatted, lon_formatted)

	// Get tags for location hash
	predictions, err := GetTagsFromDB(location_hash)

	if err != nil {
		response.Message = "There was an error getting tags from DB"
		log.Error(response.Message, ": ", err)
		api.Respond(w, response, http.StatusInternalServerError)
		return
	}

	// Get predicitons if DB was empty
	if predictions == "" {
		AddPredictionJob(Job{
			ID:            location_hash,
			Lat:           lat_formatted,
			Lon:           lon_formatted,
			Location_name: location_name})
		predictions = "Retrying"
	}

	data := mAPI.TaggingResponse{
		Tags:        predictions,
		Latitude:    lat_formatted,
		Longitude:   lon_formatted,
		Used_stored: false,
	}

	log.Trace("Finished ")
	api.RespondOK(w, data) // Return tags
}

// Grabs multiple locations around a center point and gets their tags, adding
// jobs if the predicitions didnt exist
func GetTagsCenterPos(w http.ResponseWriter, r *http.Request) {
	log.Info("Starting get tag logic")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// Read in form values
	lat_raw := r.FormValue("lat")
	lon_raw := r.FormValue("lon")
	filter := r.FormValue("filter") // Optional so can be blank
	log.WithFields(log.Fields{
		"lat":    lat_raw,
		"long":   lon_raw,
		"filter": filter,
	}).Trace("Raw form values")
	log.Info("Processed form values")

	// Error check each form value to make sure they exist
	if lat_raw == "" {
		response.Message = "Missing required query parameter: lat"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	if lon_raw == "" {
		response.Message = "Missing required query parameter: lon"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// Convert lat and lon to floats
	lat_float, err := strconv.ParseFloat(lat_raw, 64)
	if err != nil {
		response.Message = "Invalid lat query parameter: " + err.Error()
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	lon_float, err := strconv.ParseFloat(lon_raw, 64)
	if err != nil {
		response.Message = "Invalid lon query parameter: " + err.Error()
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// Expand float to have 9 decimal points as is the the db table criteria(may change)
	lat_formatted := fmt.Sprintf("%.9f", lat_float)
	lon_formatted := fmt.Sprintf("%.9f", lon_float)

	// Get locations from Locations API
	locations, err := GetLocations(lat_formatted, lon_formatted)
	if err != nil {
		response.Message = "Error while getting location from Locations API"
		log.Error(response.Message, ": ", err)
		api.Respond(w, response, http.StatusInternalServerError)
		return
	}

	data := mAPI.TaggingMultipleResponse{}
	// For each location, do get tags. (Will make sure the tags are in db)
	for i, location := range locations {
		//Skip first as it is geocoded result
		if i == 0 {

			data = append(data, struct {
				ReverseGeocodeResult string  `json:"reverseGeocodeResult,omitempty"`
				Name                 string  `json:"name,omitempty"`
				Website              string  `json:"website,omitempty"`
				Lon                  float64 `json:"lon,omitempty"`
				Lat                  float64 `json:"lat,omitempty"`
				Tags                 string  `json:"tags,omitempty"`
			}{
				ReverseGeocodeResult: location.ReverseGeocodeResult,
			})
			continue
		}

		// Convert lat and lon to floats
		if location.Name == "" {
			response.Message = "Invalid location name"
			log.Warning(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}

		// Expand float to have 9 decimal points as is the the db table criteria(may change)
		lat_formatted := fmt.Sprintf("%.9f", location.Lat)
		lon_formatted := fmt.Sprintf("%.9f", location.Lon)
		location_hash := GenerateLocationHashString(location.Name, lat_formatted, lon_formatted)

		// Get tags for location hash
		predictions, err := GetTagsFromDB(location_hash)
		if err != nil {
			response.Message = "There was an error getting tags from DB"
			log.Error(response.Message, ": ", err)
			api.Respond(w, response, http.StatusInternalServerError)
			return
		}

		// Get predicitons if DB was empty
		if predictions == "" {
			AddPredictionJob(Job{
				ID:            location_hash,
				Lat:           lat_formatted,
				Lon:           lon_formatted,
				Location_name: location.Name})
			predictions = "Retrying"
		}

		if filter != "" {
			if predictions == filter {
				data = append(data, struct {
					ReverseGeocodeResult string  `json:"reverseGeocodeResult,omitempty"`
					Name                 string  `json:"name,omitempty"`
					Website              string  `json:"website,omitempty"`
					Lon                  float64 `json:"lon,omitempty"`
					Lat                  float64 `json:"lat,omitempty"`
					Tags                 string  `json:"tags,omitempty"`
				}{
					Name:    location.Name,
					Website: location.Website,
					Lon:     location.Lon,
					Lat:     location.Lat,
					Tags:    predictions,
				})
			}
		} else {
			data = append(data, struct {
				ReverseGeocodeResult string  `json:"reverseGeocodeResult,omitempty"`
				Name                 string  `json:"name,omitempty"`
				Website              string  `json:"website,omitempty"`
				Lon                  float64 `json:"lon,omitempty"`
				Lat                  float64 `json:"lat,omitempty"`
				Tags                 string  `json:"tags,omitempty"`
			}{
				Name:    location.Name,
				Website: location.Website,
				Lon:     location.Lon,
				Lat:     location.Lat,
				Tags:    predictions,
			})
		}
	}

	log.Trace("Finished ")
	api.RespondOK(w, data)
}

// Gets locations around a point via our locations service
func GetLocations(lat string, lon string) (mAPI.LocationsAPIGetLocationsResponse, error) {
	log.Info("Getting locations from Locations API")

	// Setup Locations API POST body
	data := mAPI.LocationsAPIGetLocationsRequest{
		Lat: lat,
		Lon: lon,
	}
	body, err := json.Marshal(data)
	if err != nil {
		log.Panic("Unable to marshal into JSON: ", err)
		return nil, err
	}
	postBody := bytes.NewBuffer(body)

	// Create completions request for OpenAI
	request := &http.Request{}
	if APP_ENV == ENV_PROD {
		// request, err = http.NewRequest(http.MethodPost, "http://127.0.0.1:"+LOCATIONS_API_PORT, postBody)
		request, err = http.NewRequest(http.MethodPost, LOCATIONS_API_URL, postBody)
	} else {
		request, err = http.NewRequest(http.MethodPost, LOCATIONS_API_URL, postBody)
	}
	if err != nil {
		log.Error("Unable to create request: ", err)
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Add("Connection", "keep-alive")
	log.Info("Created completions request for Locations API")

	// Send Request
	response, err := client.Do(request)
	if err != nil {
		log.Error("Unable to send request: ", err)
		return nil, err
	}
	log.Info("Sent request for locations")
	defer response.Body.Close()

	// Read Response
	responseBody, err := ioutil.ReadAll(response.Body) // Converts response into byte array
	if err != nil {
		log.Error("Error while reading the response bytes: ", err)
		return nil, err
	}

	// Report error if API response is not 200 / OK
	if response.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": response.StatusCode,
			"json":   string(responseBody),
		}).Error("Response is not Status OK")
		return nil, fmt.Errorf("Response not OK")
	}

	// Unmarshal reponse into go struct
	var result mAPI.LocationsAPIGetLocationsResponse
	err = json.Unmarshal([]byte(responseBody), &result)
	if err != nil {
		log.Error("Can not unmarshal JSON: ", err)
		return nil, err
	}

	return result, nil
}

// Gets tags from the db
func GetTagsFromDB(location_hash string) (string, error) {
	log.Info("Getting tags from DB")

	// Dont attempt to get tags from DB when in debug
	if APP_ENV != ENV_PROD {
		log.Warning("Skipping get from DB, not in prod")
		return "", nil
	}

	// Prepare sql for select
	stmt, err := store.DB.Prepare("SELECT tag FROM location_tags WHERE location_hash = ?")
	if err != nil {
		log.Error("Failed to prepare statement: ", err)
		return "", err
	}
	defer stmt.Close()

	// Execute sql
	rows, err := stmt.Query(location_hash)
	if err != nil {
		log.Error("SELECT FROM location_tags failed: ", err)
		return "", err
	}
	defer rows.Close()

	// Loop through rows, using Scan to append tag to selected_tags string
	log.Trace("Grabbing tags from selected rows")
	selected_tags := ""
	for rows.Next() {
		var tag string
		err := rows.Scan(&tag)
		if err != nil {
			log.Error("Bad DB query: ", err)
			return "", err
		}

		log.Trace("Tag: ", tag)

		if selected_tags == "" {
			selected_tags += (tag)
		} else {
			selected_tags += ("," + tag) // CSV format
		}
	}
	log.Info("Tags for ", location_hash, ": ", selected_tags)
	return selected_tags, nil
}

// Gets the place id from googles places api
func GetPlaceID(worker_id int, lat string, lon string, loc_name string) (string, error) {
	log.Info("W", worker_id, ": Getting PlaceID for location: ", loc_name)

	// Create findplace request for Google
	req, err := http.NewRequest("GET", "https://maps.googleapis.com/maps/api/place/findplacefromtext/json?", nil)
	if err != nil {
		log.Error("W", worker_id, ": Unable to create request: ", err)
		return "", err
	}
	req.Header.Add("Connection", "keep-alive")

	q := req.URL.Query()
	q.Add("field", "place_id")
	q.Add("input", loc_name)
	q.Add("inputtype", "textquery")
	q.Add("locationbias", ("circle:1@" + lat + "," + lon))
	q.Add("key", GOOGLE_API_KEY)
	req.URL.RawQuery = q.Encode()
	log.Info("W", worker_id, ": Created findplace request for Google")

	// Send Request
	response, err := client.Do(req)
	if err != nil {
		log.Error("W", worker_id, ": Unable to send request: ", err)
		return "", err
	}
	log.Info("W", worker_id, ": Sent request for PlaceID")
	defer response.Body.Close()

	// Read Response
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("W", worker_id, ": Error while reading the response bytes: ", err)
		return "", err
	}

	// Report error if API response is not 200 / OK
	if response.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": response.StatusCode,
			"json":   string(responseBody),
		}).Error("W", worker_id, ": Response is not Status OK")
		return "", fmt.Errorf("Response not OK")
	}

	var result mAPI.GoogleAPIFindPlaces
	err = json.Unmarshal([]byte(responseBody), &result)
	if err != nil {
		log.Error("W", worker_id, ": Can not unmarshal JSON: ", err)
		return "", err
	}

	if len(result.Candidates) > 0 {
		log.Info("W", worker_id, ": PlacesID for ", loc_name, " is: ", result.Candidates[0].PlaceID)
		return result.Candidates[0].PlaceID, nil
	} else {
		log.Error("W", worker_id, ": No locations returned from FindPlaces API")
		return "", fmt.Errorf("No locations returned from FindPlaces API")
	}
}

// Gets the reviews based on the places api
func GetReviewsByPlacesID(worker_id int, place_id string) (string, error) {
	log.Info("W", worker_id, ": Getting reviews for PlaceID: ", place_id)

	// Create place details request for Google
	req, err := http.NewRequest("GET", "https://maps.googleapis.com/maps/api/place/details/json?", nil)
	if err != nil {
		log.Error("W", worker_id, ": Unable to create request: ", err)
		return "", err
	}
	req.Header.Add("Connection", "keep-alive")

	q := req.URL.Query()
	q.Add("field", "reviews")
	q.Add("placeid", place_id)
	q.Add("key", GOOGLE_API_KEY)
	req.URL.RawQuery = q.Encode()
	log.Info("W", worker_id, ": Created place details request for Google")

	// Send request
	response, err := client.Do(req)
	if err != nil {
		log.Error("W", worker_id, ": Unable to send request: ", err)
		return "", err
	}
	log.Info("W", worker_id, ": Sent request for reviews")
	defer response.Body.Close()

	// Read Response
	responseBody, err := ioutil.ReadAll(response.Body) // Makes request a byte array
	if err != nil {
		log.Error("W", worker_id, ": Error while reading the response bytes: ", err)
		return "", err
	}

	// Report error if API response is not 200 / OK
	if response.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": response.StatusCode,
			"json":   string(responseBody),
		}).Error("W", worker_id, ": Response is not Status OK")
		return "", fmt.Errorf("Response not OK")
	}

	// Convert json to go struct
	var result mAPI.GoogleAPIPlacesDetails
	err = json.Unmarshal([]byte(responseBody), &result)
	if err != nil {
		log.Error("W", worker_id, ": Can not unmarshal JSON: ", err)
		return "", err
	}

	log.Info("W", worker_id, ": Reviews collected")

	// Postprocess the reviews and concat them
	review_data := ""
	for _, review := range result.Result.Reviews {
		// Limit character space and replace whitespace characters with spaces
		tmp := regexp.MustCompile(`[^a-zA-Z0-9,!()?.\s]+`).ReplaceAllString(review.Text, "")
		tmp = regexp.MustCompile(`\s\s+`).ReplaceAllString(tmp, " ")
		tmp = LimitString(tmp, 300) // Limit to ~300 chars per review

		if review_data == "" {
			review_data += tmp
		} else {
			// \n\n good for seperating reviews (low logits value in AI model)
			review_data += " " + tmp
		}
	}

	return review_data, nil
}

// Gets the review data
func GetReviewData(worker_id int, lat string, lon string, loc_name string) (string, error) {
	// Get place id for reviews via Google API
	place_id, err := GetPlaceID(worker_id, lat, lon, loc_name)
	if err != nil {
		log.Error("W", worker_id, ": Unable to get place id: ", err)
		return "", err
	}

	// Get reviews via the place id via Google API
	text_data, err := GetReviewsByPlacesID(worker_id, place_id)
	if err != nil {
		log.Error("W", worker_id, ": Unable to get reviews by place id: ", err)
		return "", err
	}

	log.Info("W", worker_id, ": Processed reviews")
	log.Trace("W", worker_id, ": Cleaned Reviews: ", text_data)
	return text_data, nil
}

// Gets the tag predictions from OpenAI api
func GetPreditionsFromAPI(worker_id int, prompt string, tags string, lat string, lon string, loc_name string) (string, error) {
	// Only do outgoing requests if in prod
	if APP_ENV != ENV_PROD {
		log.Warning("W ", worker_id, ": Returning fake values, not in prod")
		return "wild", nil
	}

	text, err := GetReviewData(worker_id, lat, lon, loc_name) // Gets the review data for predictions
	if err != nil {
		log.Error("W", worker_id, ": Couldnt get review data ", err)
		return "", err
	}

	// Create prompt for chat completeions
	messages_data := []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{{
		Role:    "system",
		Content: prompt + "\nTags: " + tags,
	}, {
		Role:    "user",
		Content: text,
	}}

	log.Trace("W", worker_id, ": Prompt data: ", messages_data)

	// Setup Open AI POST body
	data := mAPI.OpenAIChatCompletionsRequest{
		Model:      "gpt-3.5-turbo",
		Top_p:      0.1, // Only consider top 10% of possible generations
		Max_tokens: 10,
		Messages:   messages_data,
	}

	body, err := json.Marshal(data)
	if err != nil {
		log.Panic("W", worker_id, ": Unable to marshal into JSON: ", err)
		return "", err
	}
	postBody := bytes.NewBuffer(body)

	// Create chat completions request for OpenAI
	request, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", postBody)
	if err != nil {
		log.Error("W", worker_id, ": Unable to create request: ", err)
		return "", err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Add("Authorization", "Bearer "+OPEN_API_KEY)
	request.Header.Add("Connection", "keep-alive")
	log.Info("W", worker_id, ": Created chat completions request for OpenAI")

	// Send Request
	response, err := client.Do(request)
	if err != nil {
		log.Error("W", worker_id, ": Unable to send request: ", err)
		return "", err
	}
	log.Info("W", worker_id, ": Sent request for predictions")
	defer response.Body.Close()

	// Read Response
	responseBody, err := ioutil.ReadAll(response.Body) // Converts response into byte array
	if err != nil {
		log.Error("W", worker_id, ": Error while reading the response bytes: ", err)
		return "", err
	}

	// Report error if API response is not 200 / OK
	if response.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": response.StatusCode,
			"json":   string(responseBody),
		}).Error("W", worker_id, ": Response is not Status OK")
		return "", fmt.Errorf("Response not OK")
	}

	// Unmarshal reponse into go struct
	var result mAPI.OpenAIChatCompletionsResponse
	err = json.Unmarshal([]byte(responseBody), &result)
	if err != nil {
		log.Error("W", worker_id, ": Can not unmarshal JSON: ", err)
		return "", err
	}

	// Only return if results were completed
	result_tags := ""
	if result.Choices[0].FinishReason == "stop" {
		// TODO dont just trust the model output, add some filtering logic here
		result_tags = result.Choices[0].Message.Content // Extract tags from struct
	} else {
		result_tags = "error"
		log.Warning("W", worker_id, ": OpenAI respons finish reason is not stop, check logs: ", result.Choices[0].FinishReason, " : ", result.Choices[0].Message.Content)
	}
	// Postprocess
	// TODO CLEAN INPUT MORE - Fix regex to shrink multiple , into one
	result_tags = regexp.MustCompile(`[^a-zA-Z,]+|(\s+)`).ReplaceAllString(result_tags, "") // Limit character space
	result_tags = strings.ToLower(result_tags)                                              // Convert to lowercase
	log.Info("W", worker_id, ": Processed predictions")
	log.Trace("W", worker_id, ": Predictions: ", result_tags)
	return result_tags, nil
}

// Store tags in the db
func StoreTagsInDB(worker_id int, tags string, lat string, lon string, location_name string, location_hash string) error {
	// Only store tags in DB if in prod
	if APP_ENV != ENV_PROD {
		log.Warning("W", worker_id, ": Skipping store, not in prod")
		return nil
	}

	log.Info("W", worker_id, ": Storing tags in DB: ", tags)

	// Insert each tag as a row with location_hash
	for _, tag := range strings.Split(tags, ",") {

		// REFACTOR THIS
		// Insert into locations handling (if not in table already)
		stmt, err := store.DB.Prepare("INSERT INTO locations (location_hash, location_name, lat, lon) SELECT ?, ?, ?, ? WHERE NOT EXISTS (SELECT * FROM locations WHERE location_hash = ?)")
		if err != nil {
			log.Error("W", worker_id, ": Failed to prepare statement: ", err)
			return err
		}
		defer stmt.Close()

		// Insert into DB
		result, err := stmt.Exec(location_hash, location_name, lat, lon, location_hash)
		if err != nil {
			log.Error("W", worker_id, ": INSERT INTO locations failed: ", err)
			return err
		}

		rows, err := result.RowsAffected() // Grab rows effected for logging
		if err != nil {
			log.Error("W", worker_id, ": Error getting rows effected for INSERT: ", err)
			return err
		}

		if rows != 0 {
			log.Info("W", worker_id, ": Row added to locations")
		}

		// Insert into locations_tags handling
		// Prepare sql insertion
		stmt, err = store.DB.Prepare("INSERT INTO location_tags(location_hash, tag) value(?,?)")
		if err != nil {
			log.Error("W", worker_id, ": Failed to prepare statement: ", err)
			return err
		}
		defer stmt.Close()

		// Insert into DB
		result, err = stmt.Exec(location_hash, tag)
		if err != nil {
			log.Error("W", worker_id, ": INSERT INTO location_tags failed: ", err)
			return err
		}

		rows, err = result.RowsAffected() // Grab rows effected for logging
		if err != nil {
			log.Error("W", worker_id, ": Error getting rows effected for INSERT: ", err)
			return err
		}

		if rows != 0 {
			log.Info("W", worker_id, ": Row added to location_tags")
		}
	}
	return nil
}

// Cleans up any captured resources and frees them
func Cleanup() {
	log.Info("Cleaning up tagging")
	cancel()    // Stop all workers
	close(jobs) // Close channel
	client.CloseIdleConnections()
}
