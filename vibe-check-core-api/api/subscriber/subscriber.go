package subscriber

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "io/ioutil"
	"log"
	"net/http"
	"regexp"
	"vibe/api"
	mDB "vibe/model/db"
	"vibe/store"

	_ "github.com/thedevsaddam/gojsonq"
)

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func isEmailValid(e string) bool {
	if len(e) < 3 && len(e) > 254 {
		return false
	}
	return emailRegex.MatchString(e)
}

func Subscribe(w http.ResponseWriter, r *http.Request) {
	// decode subscriber
	sub := &mDB.Subscriber{}
	err := json.NewDecoder(r.Body).Decode(sub)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check for email validity
	if !isEmailValid(sub.Email) {
		log.Println("Not a valid email")
		fmt.Printf("Not a valid email")
		api.Respond(w, true, http.StatusBadRequest)
		return
	}

	// Query db for existing subscriber
	result := store.DB.QueryRow("SELECT subscriber_id FROM subscribers WHERE email=?", string(sub.Email))

	if err != nil {
		api.Respond(w, true, http.StatusInternalServerError)
		return
	}
	storedSub := &mDB.Subscriber{}
	if err := result.Scan(&storedSub.Email); err == nil {
		log.Println("Subscriber already exists")
		api.Respond(w, true, http.StatusConflict)
		return
	} else if err == sql.ErrNoRows {
		// insert subscriber into db
		if _, err = store.DB.Query(`INSERT into subscribers (email) VALUES (?)`, string(sub.Email)); err != nil {
			// if issue with insert return error
			log.Println("error store")
			log.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully subscribed")
		fmt.Printf("Subscriber is: %+v", sub) 
		api.Respond(w, true, http.StatusCreated)

	} else {
		s := err.Error()
    fmt.Printf("type: %T; value: %q\n", s, s)
		log.Println("Bad DB query")
		log.Println(err)
		api.Respond(w, true, http.StatusInternalServerError)
		return
	}
}