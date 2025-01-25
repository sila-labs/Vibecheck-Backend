package twilio

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"vibe/api"
	mDB "vibe/model/db"
	"vibe/store"
)

type SendData struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type ErrorMessage struct {
	Message string `json:"errorMessage"`
}

func PasswordRecoveryVerifyPhoneNumber(w http.ResponseWriter, r *http.Request) {

	SERVICE_SID := os.Getenv("TWILIO_SERVICE_SID")
	ACCOUNT_SID := os.Getenv("TWILIO_ACCOUNT_SID")
	AUTH_TOKEN := os.Getenv("TWILIO_AUTH_TOKEN")

	client := &http.Client{}
	apiUrl := "https://verify.twilio.com"
	resource := "/v2/Services/"
	channel := "sms"
	creds := &SendData{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}

	errorMessage := &ErrorMessage{}

	// Check for empty values
	if string(creds.Phone) == "" {
		log.Println("Empty field(s)")
		errorMessage.Message = "Empty field"
		api.Respond(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Query db for existing user
	result := store.DB.QueryRow("SELECT user_id FROM users WHERE phone=?", string(creds.Phone))
	if err != nil {
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}
	storedCreds := &mDB.User{}
	if err := result.Scan(&storedCreds.UserId); err == nil {
		data := url.Values{}
		fmt.Println(creds.Phone)
		data.Set("To", creds.Phone)
		data.Set("Channel", channel)
		data.Set("FriendlyName", "VibeCheck")
		data.Set("serviceSid", SERVICE_SID)
		u, _ := url.ParseRequestURI(apiUrl)
		u.Path = resource + SERVICE_SID + "/Verifications"
		urlStr := u.String()
		// Define request
		req, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(ACCOUNT_SID, AUTH_TOKEN)
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("Error making access token request")
		}
		requestDump, err := httputil.DumpRequest(req, true)
		fmt.Println(string(requestDump))
		if err != nil {
			fmt.Println(err.Error())
		}
		// Make request
		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer res.Body.Close()
		// Read reponse
		body, err := ioutil.ReadAll(res.Body)
		fmt.Println(string(body))
		if err != nil {
			fmt.Println(err.Error())
		}
		api.RespondRaw(w, body, http.StatusOK)

	} else if err == sql.ErrNoRows {
		log.Println("Phone number not found")
		errorMessage.Message = "Phone number not associated with any account"
		api.Respond(w, errorMessage, http.StatusConflict)
		return

	} else {
		log.Println("Bad DB query")
		log.Println(err)
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}
}

func VerifyPhoneNumber(w http.ResponseWriter, r *http.Request) {

	SERVICE_SID := os.Getenv("TWILIO_SERVICE_SID")
	ACCOUNT_SID := os.Getenv("TWILIO_ACCOUNT_SID")
	AUTH_TOKEN := os.Getenv("TWILIO_AUTH_TOKEN")

	client := &http.Client{}
	apiUrl := "https://verify.twilio.com"
	resource := "/v2/Services/"
	channel := "sms"
	creds := &SendData{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}

	errorMessage := &ErrorMessage{}

	// Check for empty values
	if string(creds.Phone) == "" {
		log.Println("Empty field(s)")
		errorMessage.Message = "Empty field"
		api.Respond(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Query db for existing user
	result := store.DB.QueryRow("SELECT user_id FROM users WHERE phone=?", string(creds.Phone))
	if err != nil {
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}
	storedCreds := &mDB.User{}
	if err := result.Scan(&storedCreds.UserId); err == nil {
		log.Println("Phone number already used")
		errorMessage.Message = "This number is used."
		api.Respond(w, errorMessage, http.StatusConflict)
		return
	} else if err == sql.ErrNoRows {
		data := url.Values{}
		fmt.Println(creds.Phone)
		data.Set("To", creds.Phone)
		data.Set("Channel", channel)
		data.Set("FriendlyName", "VibeCheck")
		data.Set("serviceSid", SERVICE_SID)
		u, _ := url.ParseRequestURI(apiUrl)
		u.Path = resource + SERVICE_SID + "/Verifications"
		urlStr := u.String()
		// Define request
		req, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(ACCOUNT_SID, AUTH_TOKEN)
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("Error making access token request")
		}
		requestDump, err := httputil.DumpRequest(req, true)
		fmt.Println(string(requestDump))
		if err != nil {
			fmt.Println(err.Error())
		}
		// Make request
		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer res.Body.Close()
		// Read reponse
		body, err := ioutil.ReadAll(res.Body)
		fmt.Println(string(body))
		if err != nil {
			fmt.Println(err.Error())
		}
		api.RespondRaw(w, body, http.StatusOK)

	} else {
		log.Println("Bad DB query")
		log.Println(err)
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}
}

func VerifyCode(w http.ResponseWriter, r *http.Request) {
	SERVICE_SID := os.Getenv("TWILIO_SERVICE_SID")
	ACCOUNT_SID := os.Getenv("TWILIO_ACCOUNT_SID")
	AUTH_TOKEN := os.Getenv("TWILIO_AUTH_TOKEN")
	client := &http.Client{}
	apiUrl := "https://verify.twilio.com"
	resource := "/v2/Services/"
	creds := &SendData{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}
	data := url.Values{}
	fmt.Println(creds.Phone)
	data.Set("To", creds.Phone)
	data.Set("Code", creds.Code)
	data.Set("FriendlyName", "VibeCheck")
	data.Set("serviceSid", SERVICE_SID)
	u, _ := url.ParseRequestURI(apiUrl)
	u.Path = resource + SERVICE_SID + "/VerificationCheck"
	urlStr := u.String()
	// Define request
	req, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(ACCOUNT_SID, AUTH_TOKEN)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Error making access token request")
	}
	requestDump, err := httputil.DumpRequest(req, true)
	fmt.Println(string(requestDump))
	if err != nil {
		fmt.Println(err.Error())
	}
	// Make request
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer res.Body.Close()
	// Read reponse
	body, err := ioutil.ReadAll(res.Body)
	fmt.Println(string(body))
	if err != nil {
		fmt.Println(err.Error())
	}
	api.RespondRaw(w, body, http.StatusOK)
}
