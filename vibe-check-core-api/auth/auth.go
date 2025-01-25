package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "io/ioutil"
	"log"
	"net/http"
	"vibe/api"
	mAPI "vibe/model/api"
	model "vibe/model/auth"
	mDB "vibe/model/db"
	"vibe/store"

	_ "github.com/thedevsaddam/gojsonq"
	"golang.org/x/crypto/bcrypt"
)

func Signup(w http.ResponseWriter, r *http.Request) {
	authStatus := &model.Auth{}
	authStatus.IsAuth = false
	// decode creds
	creds := &mDB.User{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Check for empty values
	if string(creds.UserName) == "" || string(creds.Password) == "" || string(creds.Phone) == "" {
		log.Println("Empty field(s)")
		api.Respond(w, authStatus, http.StatusBadRequest)
		return
	}

	// Query db for existing user
	result := store.DB.QueryRow("SELECT user_id FROM users WHERE user_name=? OR phone=?", string(creds.UserName), string(creds.Phone))
	if err != nil {
		log.Println("error checking if user exists")
		api.Respond(w, authStatus, http.StatusInternalServerError)
		return
	}
	storedCreds := &mDB.User{}
	if err := result.Scan(&storedCreds.UserId); err == nil {
		log.Println("User already exists")
		api.Respond(w, authStatus, http.StatusConflict)
		return
	} else if err == sql.ErrNoRows {
		log.Println("Username available")
		//salt and hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 8)
		if err != nil {
			log.Println("error hash")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println(hashedPassword)
		// insert creds into db
		if _, err = store.DB.Query(`INSERT into users (user_id, user_name, password, phone, photo) VALUES (?, ?, ?, ?, ?)`, string(GenerateUUID()), string(creds.UserName), string(hashedPassword), string(creds.Phone), false); err != nil {
			// if issue with insert return error
			log.Println("error store")
			log.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// set session
		SetSession(w, creds.UserName)
		// if we reach this point, user password is set and default 200 status is sent

		log.Printf("Successfully signed up")
		fmt.Printf("User creds are: %+v", creds) // authStatus.IsAuth = true
		authStatus = &model.Auth{
			IsAuth: true,
			User: mAPI.User{
				UserId:   creds.UserId,
				UserName: creds.UserName,
				Phone:    creds.Phone,
			},
		}
		api.Respond(w, authStatus, http.StatusCreated)
	} else {
		log.Println("Bad DB query")
		log.Println(err)
		api.Respond(w, authStatus, http.StatusInternalServerError)
		return
	}
}

func UpdatePassword(w http.ResponseWriter, r *http.Request) {
	authStatus := &model.Auth{}
	authStatus.IsAuth = false
	// decode creds
	creds := &mDB.User{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Check for empty values
	if string(creds.Password) == "" || string(creds.Phone) == "" {
		log.Println("Empty field(s)")
		api.Respond(w, authStatus, http.StatusBadRequest)
		return
	}

	// Query db for existing user
	result := store.DB.QueryRow("SELECT user_id FROM users WHERE phone=?", string(creds.Phone))
	if err != nil {
		api.Respond(w, authStatus, http.StatusInternalServerError)
		return
	}
	storedCreds := &mDB.User{}
	if err := result.Scan(&storedCreds.UserId); err == nil {
		log.Println("User exists")
		//salt and hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 8)
		if err != nil {
			log.Println("error hash")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println(hashedPassword)
		// insert creds into db
		if _, err = store.DB.Query(`UPDATE users SET password = ? WHERE user_id = ?`, string(hashedPassword), string(storedCreds.UserId)); err != nil {
			// if issue with insert return error
			log.Println("error store")
			log.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// set session
		SetSession(w, storedCreds.UserName)
		// if we reach this point, user password is set and default 200 status is sent

		log.Printf("Successfully reset password")
		// fmt.Printf("User creds are: %+v", creds) // authStatus.IsAuth = true
		authStatus = &model.Auth{
			IsAuth: true,
			User: mAPI.User{
				UserId:   storedCreds.UserId,
				UserName: storedCreds.UserName,
				Phone:    storedCreds.Phone,
			},
		}
		api.Respond(w, authStatus, http.StatusCreated)

	} else if err == sql.ErrNoRows {
		log.Println("User does not exist")
		api.Respond(w, authStatus, http.StatusConflict)
		return

	} else {
		log.Println("Bad DB query")
		log.Println(err)
		api.Respond(w, authStatus, http.StatusInternalServerError)
		return
	}
}

func Signin(w http.ResponseWriter, r *http.Request) {
	authStatus := &model.Auth{}
	authStatus.IsAuth = false
	creds := &mDB.User{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		api.Respond(w, authStatus, http.StatusBadRequest)
		return
	}
	// Check for empty values
	if string(creds.UserName) == "" || string(creds.Password) == "" {
		log.Println("Empty field(s)")
		api.Respond(w, authStatus, http.StatusBadRequest)
		return
	}
	// Query db for user
	result := store.DB.QueryRow("SELECT user_id, user_name, password FROM users WHERE user_name=?", string(creds.UserName))
	if err != nil {
		api.Respond(w, authStatus, http.StatusInternalServerError)
	}
	// Obtain stored password
	storedCreds := &mDB.User{}
	err = result.Scan(&storedCreds.UserId, &storedCreds.UserName, &storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			println("Username not found")
			api.Respond(w, authStatus, http.StatusUnauthorized)
			return
		}
		log.Println("Bad DB query")
		log.Println(err)
		api.Respond(w, authStatus, http.StatusInternalServerError)
		return
	}
	// Compare stored hashed with hashed version of received password
	if err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		// If passwords don't match return 401
		log.Println("Incorrect password")
		api.Respond(w, authStatus, http.StatusUnauthorized)
		return
	}
	authStatus = &model.Auth{
		IsAuth: true,
		User: mAPI.User{
			UserId:   storedCreds.UserId,
			UserName: storedCreds.UserName,
			Phone:    storedCreds.Phone,
		},
	}
	// set session
	SetSession(w, authStatus.User.UserId)
	// if we reach this point, user password is correct and default 200 status is sent
	log.Println("Successfully signed in")
	api.Respond(w, authStatus, http.StatusOK)
}

func Signout(w http.ResponseWriter, r *http.Request) {
	// TODO logout method
	// currently handled by remove session method
}
