package user

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"vibe/api"
	mDB "vibe/model/db"
	model "vibe/model/db"
	"vibe/store"
)

type Response struct {
	IsAvail bool `json:"isAvail"`
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	customer := &model.Customer{}
	fmt.Println("Getting user info...")
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			api.Respond(w, nil, http.StatusUnauthorized)
			return
		}
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	conn := store.Cache.Get()
	res, err := store.ToString(conn.Do("GET", sessionToken))
	if err != nil {
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}
	if res == "" {
		api.Respond(w, nil, http.StatusUnauthorized)
		return
	}
	defer conn.Close()
	fmt.Println(res)
	customer.Email = res
	api.Respond(w, customer, http.StatusAccepted)
}

func UsernameAvailablityCheck(w http.ResponseWriter, r *http.Request) {
	res := &Response{}
	res.IsAvail = false
	creds := &mDB.User{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}
	// Query db for user
	result := store.DB.QueryRow("SELECT user_name FROM users WHERE user_name=?", string(creds.UserName))
	if err != nil {
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}
	storedCreds := &mDB.User{}
	err = result.Scan(&storedCreds.UserName)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("Username Available")
			res.IsAvail = true
			api.Respond(w, res, http.StatusOK)
			return
		}
		log.Println("Bad DB query")
		log.Println(err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}
	log.Println("Username Taken")
	api.Respond(w, res, http.StatusConflict)
}

func SetDeleteStatus(w http.ResponseWriter, r *http.Request) {
	res := &Response{}
	res.IsAvail = true
	creds := &mDB.User{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}
	// Delete user's liked videos in the database
	_, err = store.DB.Exec("DELETE FROM videos_liked WHERE user_id = ?;", string(creds.UserId))
	if err != nil {
		log.Println("Error when removing user data from videos_liked:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// Delete user's chat messages in the database
	_, err = store.DB.Exec("DELETE FROM all_chats WHERE user_id = ?;", string(creds.UserId))
	if err != nil {
		log.Println("Error when removing user data from all_chats:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// Delete user's posted videos in the latest_videos table in database
	_, err = store.DB.Exec("DELETE FROM latest_videos WHERE user_id = ?;", string(creds.UserId))
	if err != nil {
		log.Println("Error when removing user data from latest_videos:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// Delete user's posted videos in the all_videos table in database
	_, err = store.DB.Exec("DELETE FROM all_videos WHERE user_id = ?;", string(creds.UserId))
	if err != nil {
		log.Println("Error when removing user data from all_videos:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// Delete user's favorited locations in the database
	_, err = store.DB.Exec("DELETE FROM favorites WHERE user_id = ?;", string(creds.UserId))
	if err != nil {
		log.Println("Error when removing user data from favorites:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// Delete user's account in the database
	_, err = store.DB.Exec("DELETE FROM users WHERE user_id = ?;", string(creds.UserId))
	if err != nil {
		log.Println("Error when removing user data from users:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// log.Println("User", string(creds.UserName), "updated with is_delete status", creds.IsDeleted)
	log.Println("User", string(creds.UserId), "has been deleted", creds.IsDeleted)

	res.IsAvail = false
	api.Respond(w, res, http.StatusOK)
}

func SetUserFollowing(w http.ResponseWriter, r *http.Request) {
	res := &Response{}
	res.IsAvail = false
	userFollow := &mDB.UserFollower{}
	err := json.NewDecoder(r.Body).Decode(userFollow)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}

	// insert user following user_name_following in the database
	_, err = store.DB.Exec("INSERT INTO user_follower (`user_id`, `user_id_following`) VALUES(?, ?);", string(userFollow.UserId), string(userFollow.UserIdFollowing))
	if err != nil {
		log.Println("Error when adding new user follow:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// update user following count in users table in the database
	_, err = store.DB.Exec("UPDATE users SET following_count = following_count + 1 WHERE user_id = ?", string(userFollow.UserId))
	if err != nil {
		log.Println("Error when incrementing user following_count:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// update user_following follower count in users table in the database
	_, err = store.DB.Exec("UPDATE users SET follower_count = follower_count + 1 WHERE user_id = ?", string(userFollow.UserIdFollowing))
	if err != nil {
		log.Println("Error when incrementing user follower_count:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	log.Println("User", string(userFollow.UserId), "is now following", userFollow.UserIdFollowing)

	res.IsAvail = true
	api.Respond(w, res, http.StatusOK)

}

func SetUserUnfollowing(w http.ResponseWriter, r *http.Request) {
	res := &Response{}
	res.IsAvail = false
	userFollow := &mDB.UserFollower{}
	err := json.NewDecoder(r.Body).Decode(userFollow)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}

	// delete user following user_following in the database
	_, err = store.DB.Exec("DELETE FROM user_follower WHERE user_id = ? and user_id_following = ?;", string(userFollow.UserId), string(userFollow.UserIdFollowing))

	if err != nil {
		log.Println("Error when removing user follow:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// update user following count in users table in the database
	_, err = store.DB.Exec("UPDATE users SET following_count = following_count - 1 WHERE user_id = ?", string(userFollow.UserId))
	if err != nil {
		log.Println("Error when decrementing user following_count:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// update user_following follower count in users table in the database
	_, err = store.DB.Exec("UPDATE users SET follower_count = follower_count - 1 WHERE user_id = ?", string(userFollow.UserIdFollowing))
	if err != nil {
		log.Println("Error when decrementing user follower_count:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	log.Println("User", string(userFollow.UserId), "is now unfollowing", userFollow.UserIdFollowing)

	res.IsAvail = true
	api.Respond(w, res, http.StatusOK)

}

type FollowerDataRequest struct {
	UserId          string `json:"user_id" db:"user_id"`
	UserIdFollowing string `json:"user_id_following" db:"user_id_following"`
}

// in general this should be used for the users following stories page
func GetFollowingData(w http.ResponseWriter, r *http.Request) {
	res := &Response{}
	res.IsAvail = false
	userFollow := &mDB.UserFollower{}
	err := json.NewDecoder(r.Body).Decode(userFollow)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}

	var payload = []byte(`{"user_follow_data": [`)

	// select rows of user_id that the given user is following
	rows, err := store.DB.Query("SELECT user_id_following FROM user_follower WHERE user_id = ?;", string(userFollow.UserId))

	if err != nil {
		log.Println("Error when fetching list of user followings:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// if we get here that means we have videos to send to client
	defer rows.Close()

	var count = 0
	for rows.Next() {
		log.Println("in rows.next")

		var user_id_following sql.NullString
		err := rows.Scan(&user_id_following)

		if err != nil {
			panic(err)
		} else {

			log.Println("Getting latest video and associated metadata for user: ", user_id_following)

			// marshall request data body first
			body, err := json.Marshal(&FollowerDataRequest{UserId: user_id_following.String})
			if err != nil {
				log.Println("Error marshalling request data: ", err)
				panic(err)
			}

			url := "https://cdn-api.vibecheck.tech/get-user-latest-data"    // url for GetUserLatestData
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(body)) // request
			if err != nil {
				log.Println("Error building request: ", err)
				panic(err)
			}
			req.Header.Add("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second} // create an http client
			response, err := client.Do(req)                   // send the request
			if err != nil {
				log.Println("Error while sending the response bytes: ", err)
				panic(err)
			}
			log.Println("Sent request for GetUserLatestData")
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK { // check if response is not 200 OK
				log.Println("Response not received correctly.......")
				log.Println(response.StatusCode)
			}

			responseBody := new(bytes.Buffer) // read response
			_, err = responseBody.ReadFrom(response.Body)
			if err != nil {
				log.Println("Error while reading the response bytes: ", err)
				panic(err)
			}

			payload = append(payload, []byte(responseBody.String())...)
			payload = append(payload, []byte(`,`)...)
			count = count + 1

		}
	}

	if count > 0 {
		payload = payload[:len(payload)-1]
	}
	payload = append(payload, []byte(`]}`)...)
	log.Println("payload in string format = ", string(payload))
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

func GetFollowerData(w http.ResponseWriter, r *http.Request) {
	res := &Response{}
	res.IsAvail = false
	userFollow := &mDB.UserFollower{}
	err := json.NewDecoder(r.Body).Decode(userFollow)
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}

	var payload = []byte(`{"user_follow_data": [`)

	// select rows of user_id that the given user is following
	rows, err := store.DB.Query("SELECT user_id FROM user_follower WHERE user_id_following = ?;", string(userFollow.UserId))

	if err != nil {
		log.Println("Error when fetching list of user followings:", err)
		api.Respond(w, res, http.StatusInternalServerError)
		return
	}

	// if we get here that means we have videos to send to client
	defer rows.Close()

	var count = 0
	for rows.Next() {
		log.Println("in rows.next")

		var user_id_following sql.NullString
		err := rows.Scan(&user_id_following)

		if err != nil {
			panic(err)
		} else {

			log.Println("Getting latest video and associated metadata for user: ", user_id_following)

			// marshall request data body first
			body, err := json.Marshal(&FollowerDataRequest{UserId: user_id_following.String})
			if err != nil {
				log.Println("Error marshalling request data: ", err)
				panic(err)
			}

			url := "https://cdn-api.vibecheck.tech/get-user-latest-data"    // url for GetUserLatestData
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(body)) // request
			if err != nil {
				log.Println("Error building request: ", err)
				panic(err)
			}
			req.Header.Add("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second} // create an http client
			response, err := client.Do(req)                   // send the request
			if err != nil {
				log.Println("Error while sending the response bytes: ", err)
				panic(err)
			}
			log.Println("Sent request for GetUserLatestData")
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK { // check if response is not 200 OK
				log.Println("Response not received correctly.......")
				log.Println(response.StatusCode)
			}

			responseBody := new(bytes.Buffer) // read response
			_, err = responseBody.ReadFrom(response.Body)
			if err != nil {
				log.Println("Error while reading the response bytes: ", err)
				panic(err)
			}

			payload = append(payload, []byte(responseBody.String())...)
			payload = append(payload, []byte(`,`)...)
			count = count + 1

		}
	}

	if count > 0 {
		payload = payload[:len(payload)-1]
	}
	payload = append(payload, []byte(`]}`)...)
	log.Println("payload in string format = ", string(payload))
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

func GetFollowingAndFollowerCount(w http.ResponseWriter, r *http.Request) {
	request := &model.UserRequest{}
	err := json.NewDecoder(r.Body).Decode(request) // decode request first
	if err != nil {
		api.Respond(w, nil, http.StatusBadRequest)
		return
	}

	var counts model.FFCounts

	// sql query for getting follower_count and following_count columns
	q_err := store.DB.QueryRow("select follower_count, following_count from users where user_id = ?;", string(request.UserId)).Scan(&counts.FollowerCount, &counts.FollowingCount)
	if q_err != nil {
		log.Println("Error when fetching follower and following counts:", q_err)
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}
	response := struct {
		Counts model.FFCounts `json:"counts"`
	}{
		Counts: counts,
	}

	jsonData, n_err := json.Marshal(response) // convert to JSON
	if n_err != nil {
		log.Println("Error Encountered when marshalling... ", n_err)
		panic(n_err)
	}

	log.Println("response in string format = ", string(jsonData))
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
