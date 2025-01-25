package video

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"vibe/api"
	"vibe/config"
	mAPI "vibe/model/api"
	mDB "vibe/model/db"

	"vibe/store"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
)

// Response -> response for the util scope
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

func ChunkUploadHandler(w http.ResponseWriter, r *http.Request) {

	destination := config.CONFIGURATION.UPLOADS_LOCATION

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}
	log.Println(response.Message)
	var f *os.File
	file, uploadFile, err := r.FormFile("file")
	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "content-type should be multipart/formdata"})
		log.Println(response.Message)
		return
	}

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
		log.Println(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// get files size
	fileSize, err := strconv.Atoi(rangeAndSize[1])
	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "Missing file size in Content-Range header"})
		response.Message = "Missing file size in Content-Range header"
		log.Println(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// validate file size is within max bounds (100MB)
	if fileSize > 100*1024*1024 {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "File size should be less than 100MB"})
		response.Message = "File size should be less than 100MB"
		log.Println(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// create temp directory
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		err := os.Mkdir(destination, 0777)
		if err != nil {
			// c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating temporary directory"})
			response.Message = "Error creating temporary directory " + destination
			log.Println(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
	}

	// create/append to current file being copied
	if f == nil {
		f, err = os.OpenFile(destination+"/"+uploadFile.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating file"})
			response.Message = "Error creating file" + destination + "/" + uploadFile.Filename
			log.Println(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
	}

	// copies bytes from file chunk to the file
	if _, err := io.Copy(f, file); err != nil {
		// c.JSON(http.StatusInternalServerError, gin.H{"error": "Error writing to a file"})
		response.Message = "Error writing to a file"
		log.Println(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// close file and report status
	defer f.Close()
	if rangeMax >= fileSize-1 {
		combinedFile := destination + "/" + uploadFile.Filename

		uploadingFile, err := os.Open(combinedFile)
		if err != nil {
			response.Message = "Failed to upload file "
			log.Println(response.Message)
			api.Respond(w, response, http.StatusBadRequest)
			return
		}
		uploadingFile.Close()
		response.Message = "Uploaded File Successfully"
		response.Success = true
		response.Name = uploadFile.Filename
		log.Println(response.Message)
		api.Respond(w, response, http.StatusOK)
		return
	}

	//INSERT INTO `vibe_db`.`video` (`id`, `user_id`, `latitude`, `long`, `date_created`) VALUES ('2', 'dadfb1a2-6d6a-4c8d-baf8-6ba4a07d7d29', '2', '2', '2022-07-07 04:37:07.476');

	coll := store.MONGO_DB_CLIENT.Database("vibecheck").Collection("vibes")
	doc := bson.D{{"title", "Invisible Cities"}, {"user", "Italo Calvino"}, {"year_published", 1974}}
	result, err := coll.InsertOne(context.TODO(), doc)
	fmt.Printf("Inserted document with _id: %v\n", result.InsertedID)

	response.Message = "Uploaded Chunk Successfully"
	response.Success = true
	log.Println(response.Message)
	api.Respond(w, response, http.StatusOK)
}

/*
	Get the most recent video via the latitude/longitude from the database
*/
func GetLatestVideo(w http.ResponseWriter, r *http.Request) {

	//temp path location
	//UPLOADS_DIR := "/root/UPLOADS/"

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	params := mux.Vars(r)
	latitude := params["latitude"]
	longitude := params["longitude"]

	// Query db for video based on location
	//select * from vibe_db.video where vibe_db.video.latitude = '1' AND vibe_db.video.long = '1' ORDER BY vibe_db.video.date_created DESC LIMIT 1;
	result := store.DB.QueryRow("SELECT id, latitude, longitude, date_created, user_id, vibe_points FROM video WHERE latitude = ? AND longitude = ? ORDER BY date_created DESC LIMIT 1", string(latitude), string(longitude))
	if result != nil {
		response.Message = "unable to connect to database server"
		api.Respond(w, nil, http.StatusInternalServerError)
	}
	// Obtain stored password
	videoFields := &mDB.Video{}
	err := result.Scan(&videoFields.Id, &videoFields.Latitude, &videoFields.Longitude, &videoFields.DateCreated, &videoFields.User_Id, &videoFields.Vibe_Points)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Message = "No latest video found for this location!"
			api.Respond(w, response, http.StatusUnauthorized)
			return
		}

		response.Message = "Bad DB query"
		log.Println(err)
		api.Respond(w, response, http.StatusInternalServerError)
		return
	}

	data := mAPI.Video{
		Id:          videoFields.Id,
		User_Id:     videoFields.User_Id,
		Latitude:    videoFields.Latitude,
		Longitude:   videoFields.Longitude,
		DateCreated: videoFields.DateCreated,
		Vibe_Points: videoFields.Vibe_Points,
	}

	var videoData = json.NewEncoder(w).Encode(data)
	if err != nil {
		response.Message = "unable to encode data to Video model"
		//api.Respond(w, response, http.StatusInternalServerError)
	}

	//log.Println("Retrieved Video")
	api.RespondOK(w, videoData)
}
