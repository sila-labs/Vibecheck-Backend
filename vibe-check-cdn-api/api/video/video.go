package video

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"

	"os"
	"strconv"
	"strings"
	"vibe/api"

	"fmt"
	"hash/fnv"
	"time"
	"vibe/store"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	log "github.com/sirupsen/logrus"
)

// central services info
var DESTINATION string
var STREAM_HOST string

// vibe content storage
var VIBE_CONTENT_STORAGE string
var VIBE_CONTENT_STREAM string

// vibe
var VIBE_THUMBNAIL string
var VIBE_VIDEO string
var VIBE_SELFIE string

// user-specific storage
var USER_CONTENT_STORAGE string
var USER_CONTENT_STREAM string
var USER_PICTURE string

// general content storage
var ASSETS_CONTENT_STORAGE string
var ASSETS_CONTENT_STREAM string

var ASSETS_IN_APP_ICONS string
var FALLBACK_CONTENT string

func init() {

	// load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("err loading: %v", err)
	}

	DESTINATION = os.Getenv("DESTINATION")
	VIBE_CONTENT_STORAGE = DESTINATION + "/videos"
	ASSETS_CONTENT_STORAGE = DESTINATION + "/assets"
	USER_CONTENT_STORAGE = DESTINATION + "/users"

	STREAM_HOST = os.Getenv("STREAM_HOST")
	VIBE_CONTENT_STREAM = STREAM_HOST + "/videos"
	ASSETS_CONTENT_STREAM = STREAM_HOST + "/assets"
	USER_CONTENT_STREAM = STREAM_HOST + "/users"
	ASSETS_IN_APP_ICONS = ASSETS_CONTENT_STREAM + "/inAppIcons"
	FALLBACK_CONTENT = ASSETS_IN_APP_ICONS + "/vibecheck_logo_white.png"

	USER_PICTURE = "user.png"

	VIBE_THUMBNAIL = os.Getenv("VIBE_THUMBNAIL")
	VIBE_VIDEO = os.Getenv("VIBE_VIDEO")
	VIBE_SELFIE = os.Getenv("VIBE_SELFIE")

}

// Response -> response for the util scope
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

// helper function to generate the location hash based off the name, lat an long
func GenerateLocationHashString(name string, lat string, long string) string {

	log.WithFields(log.Fields{
		"name": name,
		"lat":  lat,
		"long": long,
	}).Trace("generating location hash")

	hash32 := fnv.New32a()
	hash32.Write([]byte(name + lat + long))
	hashString := strconv.Itoa(int(hash32.Sum32()))

	log.WithFields(log.Fields{
		"hashString": hashString,
	}).Trace("location hash generated")

	return hashString
}

func ChunkUploadHandler(w http.ResponseWriter, r *http.Request) {

	log.Trace("entered video upload handler")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}
	var f *os.File

	file, uploadFile, err := r.FormFile("file")

	if err != nil {
		response.Message = "error occured while reading in file -> "
		log.Info(response.Message, err.Error())
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// Content-Range needed in header to determine overall size and
	// what chunk we are currently working with
	contentRangeHeader := r.Header.Get("Content-Range")
	rangeAndSize := strings.Split(contentRangeHeader, "/")
	rangeParts := strings.Split(rangeAndSize[0], "-")

	log.Trace("current content range: ", contentRangeHeader)

	// get max range
	rangeMax, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "Missing range in Content-Range header"})
		response.Message = "Missing range in Content-Range header"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// get files size
	fileSize, err := strconv.Atoi(rangeAndSize[1])
	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "Missing file size in Content-Range header"})
		response.Message = "Missing file size in Content-Range header"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// validate file size is within max bounds (100MB)
	if fileSize > 100*1024*1024 {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "File size should be less than 100MB"})
		response.Message = "File size should be less than 100MB"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// extract locationName, lat, long; concatenate string, and use as input for generating locationHash
	log.Trace("locationName processing...")
	locationName := r.FormValue("locationName")
	lat := r.FormValue("lat")
	lon := r.FormValue("lon")

	// take string lat and lon and make into float
	lat_float, err := strconv.ParseFloat(lat, 64)
	lon_float, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		response.Message = "there was an error parsing the lat and long"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// expand float to have 9 decimal points as is the the db table criteria
	formattedLat := fmt.Sprintf("%.9f", lat_float)
	formattedLon := fmt.Sprintf("%.9f", lon_float)
	locationHash := GenerateLocationHashString(locationName, formattedLat, formattedLon)

	// extract time_stamp and parse to match golang time.Time format, to have correct formatting when SQL inserting
	log.Info("time_stamp processing...")
	time_stamp_unparsed := r.FormValue("time_stamp")
	log.Info(time_stamp_unparsed)
	layout_react_native_time_stamp := "2006-01-02T15:04:05.000Z" // Must specify the layout of the input string
	time_stamp, err := time.Parse(layout_react_native_time_stamp, time_stamp_unparsed)
	if err != nil {
		log.Info("time_stamp parsing issue")
		log.Info(err)
		return
	}
	log.Info(time_stamp) // Output: 2022-12-29 23:46:02
	layout_folder_time_stamp := "2006-01-02-15-04-05"
	time_stamp_folder := time_stamp.Format(layout_folder_time_stamp)

	// extract user_id to provide linking of who posted this video
	log.Info("user_id processing...")
	user_id := r.FormValue("user_id")
	log.Info(user_id)

	filepath := VIBE_CONTENT_STORAGE + "/" + string(locationHash) + "/" + string(user_id) + "-" + string(time_stamp_folder)
	log.Info("File location-----> " + filepath)
	filename := r.Header.Get("x-file-name") //uploadFile.Filename
	log.Info("Filename-----> " + (filename))

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		err := os.MkdirAll(filepath, 0777)
		if err != nil {
			response.Message = "Error creating temporary directory: " + filepath
			log.Info(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
	}

	full_filepath := (filepath) + "/" + filename
	// create/append to current file being copied
	if f == nil {
		f, err = os.OpenFile(full_filepath, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModeAppend)
		if err != nil {
			response.Message = "Error creating file"
			log.Info(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
	}

	// copies bytes from file chunk to the file
	if _, err := io.Copy(f, file); err != nil {
		response.Message = "Error writing to a file"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	log.Trace("rangeMax: ", rangeMax, " fileSize: ", fileSize)

	// close file and report status
	defer f.Close()
	if rangeMax >= fileSize-1 {
		combinedFile := (VIBE_CONTENT_STORAGE + "/" + string(locationHash) + "/" + string(user_id) + "-" + string(time_stamp_folder) + "/" + filename)

		uploadingFile, err := os.Open(combinedFile)
		if err != nil {
			response.Message = "Failed to upload file "
			log.Info(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
		uploadingFile.Close()

		log.Info("video file sucessfully hit the server")

		// INSERT into locations table if first-ever video upload to location
		query := "INSERT INTO locations (location_hash, location_name, lat, lon) SELECT ?, ?, ?, ? WHERE NOT EXISTS (SELECT * FROM locations WHERE location_hash = ?)"
		video_folder := string(user_id) + "-" + string(time_stamp_folder)

		result, err := store.DB.Exec(query, locationHash, locationName, formattedLat, formattedLon, locationHash)
		if err != nil {
			log.Info("INSERT INTO locations failed: ")
			log.Info(err.Error())
			return
		} else {
			rows, err := result.RowsAffected()
			if err != nil {
				log.Error("error retrieving rows affected, database most likely not supported")
			} else {
				if rows != 0 {
					log.Info("added to locations , rows affected: ", rows)
				}
			}
		}

		// INSERT into general videos store
		query = "INSERT INTO `vibe_db`.`all_videos` (`video_folder`, `location_hash`, `user_id`, `time_stamp`, `like_count`) VALUES (?, ?, ?, ?, ?)"
		video_folder = string(user_id) + "-" + string(time_stamp_folder)

		result, err = store.DB.Exec(query, video_folder, locationHash, user_id, time_stamp, 0)
		// result := store.DB.QueryRow(query, locationHash, uID_timeStamp)
		if err != nil {
			log.Error("add vibe to all_videos table failed: ", err.Error())
			return
		} else {
			rows, err := result.RowsAffected()
			if err != nil {
				log.Error("error retrieving rows affected, database most likely not supported")
			} else {
				if rows != 0 {
					log.Info("added vibe to all_videos, rows affected: ", rows)
				}
			}
		}

		// INSERT into `latest.video` not `all_videos` store {locationHash: latestVideo}
		// query = "INSERT INTO `vibe_db`.`latest_videos` (`location_hash`, `video_folder`) VALUES (?, ?) ON DUPLICATE KEY UPDATE video_folder = VALUES(video_folder)"
		query = "INSERT INTO `vibe_db`.`latest_videos` (`video_folder`, `location_hash`, `user_id`, `time_stamp`, `like_count`) VALUES (?, ?, ?, ?, ?)"

		result, err = store.DB.Exec(query, locationHash, video_folder)
		if err != nil {
			log.Error("add vibe to latest_videos failed, ", err.Error())
		} else {
			rows, err := result.RowsAffected()
			if err != nil {
				log.Error("error retrieving rows affected, database most likely not supported")
			} else {
				if rows != 0 {
					log.Info("added vibe to latest_videos , rows affected: ", rows)
				}
			}
		}

		log.Info("sucessfully created database items for current video")

		// if strings.HasSuffix(filename, ".mp4") {

		// 	// perform chunking of .mp4 file to allow for increased ability to stream content
		// 	input_url := VIBE_CONTENT_STORAGE + "/" + string(locationHash) + "/" + string(user_id) + "-" + string(time_stamp_folder) + "/" + string(filename)
		// 	storage_location := VIBE_CONTENT_STORAGE + "/" + string(locationHash) + "/" + string(user_id) + "-" + string(time_stamp_folder) + "/"
		// 	output_file := storage_location + "outputlist.m3u8"

		// 	log.Info("running ffmpeg")

		// 	err = ffmpeg_go.Input(input_url).
		// 		Output(output_file, ffmpeg_go.KwArgs{
		// 			// "b:v":               "5000k",
		// 			"s":        "720:1280",
		// 			"level":    "3.0",
		// 			"hls_time": "1",
		// 			// "hls_playlist_type": "vod",
		// 			// "hls_flags":         "independent_segments",
		// 			// "hls_segment_type":  "mpegts",
		// 			"hls_list_size": "0",
		// 			"f":             "hls"}).
		// 		OverWriteOutput().ErrorToStdOut().Run()

		// 	log.Info("output file path: " + output_file)

		// 	if err != nil {
		// 		// error occured, removing file from the directory
		// 		log.Error("error occured running ffmpeg" + err.Error())
		// 		RemoveFile(storage_location + "video.mp4")
		// 		return
		// 	} else {
		// 		log.Info("ffmpeg completed")
		// 	}

		// 	log.Info("generating thumbnail")

		// 	// generate thumbnail image of .mp4 file to use when looking at pins/ReelsView
		// 	input_url = VIBE_CONTENT_STORAGE + "/" + string(locationHash) + "/" + string(user_id) + "-" + string(time_stamp_folder) + "/" + string(filename)
		// 	output_location_thumbnail := VIBE_CONTENT_STORAGE + "/" + string(locationHash) + "/" + string(user_id) + "-" + string(time_stamp_folder) + "/" + "thumbnail.jpg"

		// 	err = ffmpeg_go.Input(input_url).Output(output_location_thumbnail, ffmpeg_go.KwArgs{
		// 		"ss":      "00:00:01",
		// 		"vframes": "1",
		// 		"s":       "320x640"}).
		// 		OverWriteOutput().ErrorToStdOut().Run()

		// 	if err != nil {
		// 		// error occured, removing file from the directory
		// 		log.Error("error occured running ffmpeg" + err.Error())
		// 		RemoveFile(storage_location + "video.mp4")
		// 		return
		// 	} else {
		// 		log.Info("ffmpeg thumbnail completed")
		// 	}

		// }

		response.Message = "file successfully uploaded"
		response.Success = true
		response.Name = uploadFile.Filename
		log.Info(response.Message)
		api.RespondOK(w, response)
	} else {
		response.Message = "file chunk hit the server, moving on to the next chunk"
		response.Success = true
		response.Name = uploadFile.Filename
		log.Info(response.Message)
		api.Respond(w, response, 201) // alows the next() function to be called on client api
	}

}

// helper function to remove a file when given a path
func RemoveFile(file string) {
	log.Trace("attemtping to remove file: ", file)
	e := os.Remove(file)
	if e != nil {
		log.Fatal(e)
	} else {
		log.Info("sucessfully removed file: ", file)
	}
}

func UserPicUploadHandler(w http.ResponseWriter, r *http.Request) {

	log.Info("in user pic upload handler-------------------------")
	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}
	log.Info(response.Message)
	var f *os.File

	file, uploadFile, err := r.FormFile("file")

	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "content-type should be multipart/formdata"})
		log.Info(response.Message)
		return
	}

	// user_id
	// extract user_id to provide linking of whose photo this is
	log.Info("user_id processing: ")
	user_id := r.FormValue("user_id")
	log.Info(user_id)

	// Content-Range needed in header to determine overall size and
	// what chunk we are currently working with
	contentRangeHeader := r.Header.Get("Content-Range")
	rangeAndSize := strings.Split(contentRangeHeader, "/")
	rangeParts := strings.Split(rangeAndSize[0], "-")

	// get max range
	rangeMax, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "Missing range in Content-Range header"})
		response.Message = "Missing range in Content-Range header"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// get files size
	fileSize, err := strconv.Atoi(rangeAndSize[1])
	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "Missing file size in Content-Range header"})
		response.Message = "Missing file size in Content-Range header"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// validate file size is within max bounds (100MB)
	if fileSize > 100*1024*1024 {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "File size should be less than 100MB"})
		response.Message = "File size should be less than 100MB"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// create temp directory
	log.Info("File location----->" + (USER_CONTENT_STORAGE + "/" + string(user_id)))
	filename := uploadFile.Filename
	log.Info("Filename----->" + (filename))

	if _, err := os.Stat(USER_CONTENT_STORAGE + "/" + string(user_id)); os.IsNotExist(err) {
		err := os.MkdirAll(USER_CONTENT_STORAGE+"/"+string(user_id), 0777)
		if err != nil {
			// c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating temporary directory"})
			response.Message = "Error creating temporary directory"
			log.Info(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
	}

	// create/append to current file being copied
	if f == nil {
		f, err = os.OpenFile((USER_CONTENT_STORAGE+"/"+string(user_id))+"/"+filename, os.O_TRUNC|os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating file"})
			response.Message = "Error creating user pic file"
			log.Info(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
	}

	// copies bytes from file chunk to the file
	if _, err := io.Copy(f, file); err != nil {
		// c.JSON(http.StatusInternalServerError, gin.H{"error": "Error writing to a file"})
		response.Message = "Error writing to user pic file"
		log.Info(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// close file and report status
	defer f.Close()
	if rangeMax >= fileSize-1 {
		combinedFile := (USER_CONTENT_STORAGE + "/" + string(user_id) + "/" + filename)

		uploadingFile, err := os.Open(combinedFile)
		if err != nil {
			response.Message = "Failed to upload user pic file "
			log.Info(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
		uploadingFile.Close()

		// return
	}

	query := "UPDATE users SET photo = ? WHERE user_id = ?"

	result, err := store.DB.Exec(query, true, user_id)
	// result := store.DB.QueryRow(query, locationHash, uID_timeStamp)
	if err != nil {
		log.Info("upload user pic failed:")
		log.Info(err)
	} else {
		log.Info("upload user pic worked:")
		log.Info(result)
	}

	response.Message = "Uploaded User Pic File Successfully"
	response.Success = true
	response.Name = uploadFile.Filename
	log.Info(response.Message)
	api.Respond(w, response, http.StatusOK)

}

type GiftedUserStruct struct {
	// _id    int    `json:"_id"`
	Id     string `json:"_id"` // will using string be an issue? // setting _id to be same value as name for now
	UserID string `json:"user_id"`
	Avatar string `json:"avatar"`
}

type GiftedChatStruct struct {
	// _id       int        `json:"_id"`
	Id        string           `json:"_id"` // will using string be an issue?
	Text      string           `json:"text"`
	CreatedAt time.Time        `json:"createdAt"`
	User      GiftedUserStruct `json:"user"`
}

func ChatMessageUpload(w http.ResponseWriter, r *http.Request) {
	log.Info("in chat msg upload handler-------------------------")
	log.Info(r)
	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	log.Info("location_name processing: ")
	location_name := r.FormValue("location_name")
	log.Info(location_name)
	lat := r.FormValue("lat")
	lon := r.FormValue("lon")
	thread_name := r.FormValue("thread_name")

	// latString := "39.950"

	// take string lat and lon and make into float
	lat_float, err := strconv.ParseFloat(lat, 64)
	lon_float, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		// handle error
	}

	// expand float to have 9 decimal points as is the the db table criteria
	formattedLat := fmt.Sprintf("%.9f", lat_float)
	formattedLon := fmt.Sprintf("%.9f", lon_float)

	locationHash := GenerateLocationHashString(location_name, formattedLat, formattedLon)

	// var msg []GiftedChatStruct
	text := r.FormValue("text")
	user_id := r.FormValue("user_id")
	createdAtRaw := r.FormValue("createdAt")
	_id := r.FormValue("_id")

	// json.Unmarshal([]byte(message), &msg)
	// if err != nil {
	// 	fmt.Println("error:", err)
	// }5
	// fmt.Printf("%+v", msg)
	log.Info("text is: ")
	log.Info(text)
	log.Info("user_id is: ")
	log.Info(user_id)
	log.Info("createdAt is: ")
	log.Info(createdAtRaw)
	log.Info("_id is: ") // message _id
	log.Info(_id)

	// extract time_stamp and parse to match golang time.Time format, to have correct formatting when SQL inserting
	log.Info("time_stamp processing: ")
	layout_react_native_time_stamp := "2006-01-02T15:04:05.000Z" // Must specify the layout of the input string
	createdAt, err := time.Parse(layout_react_native_time_stamp, createdAtRaw)
	if err != nil {
		// Handle error
		log.Info("time_stamp parsing issue")
		log.Info(err)
		return
	}
	log.Info(createdAt) // Output: 2022-12-29 23:46:02

	// insert into table all_chats, most likely will need to use a noSQL db but for now using mysql since we have boilerplate code/know-how

	// insert into all_chats
	query := "INSERT INTO all_chats (location_hash, thread_name, _id, msg_text, createdAt, user_id) SELECT ?, ?, ?, ?, ?, ?"
	log.Info(query)
	result, err := store.DB.Exec(query, locationHash, thread_name, _id, text, createdAt, user_id)
	// result := store.DB.QueryRow(query, locationHash, uID_timeStamp)
	if err != nil {
		log.Info("INSERT INTO all_chats failed:")
		log.Info(err)
		return
	} else {
		log.Info("no isses with all_chats, insert worked:")
		log.Info(result)
	}

	response.Message = "Uploaded Chat Message Successfully"
	response.Success = true
	// response.Name = user_id + createdAt
	log.Info(response.Message)
	api.Respond(w, response, http.StatusOK)

}

/*
Get the most recent location data via the location name, latitude, longitude, and requesting user from the database
*/
func GetLocationChat(w http.ResponseWriter, r *http.Request) {

	//temp path location
	//UPLOADS_DIR := "/root/UPLOADS/"

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// params := mux.Vars(r)

	// locationName_raw := params["locationName"]
	// // log.Info("locationName_raw in GetLocationChat is: " + string(locationName_raw))

	// lat_raw := params["lat"]
	// lon_raw := params["lon"]
	// // latString := "39.950"

	log.Info("location_name processing: ")
	locationName_raw := r.FormValue("location_name")
	log.Info(locationName_raw)
	lat_raw := r.FormValue("lat")
	lon_raw := r.FormValue("lon")
	threadName := r.FormValue("thread_name")

	// take string lat and lon and make into float
	lat_float, err := strconv.ParseFloat(lat_raw, 64)
	lon_float, err := strconv.ParseFloat(lon_raw, 64)
	if err != nil {
		// handle error
		log.Info("issue with lat/lon float gen in GetLocationChat")
	}

	// expand float to have 9 decimal points as is the the db table criteria
	formattedLat := fmt.Sprintf("%.9f", lat_float)
	formattedLon := fmt.Sprintf("%.9f", lon_float)
	locationHash := GenerateLocationHashString(locationName_raw, formattedLat, formattedLon)

	// Query db for all videos based on user_id
	// ending result set, a list of videos, should contain:
	// locationHash + video_folder (these values are used to generate the hyperlink that the streaming service will respond to)
	// time_stamp (to show on ReelsView)
	// location_name (to show on ReelsView)
	// lat, lon (to navigate to given location in MapView when user taps on title in ReelsView)
	// username (to display on ReelsView)
	// profile_picture (to display on ReelsViews)

	var payload = []byte(`{"messages": [`)
	var entry interface{}
	// log.Info(entry)
	//
	rows, err := store.DB.Query("SELECT * FROM all_chats WHERE location_hash = ? AND thread_name = ?  ORDER BY createdAt DESC", locationHash, threadName)
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Info("error in location-indexed chat SQL query")
			log.Info(err)
			panic(err)
		} else {
			// NO VIDEO RESULT IN DB - send empty array
			// log.Info("result set is empty in chat SQL query")

			payload = append(payload, []byte(`]}`)...)
			api.RespondRaw(w, payload, http.StatusOK)
		}

	}

	// log.Info("query is not null")

	// if we get here that means we have videos to send to client
	defer rows.Close()
	// buf := bytes.NewBuffer(newJSON)
	// encoder := json.NewEncoder(buf)

	var id = 0
	for rows.Next() {
		var _id string
		var location_hash string
		var thread_name string
		var user_id string //as an int yes, no?
		var createdAt time.Time
		var text string

		// log.Info("in rows.next")
		err := rows.Scan(&_id, &location_hash, &thread_name, &user_id, &createdAt, &text)

		if err != nil {
			panic(err)
		} else {
			// log.Info("found video result in db preparing chat struct")
			// FOUND VIDEO RESULT IN DB - send VideoStruct

			//create user_pic_link to use as avatar
			user_pic_link := USER_CONTENT_STREAM + "/" + user_id + "/" + USER_PICTURE

			// type GiftedUserStruct struct {
			// 	// _id    int    `json:"_id"`
			// 	Id    string `json:"_id"` // will using string be an issue? // setting _id to be same value as name for now
			// 	Name   string `json:"name"`
			// 	Avatar string `json:"avatar"`
			// }

			// type GiftedChatStruct struct {
			// 	// _id       int        `json:"_id"`
			// 	Id       string           `json:"_id"` // will using string be an issue?
			// 	Text      string           `json:"text"`
			// 	CreatedAt time.Time        `json:"createdAt"`
			// 	User      GiftedUserStruct `json:"user"`
			// }

			userStruct := GiftedUserStruct{Id: user_id, UserID: user_id, Avatar: user_pic_link}

			chatStruct := GiftedChatStruct{Id: _id, Text: text, CreatedAt: createdAt, User: userStruct}

			id = id + 1
			// Encode the data to JSON
			entry = chatStruct
			jsonData, err := json.Marshal(entry)
			if err != nil {
				log.Info(err)
				return
			}

			payload = append(payload, jsonData...)
			payload = append(payload, []byte(`,`)...)
			// log.Info("adding result row to response")
		}
		// log.Info(_id, location_hash, name, createdAt, text)
	}

	if id > 0 {
		payload = payload[:len(payload)-1]
	}
	payload = append(payload, []byte(`]}`)...)

	// log.Info(string(payload))
	// Output: {"baz": [{"foo":"hello","bar":42}]}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	if err != nil {
		log.Info("Unable to connect to DB")
		response.Message = "unable to connect to database server"
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}

	//api.RespondOK(w, msg)
	api.RespondRaw(w, payload, http.StatusOK)

}

type LocationDataQuery struct {
	Location string `json:"location"`
	Lat      string `json:"lat"`
	Lon      string `json:"lon"`
	UserId   string `json:"userID"`
}

var locationData LocationDataQuery

/*
Get the most recent location data via the location name, latitude, longitude, and requesting user from the database
*/
func GetLocationLatestData(w http.ResponseWriter, r *http.Request) {

	log.Trace("entered GetLocationLatestData")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    ""}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&locationData)
	if err != nil {
		log.Panic(err)
	}

	var loc string
	var user_id string
	var _lat string
	var _lon string

	loc = locationData.Location
	user_id = locationData.UserId
	_lat = locationData.Lat
	_lon = locationData.Lon

	log.WithFields(log.Fields{
		"Location": locationData.Location,
		"Lat":      locationData.Lat,
		"Lon":      locationData.Lon,
		"UserID":   locationData.UserId,
	}).Info("location data")

	// take lat and lon strings and convert to float
	lon_float, err := strconv.ParseFloat(_lon, 64)
	if err != nil {
		log.Error("could not parse incomming lon to float")
	}

	lat_float, err := strconv.ParseFloat(_lat, 64)
	if err != nil {
		log.Error("could not parse incomming lat to float")
	}

	// expand float to have 9 decimal points as is the the db table criteria
	formattedLat := fmt.Sprintf("%.9f", lat_float)
	formattedLon := fmt.Sprintf("%.9f", lon_float)
	locationHash := GenerateLocationHashString(loc, formattedLat, formattedLon)

	// INSERT into locations table if first-ever video upload to location
	query := "INSERT INTO locations (location_hash, location_name, lat, lon) SELECT ?, ?, ?, ? WHERE NOT EXISTS (SELECT * FROM locations WHERE location_hash = ?)"
	result, err := store.DB.Exec(query, locationHash, loc, formattedLat, formattedLon, locationHash)
	if err != nil {
		response.Message = "INSERT INTO locations failed " + err.Error()
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	} else {
		rows, err := result.RowsAffected()
		if err != nil {
			log.Error("error retrieving rows affected, database most likely not supported")
		} else {
			if rows != 0 {
				log.Info("attempted to insert into locations table, rows affected: ", rows)
			}
		}
	}

	// get status on whether the user has liked this location or not
	var isLiked bool
	query = "SELECT * FROM favorites WHERE user_id = ? AND favorites.location_hash = ?"

	var favoritesUserID string
	var favoritesLocation string
	err = store.DB.QueryRow(query, user_id, locationHash).Scan(&favoritesUserID, &favoritesLocation)

	// might need to do special mysql error handling, outside of `sql: no rows in result set` error which will be handled below
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Error("issue with accessing favorites table in db")
		} else {
			isLiked = false
		}
	} else {
		if favoritesUserID != "" && favoritesLocation != "" {
			// FOUND FAVORITE RESULT MATCH IN DB - set isLiked to true
			isLiked = true
		}
	}

	// Query db for video based on location
	var id = 0
	var video_folder string
	var location_hash string
	var video_like_count float64
	var video_is_liked_by_user bool
	var time_stamp time.Time
	var result_user_id string
	var user_name string
	var photo bool
	var location_name string
	var lat float64
	var lon float64

	var payload interface{}
	var videoPayload interface{}

	err = store.DB.QueryRow("SELECT all_videos.video_folder, all_videos.location_hash, all_videos.like_count, IF(ISNULL(videos_liked.user_id), false, true) AS 'is_liked', time_stamp, users.user_id, users.user_name, location_name, lat, lon FROM all_videos JOIN locations ON all_videos.location_hash = locations.location_hash JOIN users ON all_videos.user_id = users.user_id LEFT JOIN videos_liked ON all_videos.video_folder = videos_liked.video_folder AND all_videos.location_hash = videos_liked.location_hash AND videos_liked.user_id = ? WHERE all_videos.location_hash = ? AND all_videos.is_deleted = 0 ORDER BY all_videos.time_stamp DESC LIMIT 1", user_id, locationHash).
		Scan(&video_folder, &location_hash, &video_like_count, &video_is_liked_by_user, &time_stamp, &result_user_id, &user_name, &location_name, &lat, &lon)
	// +----------------------------------+---------------+------------+----------+---------------------+--------------+---------------+--------------+---------------+
	// | video_folder                     | location_hash | like_count | is_liked | time_stamp          | user_name    | location_name | lat          | lon           |
	// +----------------------------------+---------------+------------+----------+---------------------+--------------+---------------+--------------+---------------+
	// | frankmantest-2023-02-24-20-38-10 | 1220036614    |       NULL |        0 | 2023-02-24 20:38:10 | frankmantest | Galway's Pub  | 39.950474500 | -75.262460000 |
	// +----------------------------------+---------------+------------+----------+---------------------+--------------+---------------+--------------+---------------+
	// orig when using user_id to query err = store.DB.QueryRow("SELECT video_folder, all_videos.location_hash, time_stamp, all_videos.user_id, user_name, location_name, lat, lon FROM all_videos JOIN locations ON all_videos.location_hash = locations.location_hash JOIN users ON all_videos.user_name = users.user_name WHERE all_videos.location_hash = ? ORDER BY all_videos.time_stamp DESC LIMIT 1;", locationHash).Scan(&video_folder, &location_hash, &time_stamp, &user_name, &username, &location_name, &lat, &lon)

	log.WithFields(log.Fields{
		"video_folder":           video_folder,
		"location_hash":          location_hash,
		"video_like_count":       video_like_count,
		"video_is_liked_by_user": video_is_liked_by_user,
		"time_stamp":             time_stamp,
		"user_id":                result_user_id,
		"user_name":              user_name,
		"photo":                  photo,
		"location_name":          location_name,
		"lat":                    lat,
		"lon":                    lon,
	}).Trace("location data")

	// might need to do special mysql error handling, outside of `sql: no rows in result set` error which will be handled below
	if err != nil {
		log.Error(err.Error())
		payload = VibecheckLocationData{
			Video:   VideoStruct{},
			IsLiked: isLiked,
		}
	} else { // found video result in db
		thumbnail_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_THUMBNAIL
		video_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_VIDEO
		selfie_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_SELFIE
		var user_pic_link string
		if photo == true {
			user_pic_link = USER_CONTENT_STREAM + "/" + result_user_id + "/" + USER_PICTURE
		} else {
			user_pic_link = FALLBACK_CONTENT
		}

		log.WithFields(log.Fields{
			"thumbnail_link": thumbnail_link,
			"video_link":     video_link,
			"selfie_link":    selfie_link,
			"user_pic_link":  user_pic_link,
		}).Info("video result data")

		videoPayload = VideoStruct{
			Id:                 id,
			ThumbnailLink:      thumbnail_link,
			VideoLink:          video_link,
			SelfieLink:         selfie_link,
			UserPicLink:        user_pic_link,
			VideoFolder:        video_folder,
			LocationHash:       location_hash,
			VideoLikeCount:     video_like_count,
			VideoIsLikedByUser: video_is_liked_by_user,
			TimeStamp:          time_stamp,
			UserId:             result_user_id,
			Username:           user_name,
			LocationName:       location_name,
			Lat:                lat,
			Lon:                lon}

		log.Trace(videoPayload)

		// assert that videoPayload is of type VideoStruct
		videoStruct, ok := videoPayload.(VideoStruct)
		if !ok {
			// handle the case where videoPayload is not a VideoStruct
			response.Message = "issue parsing video payload to VideoStruct"
			api.Respond(w, nil, http.StatusInternalServerError)
			return
		} else {
			payload = VibecheckLocationData{
				Video:   videoStruct,
				IsLiked: isLiked,
			}
		}
	}
	api.RespondOK(w, payload)

}

/* Set the given video to increment or decrement by one, then if incremented, add the user-video like mapping
 */
func SetVideoLikedStatus(w http.ResponseWriter, r *http.Request) {

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	decoder := json.NewDecoder(r.Body)
	var q VideoLikedSetter
	err := decoder.Decode(&q)
	if err != nil {
		panic(err)
	}
	log.Info(q.VideoFolder)

	video_folder := q.VideoFolder
	location_hash := q.LocationHash

	log.Info("video_folder in setVideoLikedStatus is: " + string(video_folder))

	user_id := q.UserId
	liked_status := q.LikedStatus

	if err != nil {
		// handle error
		log.Info("issue with data retrieval from API call in setVideoLikedStatus")
	}
	like_count := 0
	if liked_status == false {
		like_count = -1
	} else if liked_status == true {
		like_count = 1
	}

	// UPDATE all_videos
	query := "UPDATE all_videos SET like_count = IFNULL(like_count, 0) + ? WHERE video_folder = ? and location_hash = ?"
	result, err := store.DB.Exec(query, like_count, video_folder, location_hash)
	// result := store.DB.QueryRow(query, locationHash, uID_timeStamp)
	if err != nil {
		log.Info("UPDATE all_videos failed:")
		log.Info(err)
		return
	} else {
		log.Info("no isses with UPDATE all_videos:")
		log.Info(result)
	}

	// UPDATE latest_videos
	query = "UPDATE latest_videos SET like_count = IFNULL(like_count, 0) + ? WHERE video_folder = ? and location_hash = ?"
	result, err = store.DB.Exec(query, like_count, video_folder, location_hash)
	// result := store.DB.QueryRow(query, locationHash, uID_timeStamp)
	if err != nil {
		log.Info("UPDATE latest_videos failed:")
		log.Info(err)
		return
	} else {
		log.Info("no isses with UPDATE latest_videos:")
		log.Info(result)
	}

	query = ""

	if liked_status == true {
		query = "INSERT INTO videos_liked (`video_folder`, `location_hash`, `user_id`) VALUES (?, ?, ?)"
	} else { // false, remove
		query = "DELETE FROM videos_liked WHERE video_folder = ? AND location_hash = ? AND user_id = ?"
	}
	// query = "INSERT INTO favorites (`location_hash`, `user_name`) VALUES (?, ?) ON DUPLICATE KEY UPDATE video_folder = VALUES(video_folder)"

	result, err = store.DB.Exec(query, video_folder, location_hash, user_id)
	// result := store.DB.QueryRow(query, locationHash, uID_timeStamp)
	if err != nil {
		log.Info("INSERT INTO videos_liked failed:")
		log.Info(err)
		return
	} else {
		// log.Info("INSERT INTO favorites worked:")
		log.Info(result)
	}

	response.Message = "Video Like Change Updated Successfully"
	response.Success = true
	response.Name = "video_like_change"
	// log.Info(response.Message)
	api.Respond(w, response, http.StatusOK)

}

type VideoLikedSetter struct {
	VideoFolder  string `json:"videoFolder"`
	LocationHash string `json:"locationHash"`
	UserId       string `json:"userID"`
	LikedStatus  bool   `json:"likedStatus"`
}

// so far, this struct will handle latest video of location and status of whether user has liked location or not
type VibecheckLocationData struct {
	Video   VideoStruct `json:"video"`
	IsLiked bool        `json:"isLiked"`
}

type VibecheckUserData struct {
	UserId      string `json:"userID"`
	Username    string `json:"username"`
	UserPicLink string `json:"userPicLink"`
	Following   bool   `json:"following"`
}

type VideoStruct struct {
	Id                 int       `json:"id"`
	ThumbnailLink      string    `json:"thumbnailLink"`
	VideoLink          string    `json:"videoLink"`
	PosterLink         string    `json:"posterLink"`
	SelfieLink         string    `json:"selfieLink"`
	UserPicLink        string    `json:"userPicLink"`
	VideoFolder        string    `json:"videoFolder"`
	LocationHash       string    `json:"locationHash"`
	VideoLikeCount     float64   `json:"videoLikeCount"`
	VideoIsLikedByUser bool      `json:"videoIsLikedByUser"`
	TimeStamp          time.Time `json:"timeStamp"`
	UserId             string    `json:"userID"`
	Username           string    `json:"username"`
	LocationName       string    `json:"locationName"`
	Lat                float64   `json:"lat"`
	Lon                float64   `json:"lon"`
}

type TestVideoStruct struct {
	ThumbnailLink sql.NullString `json:"thumbnailLink"`
	VideoLink     sql.NullString `json:"videoLink"`
	PosterLink    sql.NullString `json:"posterLink"`
	SelfieLink    sql.NullString `json:"selfieLink"`
	UserPicLink   sql.NullString `json:"userPicLink"`
	VideoFolder   sql.NullString `json:"videoFolder"`
	LocationHash  sql.NullString `json:"locationHash"`
	TimeStamp     sql.NullTime   `json:"timeStamp"`
	UserId        sql.NullString `json:"userID"`
	Username      sql.NullString `json:"username"`
	LocationName  sql.NullString `json:"locationName"`
	Lat           sql.NullString `json:"lat"`
	Lon           sql.NullString `json:"lon"`
}

// when no hits in the db and need to display generic content
type EmptyVideoStruct struct {
	ThumbnailLink string `json:"thumbnailLink"`
	VideoLink     string `json:"videoLink"`
}

type UserFollower struct {
	UserId          string `json:"user_id" db:"user_id"`
	UserIdFollowing string `json:"user_id_following" db:"user_id_following"`
}

/*
Get the set of videos associated with a given user_id from the database
*/
func GetDataByUser(w http.ResponseWriter, r *http.Request) {

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// decode creds
	userFollower := &UserFollower{}
	err := json.NewDecoder(r.Body).Decode(userFollower)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user_id := userFollower.UserId
	user_id_following := userFollower.UserIdFollowing

	// Check for empty values
	if string(user_id_following) == "" {
		// just user requesting their own page
		user_id_following = user_id
	}

	log.Trace("user_id is: ", user_id)
	log.Trace("in GetDataByUser")

	var payload = []byte(`{"data": [`)
	var user_name string
	var hasPhoto bool
	var isFollowing bool
	// SELECT users.user_name, users.photo, IF(ISNULL(user_follower.user_id), false, true) AS 'is_following' FROM users LEFT JOIN user_follower on user_follower.user_id_following = users.user_id and user_follower.user_id = '91752724-8c77-473d-85c0-0d22ace7d769' where users.user_id = '18fea441-e325-4893-9cc7-76b8ab2b7cad';
	err = store.DB.QueryRow("SELECT users.user_name, users.photo, IF(ISNULL(user_follower.user_id), false, true) AS 'is_following' FROM users LEFT JOIN user_follower on user_follower.user_id_following = users.user_id and user_follower.user_id = ? where users.user_id = ?;", user_id, user_id_following).Scan(&user_name, &hasPhoto, &isFollowing) //true or false
	// err = store.DB.QueryRow("SELECT user_name, photo from users where user_id = ?", user_id).Scan(&user_name, &hasPhoto) //true or false

	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Info("error in user-indexed video query SQL")
			log.Info(err)
			panic(err)
		} else {

			// this shouldn't be hit
			entry := VibecheckUserData{}
			// Encode the data to JSON
			jsonData, err := json.Marshal(entry)
			if err != nil {
				log.Info(err)
				return
			}
			payload = append(payload, jsonData...)
			// api.RespondRaw(w, payload, http.StatusOK)
		}

	}

	var query_user_pic_link string
	var jsonData []byte
	if hasPhoto == true {
		// return default link to streaming content
		query_user_pic_link = USER_CONTENT_STREAM + "/" + user_id_following + "/" + USER_PICTURE
		entry := VibecheckUserData{UserId: user_id_following, Username: user_name, Following: isFollowing, UserPicLink: query_user_pic_link}
		// Encode the data to JSON
		jsonData, err = json.Marshal(entry)
		if err != nil {
			log.Info(err)
			return
		}
		payload = append(payload, jsonData...)

	} else {
		// return empty string for user_pic_link
		query_user_pic_link = FALLBACK_CONTENT
		entry := VibecheckUserData{UserId: user_id_following, Username: user_name, Following: isFollowing, UserPicLink: query_user_pic_link}
		// Encode the data to JSON
		jsonData, err = json.Marshal(entry)
		if err != nil {
			log.Info(err)
			return
		}
		payload = append(payload, jsonData...)

	}

	payload = append(payload, []byte(`, `)...)
	// Query db for all videos based on user_id
	// ending result set, a list of videos, should contain:
	// locationHash + video_folder (these values are used to generate the hyperlink that the streaming service will respond to)
	// time_stamp (to show on ReelsView)
	// location_name (to show on ReelsView)
	// lat, lon (to navigate to given location in MapView when user taps on title in ReelsView)
	// username (to display on ReelsView)
	// profile_picture (to display on ReelsViews)

	payload = append(payload, []byte(`{"videos": [`)...)
	// var payload = []byte(`"{videos": [`)
	var entry interface{}

	rows, err := store.DB.Query("SELECT all_videos.video_folder, all_videos.location_hash, all_videos.like_count, IF(ISNULL(videos_liked.user_id), false, true) AS 'is_liked', all_videos.time_stamp, all_videos.user_id, users.user_name, users.photo, locations.location_name, locations.lat, locations.lon FROM all_videos JOIN users ON all_videos.user_id = users.user_id JOIN locations ON all_videos.location_hash = locations.location_hash LEFT JOIN videos_liked ON all_videos.video_folder = videos_liked.video_folder AND all_videos.location_hash = videos_liked.location_hash and videos_liked.user_id = all_videos.user_id WHERE all_videos.user_id = ? AND all_videos.is_deleted = 0 ORDER BY all_videos.time_stamp DESC", user_id_following)
	// new query
	// SELECT all_videos.video_folder, all_videos.location_hash, all_videos.like_count, all_videos.time_stamp, all_videos.user_name, users.user_name, videos_liked.user_name AS 'liked', users.photo, locations.location_name, locations.lat, locations.lon FROM all_videos JOIN users ON all_videos.user_name = users.user_name JOIN locations ON all_videos.location_hash = locations.location_hash LEFT JOIN videos_liked ON all_videos.video_folder = videos_liked.video_folder AND all_videos.location_hash = videos_liked.location_hash and videos_liked.user_name = all_videos.user_name WHERE all_videos.user_name = 'vcruky' AND all_videos.is_deleted = 0 ORDER BY all_videos.time_stamp DESC;
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Info("error in user-indexed video query SQL")
			log.Info(err)
			panic(err)
		} else {
			// NO VIDEO RESULT IN DB - send EmptyVideoStruct with default thumbnail content
			//create links
			// thumbnail_link := FALLBACK_CONTENT
			// video_link := FALLBACK_CONTENT

			// entry = EmptyVideoStruct{ThumbnailLink: thumbnail_link, VideoLink: video_link}
			// // Encode the data to JSON
			// jsonData, err := json.Marshal(entry)
			// if err != nil {
			// 	log.Info(err)
			// 	return
			// }

			// payload = append(payload, jsonData...)
			payload = append(payload, []byte(`]}`)...)
			api.RespondRaw(w, payload, http.StatusOK)
		}

	}

	defer rows.Close()
	// buf := bytes.NewBuffer(newJSON)
	// encoder := json.NewEncoder(buf)
	var id = 0
	for rows.Next() {
		var video_folder string
		var location_hash string
		var time_stamp time.Time
		var result_user_id string
		var user_name string //as an int yes, no?
		// var username string
		var like_count float64
		var video_is_liked_by_user bool
		var photo bool
		var location_name string
		var lat float64
		var lon float64

		// all_videos.video_folder,
		// all_videos.location_hash,
		// all_videos.like_count,
		// IF(ISNULL(videos_liked.user_name), false, true) AS 'is_liked',
		// all_videos.time_stamp,
		// all_videos.user_name,
		// users.user_name,
		// users.photo,
		// locations.location_name,
		// locations.lat,
		// locations.lon
		err := rows.Scan(&video_folder, &location_hash, &like_count, &video_is_liked_by_user, &time_stamp, &result_user_id, &user_name, &photo, &location_name, &lat, &lon)

		if err != nil {
			panic(err)
		} else {
			// FOUND VIDEO RESULT IN DB - send VideoStruct
			//create links
			thumbnail_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_THUMBNAIL
			video_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_VIDEO
			selfie_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_SELFIE
			var user_pic_link string
			if photo == true {
				user_pic_link = USER_CONTENT_STREAM + "/" + result_user_id + "/" + USER_PICTURE
			} else {
				user_pic_link = FALLBACK_CONTENT
			}

			videoStruct := VideoStruct{
				Id:                 id,
				ThumbnailLink:      thumbnail_link,
				VideoLink:          video_link,
				SelfieLink:         selfie_link,
				UserPicLink:        user_pic_link,
				VideoFolder:        video_folder,
				LocationHash:       location_hash,
				VideoLikeCount:     like_count,
				VideoIsLikedByUser: video_is_liked_by_user,
				TimeStamp:          time_stamp,
				UserId:             result_user_id,
				Username:           user_name,
				LocationName:       location_name,
				Lat:                lat,
				Lon:                lon}
			id = id + 1
			entry = VibecheckLocationData{Video: videoStruct, IsLiked: false}
			// Encode the data to JSON
			jsonData, err := json.Marshal(entry)
			if err != nil {
				log.Info(err)
				return
			}

			payload = append(payload, jsonData...)
			payload = append(payload, []byte(`,`)...)

		}
		log.Trace(video_folder, location_hash, like_count, video_is_liked_by_user, time_stamp, result_user_id, user_name, location_name, lat, lon)
	}

	if id > 0 {
		payload = payload[:len(payload)-1]
	}
	payload = append(payload, []byte(`]}]}`)...)

	// log.Info(string(payload))
	// Output: {"baz": [{"foo":"hello","bar":42}]}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	if err != nil {
		log.Info("Unable to connect to DB")
		response.Message = "unable to connect to database server"
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}

	//api.RespondOK(w, msg)
	api.RespondRaw(w, payload, http.StatusOK)
}

/*
Get the latest video associated with a given user_id from the database
*/
func GetUserLatestData(w http.ResponseWriter, r *http.Request) {

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// decode creds
	userFollower := &UserFollower{}
	err := json.NewDecoder(r.Body).Decode(userFollower)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user_id := userFollower.UserId
	user_id_following := userFollower.UserIdFollowing

	// Check for empty values
	if string(user_id_following) == "" {
		// just user requesting their own page
		user_id_following = user_id
	}

	log.Trace("user_id is: ", user_id)
	log.Trace("in GetUserLatestData")

	var payload = []byte(`{"data": [`)
	var user_name string
	var hasPhoto bool
	var isFollowing bool
	// SELECT users.user_name, users.photo, IF(ISNULL(user_follower.user_id), false, true) AS 'is_following' FROM users LEFT JOIN user_follower on user_follower.user_id_following = users.user_id and user_follower.user_id = '91752724-8c77-473d-85c0-0d22ace7d769' where users.user_id = '18fea441-e325-4893-9cc7-76b8ab2b7cad';
	err = store.DB.QueryRow("SELECT users.user_name, users.photo, IF(ISNULL(user_follower.user_id), false, true) AS 'is_following' FROM users LEFT JOIN user_follower on user_follower.user_id_following = users.user_id and user_follower.user_id = ? where users.user_id = ?;", user_id, user_id_following).Scan(&user_name, &hasPhoto, &isFollowing) //true or false
	// err = store.DB.QueryRow("SELECT user_name, photo from users where user_id = ?", user_id).Scan(&user_name, &hasPhoto) //true or false

	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Info("error in user-indexed video query SQL")
			log.Info(err)
			panic(err)
		} else {

			// this shouldn't be hit
			entry := VibecheckUserData{}
			// Encode the data to JSON
			jsonData, err := json.Marshal(entry)
			if err != nil {
				log.Info(err)
				return
			}
			payload = append(payload, jsonData...)
			// api.RespondRaw(w, payload, http.StatusOK)
		}

	}

	var query_user_pic_link string
	var jsonData []byte
	if hasPhoto == true {
		// return default link to streaming content
		query_user_pic_link = USER_CONTENT_STREAM + "/" + user_id + "/" + USER_PICTURE
		entry := VibecheckUserData{UserId: user_id, Username: user_name, Following: isFollowing, UserPicLink: query_user_pic_link}
		// Encode the data to JSON
		jsonData, err = json.Marshal(entry)
		if err != nil {
			log.Info(err)
			return
		}
		payload = append(payload, jsonData...)

	} else {
		// return empty string for user_pic_link
		query_user_pic_link = FALLBACK_CONTENT
		entry := VibecheckUserData{UserId: user_id, Username: user_name, Following: isFollowing, UserPicLink: query_user_pic_link}
		// Encode the data to JSON
		jsonData, err = json.Marshal(entry)
		if err != nil {
			log.Info(err)
			return
		}
		payload = append(payload, jsonData...)

	}

	payload = append(payload, []byte(`, `)...)
	// Query db for all videos based on user_id
	// ending result set, a list of videos, should contain:
	// locationHash + video_folder (these values are used to generate the hyperlink that the streaming service will respond to)
	// time_stamp (to show on ReelsView)
	// location_name (to show on ReelsView)
	// lat, lon (to navigate to given location in MapView when user taps on title in ReelsView)
	// username (to display on ReelsView)
	// profile_picture (to display on ReelsViews)

	payload = append(payload, []byte(`{"videos": [`)...)
	// var payload = []byte(`"{videos": [`)
	var entry interface{}

	rows, err := store.DB.Query("SELECT all_videos.video_folder, all_videos.location_hash, all_videos.like_count, IF(ISNULL(videos_liked.user_id), false, true) AS 'is_liked', all_videos.time_stamp, users.user_name, users.photo, locations.location_name, locations.lat, locations.lon FROM all_videos JOIN users ON all_videos.user_id = users.user_id JOIN locations ON all_videos.location_hash = locations.location_hash LEFT JOIN videos_liked ON all_videos.video_folder = videos_liked.video_folder AND all_videos.location_hash = videos_liked.location_hash and videos_liked.user_id = all_videos.user_id WHERE all_videos.user_id = ? AND all_videos.is_deleted = 0 ORDER BY all_videos.time_stamp DESC LIMIT 1", user_id)
	// new query
	// SELECT all_videos.video_folder, all_videos.location_hash, all_videos.like_count, all_videos.time_stamp, all_videos.user_name, users.user_name, videos_liked.user_name AS 'liked', users.photo, locations.location_name, locations.lat, locations.lon FROM all_videos JOIN users ON all_videos.user_name = users.user_name JOIN locations ON all_videos.location_hash = locations.location_hash LEFT JOIN videos_liked ON all_videos.video_folder = videos_liked.video_folder AND all_videos.location_hash = videos_liked.location_hash and videos_liked.user_name = all_videos.user_name WHERE all_videos.user_name = 'vcruky' AND all_videos.is_deleted = 0 ORDER BY all_videos.time_stamp DESC;
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Info("error in user-indexed video query SQL")
			log.Info(err)
			panic(err)
		} else {
			// NO VIDEO RESULT IN DB - send EmptyVideoStruct with default thumbnail content
			//create links
			// thumbnail_link := FALLBACK_CONTENT
			// video_link := FALLBACK_CONTENT

			// entry = EmptyVideoStruct{ThumbnailLink: thumbnail_link, VideoLink: video_link}
			// // Encode the data to JSON
			// jsonData, err := json.Marshal(entry)
			// if err != nil {
			// 	log.Info(err)
			// 	return
			// }

			// payload = append(payload, jsonData...)
			payload = append(payload, []byte(`]}`)...)
			api.RespondRaw(w, payload, http.StatusOK)
		}

	}

	defer rows.Close()
	// buf := bytes.NewBuffer(newJSON)
	// encoder := json.NewEncoder(buf)
	var id = 0
	for rows.Next() {
		var video_folder string
		var location_hash string
		var time_stamp time.Time
		var user_name string //as an int yes, no?
		// var username string
		var like_count float64
		var video_is_liked_by_user bool
		var photo bool
		var location_name string
		var lat float64
		var lon float64

		// all_videos.video_folder,
		// all_videos.location_hash,
		// all_videos.like_count,
		// IF(ISNULL(videos_liked.user_name), false, true) AS 'is_liked',
		// all_videos.time_stamp,
		// all_videos.user_name,
		// users.user_name,
		// users.photo,
		// locations.location_name,
		// locations.lat,
		// locations.lon
		err := rows.Scan(&video_folder, &location_hash, &like_count, &video_is_liked_by_user, &time_stamp, &user_name, &photo, &location_name, &lat, &lon)

		if err != nil {
			panic(err)
		} else {
			// FOUND VIDEO RESULT IN DB - send VideoStruct
			//create links
			thumbnail_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_THUMBNAIL
			video_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_VIDEO
			selfie_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_SELFIE
			var user_pic_link string
			if photo == true {
				user_pic_link = USER_CONTENT_STREAM + "/" + user_id + "/" + USER_PICTURE
			} else {
				user_pic_link = FALLBACK_CONTENT
			}

			videoStruct := VideoStruct{
				Id:                 id,
				ThumbnailLink:      thumbnail_link,
				VideoLink:          video_link,
				SelfieLink:         selfie_link,
				UserPicLink:        user_pic_link,
				VideoFolder:        video_folder,
				LocationHash:       location_hash,
				VideoLikeCount:     like_count,
				VideoIsLikedByUser: video_is_liked_by_user,
				TimeStamp:          time_stamp,
				UserId:             user_id,
				Username:           user_name,
				LocationName:       location_name,
				Lat:                lat,
				Lon:                lon}
			id = id + 1
			entry = VibecheckLocationData{Video: videoStruct, IsLiked: false}
			// Encode the data to JSON
			jsonData, err := json.Marshal(entry)
			if err != nil {
				log.Info(err)
				return
			}

			payload = append(payload, jsonData...)
			payload = append(payload, []byte(`,`)...)

		}
		log.Trace(video_folder, location_hash, like_count, video_is_liked_by_user, time_stamp, user_id, user_name, location_name, lat, lon)
	}

	if id > 0 {
		payload = payload[:len(payload)-1]
	}
	payload = append(payload, []byte(`]}]}`)...)

	// log.Info(string(payload))
	// Output: {"baz": [{"foo":"hello","bar":42}]}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	if err != nil {
		log.Info("Unable to connect to DB")
		response.Message = "unable to connect to database server"
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}

	//api.RespondOK(w, msg)
	api.RespondRaw(w, payload, http.StatusOK)

}

/*
Get the set of videos associated with a given location+lat+lon from the database
*/
func GetVideosByLocation(w http.ResponseWriter, r *http.Request) {
	log.Trace("in GetVideosByLocation")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// params := mux.Vars(r)
	decoder := json.NewDecoder(r.Body)
	var q LocationDataQuery
	err := decoder.Decode(&q)
	if err != nil {
		log.Panic(err)
	}
	log.Info(q.Location)

	// locationName_raw := params["locationName"]
	locationName_raw := q.Location

	log.Info("locationName_raw in GetVideosByLocation is: " + string(locationName_raw))

	// lat_raw := params["lat"]
	// lon_raw := params["lon"]
	lat_raw := q.Lat
	lon_raw := q.Lon

	// query_user_name := params["user_name"]
	user_id := q.UserId

	// latString := "39.950"

	// take string lat and lon and make into float
	lat_float, err := strconv.ParseFloat(lat_raw, 64)
	lon_float, err := strconv.ParseFloat(lon_raw, 64)
	if err != nil {
		// handle error
	}

	// expand float to have 9 decimal points as is the the db table criteria
	formattedLat := fmt.Sprintf("%.9f", lat_float)
	formattedLon := fmt.Sprintf("%.9f", lon_float)
	locationHash := GenerateLocationHashString(locationName_raw, formattedLat, formattedLon)

	// Query db for all videos based on user_id
	// ending result set, a list of videos, should contain:
	// locationHash + video_folder (these values are used to generate the hyperlink that the streaming service will respond to)
	// time_stamp (to show on ReelsView)
	// location_name (to show on ReelsView)
	// lat, lon (to navigate to given location in MapView when user taps on title in ReelsView)
	// username (to display on ReelsView)
	// profile_picture (to display on ReelsViews)

	var payload = []byte(`{"videos": [`)
	var entry interface{}

	// leaving off here
	// rows, err := store.DB.Query("SELECT all_videos.video_folder, all_videos.location_hash, all_videos.like_count, IF(ISNULL(videos_liked.user_id), false, true) AS 'is_liked', all_videos.time_stamp, users.user_name, users.photo, locations.location_name, locations.lat, locations.lon FROM all_videos JOIN users ON all_videos.user_id = users.user_id JOIN locations ON all_videos.location_hash = locations.location_hash LEFT JOIN videos_liked on all_videos.video_folder = videos_liked.video_folder AND all_videos.location_hash = videos_liked.location_hash AND all_videos.user_id = ? WHERE all_videos.location_hash = ? ORDER BY all_videos.time_stamp DESC;", user_id, locationHash)

	rows, err := store.DB.Query("SELECT all_videos.video_folder, all_videos.location_hash, all_videos.like_count, IF(ISNULL(videos_liked.user_id), false, true) AS 'is_liked', time_stamp, users.user_id, users.user_name, location_name, lat, lon FROM all_videos JOIN locations ON all_videos.location_hash = locations.location_hash JOIN users ON all_videos.user_id = users.user_id LEFT JOIN videos_liked ON all_videos.video_folder = videos_liked.video_folder AND all_videos.location_hash = videos_liked.location_hash AND videos_liked.user_id = ? WHERE all_videos.location_hash = ? AND all_videos.is_deleted = 0 ORDER BY all_videos.time_stamp DESC;", user_id, locationHash)
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Info("error in user-indexed video SQL query")
			log.Info(err)
			panic(err)
		} else {
			// NO VIDEO RESULT IN DB - send empty array
			log.Info("result set is empty in video SQL query")

			payload = append(payload, []byte(`]}`)...)
			api.RespondRaw(w, payload, http.StatusOK)
		}

	}

	// if we get here that means we have videos to send to client
	defer rows.Close()
	// buf := bytes.NewBuffer(newJSON)
	// encoder := json.NewEncoder(buf)

	// Skip the first row, because getLocationLatestVideo already did that for us
	rows.Next()
	var id = 0
	for rows.Next() {

		// Query db for video based on location
		var video_folder string
		var location_hash string
		var video_like_count float64
		var video_is_liked_by_user bool
		var time_stamp time.Time
		var result_user_id string
		var user_name string
		var photo bool
		var location_name string
		var lat float64
		var lon float64

		// log.Info("in rows.next")
		err := rows.Scan(&video_folder, &location_hash, &video_like_count, &video_is_liked_by_user, &time_stamp, &result_user_id, &user_name, &location_name, &lat, &lon)

		if err != nil {
			panic(err)
		} else {
			// FOUND VIDEO RESULT IN DB - send VideoStruct
			//create links
			thumbnail_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_THUMBNAIL
			video_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_VIDEO
			selfie_link := VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_SELFIE
			var user_pic_link string
			if photo == true {
				user_pic_link = USER_CONTENT_STREAM + "/" + result_user_id + "/" + USER_PICTURE
			} else {
				user_pic_link = FALLBACK_CONTENT
			}

			videoPayload := VideoStruct{
				Id:                 id,
				ThumbnailLink:      thumbnail_link,
				VideoLink:          video_link,
				SelfieLink:         selfie_link,
				UserPicLink:        user_pic_link,
				VideoFolder:        video_folder,
				LocationHash:       location_hash,
				VideoLikeCount:     video_like_count,
				VideoIsLikedByUser: video_is_liked_by_user,
				TimeStamp:          time_stamp,
				UserId:             result_user_id,
				Username:           user_name,
				LocationName:       location_name,
				Lat:                lat,
				Lon:                lon}

			id = id + 1
			// Encode the data to JSON
			entry = VibecheckLocationData{Video: videoPayload, IsLiked: false}
			jsonData, err := json.Marshal(entry)
			if err != nil {
				log.Info(err)
				return
			}

			payload = append(payload, jsonData...)
			payload = append(payload, []byte(`,`)...)
			// log.Info("adding result row to response")
		}
		// log.Info(video_folder, location_hash, time_stamp, user_name, username, location_name, lat, lon)
	}

	if id > 0 {
		payload = payload[:len(payload)-1]
	}
	payload = append(payload, []byte(`]}`)...)

	// log.Info(string(payload))
	// Output: {"baz": [{"foo":"hello","bar":42}]}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	if err != nil {
		log.Info("Unable to connect to DB")
		response.Message = "unable to connect to database server"
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}

	//api.RespondOK(w, msg)
	api.RespondRaw(w, payload, http.StatusOK)

}

/*
Inserts (liked_status = true) a new row if a user hearts a location
Removes (liked_status = false) if user removes heart from a location
*/
func SetFavoriteStatus(w http.ResponseWriter, r *http.Request) {
	// log.Info("in SetFavoriteStatus")
	//temp path location
	//UPLOADS_DIR := "/root/UPLOADS/"

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	params := mux.Vars(r)
	// latitude := params["latitude"]
	// longitude := params["longitude"]

	locationName_raw := params["locationName"]
	// log.Info("locationLatLon in setFavoriteStatus is: ")
	// log.Info(locationName_raw)

	lat_raw := params["lat"]
	lon_raw := params["lon"]

	user_id := params["user_id"]

	liked_status := params["liked_status"]

	// latString := "39.950"

	// take string lat and lon and make into float
	lat_float, err := strconv.ParseFloat(lat_raw, 64)
	lon_float, err := strconv.ParseFloat(lon_raw, 64)
	if err != nil {
		// handle error
	}

	// expand float to have 9 decimal points as is the the db table criteria
	formattedLat := fmt.Sprintf("%.9f", lat_float)
	formattedLon := fmt.Sprintf("%.9f", lon_float)
	locationHash := GenerateLocationHashString(locationName_raw, formattedLat, formattedLon)

	query := ""

	if liked_status == "true" {
		query = "INSERT INTO favorites (`user_id`, `location_hash`) VALUES (?, ?)"
	} else { // false, remove
		query = "DELETE FROM favorites WHERE user_id = ? AND location_hash = ?"
	}
	// query = "INSERT INTO favorites (`location_hash`, `user_name`) VALUES (?, ?) ON DUPLICATE KEY UPDATE video_folder = VALUES(video_folder)"

	result, err := store.DB.Exec(query, user_id, locationHash)
	// result := store.DB.QueryRow(query, locationHash, uID_timeStamp)
	if err != nil {
		log.Info("INSERT INTO favorites failed:")
		log.Info(err)
		return
	} else {
		// log.Info("INSERT INTO favorites worked:")
		log.Info(result)
	}

	response.Message = "Favorite Updated Successfully"
	response.Success = true
	response.Name = "favorites_update"
	// log.Info(response.Message)
	api.Respond(w, response, http.StatusOK)
}

func SetIsVideoDeletedStatus(w http.ResponseWriter, r *http.Request) {

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// just need time_stamp, user_id, and deleted_status from the client
	// query statement for sql:
	// UPDATE all_videos SET is_deleted = 1 WHERE user_id = '18fea441-e325-4893-9cc7-76b8ab2b7cad' AND time_stamp = '2023-03-10 22:22:37';

	user_id := r.FormValue("user_id")
	log.Info("time_stamp processing...")
	time_stamp_unparsed := r.FormValue("time_stamp")
	log.Info(time_stamp_unparsed)
	// layout_react_native_time_stamp := "2006-01-02 15:04:05" // Must specify the layout of the input string
	// time_stamp, err := time.Parse(layout_react_native_time_stamp, time_stamp_unparsed)
	// if err != nil {
	// 	log.Info("time_stamp parsing issue")
	// 	log.Info(err)
	// 	return
	// }
	time_stamp := strings.Replace(time_stamp_unparsed, "T", " ", 1)
	time_stamp = strings.Replace(time_stamp, "Z", "", 1)
	log.Info("After processing, time_stamp = " + time_stamp)
	deleted_status := r.FormValue("deleted_status")
	log.Info(user_id)
	log.Info(deleted_status)

	query1 := ""
	query2 := ""

	if deleted_status == "true" {
		// keep below line, will implement fully later
		// query = "UPDATE all_videos SET is_deleted = 1 WHERE user_id = ? AND time_stamp = ?;"

		query1 = "DELETE from latest_videos WHERE user_id = ? AND time_stamp = ?;"
		query2 = "DELETE from all_videos WHERE user_id = ? AND time_stamp = ?;"

	}

	result, err := store.DB.Exec(query1, user_id, time_stamp)

	if err != nil {
		log.Info(query1)
		log.Info("UPDATE latest_videos failed:")
		log.Info(err)
		return
	} else {
		log.Info(result)
	}

	result, err = store.DB.Exec(query2, user_id, time_stamp)

	if err != nil {
		log.Info(query1)
		log.Info("UPDATE all_videos failed:")
		log.Info(err)
		return
	} else {
		log.Info(result)
	}

	response.Message = "is_deleted set successfully"
	response.Success = true
	response.Name = "is_deleted_set"
	api.Respond(w, response, http.StatusOK)
}

func GetUserFavoriteLocationData(w http.ResponseWriter, r *http.Request) {
	// log.Info("in GetUserFavoriteLocations")
	//temp path location
	//UPLOADS_DIR := "/root/UPLOADS/"

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	params := mux.Vars(r)
	// latitude := params["latitude"]
	// longitude := params["longitude"]

	user_id := params["user_id"]
	// log.Info("user_name in GetUserFavoriteLocations is: ")
	// log.Info(request_user_name)

	// Query db for all favorites based on user_name
	// get latest video for each location
	var payload = []byte(`{"videos": [`)
	var entry interface{}

	//

	rows, err := store.DB.Query(
		`select latest_videos.video_folder, favorites.location_hash, all_videos.time_stamp, users.user_id, locations.location_name, locations.lat, locations.lon 
			from favorites 
		left join latest_videos on favorites.location_hash = latest_videos.location_hash 
		left join locations on favorites.location_hash = locations.location_hash 
		left join all_videos on locations.location_hash = all_videos.location_hash and latest_videos.video_folder = all_videos.video_folder
		left join users on users.user_name = favorites.user_id
		where favorites.user_id = ?;`, user_id)
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Info("error in user-indexed video SQL query")
			log.Info(err)
			panic(err)
		} else {
			// NO VIDEO RESULT IN DB - send empty array
			// log.Info("result set is empty in video SQL query")

			payload = append(payload, []byte(`]}`)...)
			api.RespondRaw(w, payload, http.StatusOK)
		}

	}

	var id = 0
	// if we get here that means we have videos to send to client
	defer rows.Close()

	for rows.Next() {
		var null_video_folder sql.NullString
		var null_location_hash sql.NullString
		var null_time_stamp sql.NullTime
		var null_user_name sql.NullString //as an int yes, no?
		// var null_username sql.NullString
		var null_location_name sql.NullString
		var null_lat sql.NullFloat64
		var null_lon sql.NullFloat64

		// log.Info("in rows.next")
		err := rows.Scan(&null_video_folder, &null_location_hash, &null_time_stamp, &null_user_name, &null_location_name, &null_lat, &null_lon)

		if err != nil {
			panic(err)
		} else {

			var video_folder = null_video_folder.String
			var location_hash = null_location_hash.String
			var time_stamp = null_time_stamp.Time
			var user_name = null_user_name.String
			// var username = null_username.String
			var location_name = null_location_name.String
			var lat = null_lat.Float64
			var lon = null_lon.Float64

			// FOUND VIDEO RESULT IN DB - send VideoStruct
			//create links
			thumbnail_link := ""
			video_link := ""
			selfie_link := ""
			user_pic_link := ""
			if video_folder != "" { // handling for favorite location doesn't have a latest video
				thumbnail_link = VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_THUMBNAIL
				video_link = VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_VIDEO
				selfie_link = VIBE_CONTENT_STREAM + "/" + location_hash + "/" + video_folder + "/" + VIBE_SELFIE
				user_pic_link = USER_CONTENT_STREAM + "/" + user_id + "/" + USER_PICTURE
			}

			videoStruct := VideoStruct{Id: id, ThumbnailLink: thumbnail_link, VideoLink: video_link, SelfieLink: selfie_link, UserPicLink: user_pic_link, VideoFolder: video_folder, LocationHash: location_hash, TimeStamp: time_stamp, UserId: user_id, Username: user_name, LocationName: location_name, Lat: lat, Lon: lon}
			id = id + 1

			entry = VibecheckLocationData{Video: videoStruct, IsLiked: true}

			// Encode the data to JSON
			jsonData, err := json.Marshal(entry)
			if err != nil {
				log.Info(err)
				return
			}

			payload = append(payload, jsonData...)
			payload = append(payload, []byte(`,`)...)
			// log.Info("adding result row to response")
			// log.Info(video_folder, location_hash, time_stamp, user_id, user_name, location_name, lat, lon)
		}

	}
	if id > 0 {
		payload = payload[:len(payload)-1]
	}
	payload = append(payload, []byte(`]}`)...)

	// log.Info(string(payload))
	// Output: {"baz": [{"foo":"hello","bar":42}]}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	if err != nil {
		log.Info("Unable to connect to DB")
		response.Message = "unable to connect to database server"
		api.Respond(w, nil, http.StatusInternalServerError)
		return
	}

	//api.RespondOK(w, msg)
	api.RespondRaw(w, payload, http.StatusOK)

}
