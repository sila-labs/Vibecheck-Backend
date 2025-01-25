package notifications

import (
	"context"
	"net/http"
	"strings"
	"vibe/api"

	log "github.com/sirupsen/logrus"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

// Response -> response for the util scope
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

var ctx context.Context
var cancel context.CancelFunc
var app *firebase.App

var client *messaging.Client

func Setup() {
	opt := option.WithCredentialsFile("serviceAccountKey.json")
	ctx, cancel = context.WithCancel(context.Background())

	// Initialize a new FCM app object
	var err error // So it wont complain about undeclared err var
	app, err = firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal("Error initializing firebase app: ", err)
	}

	// Get a messaging client
	client, err = app.Messaging(ctx) // Do not need to cleanup, done automatically
	if err != nil {
		log.Fatal("There was an error getting a messaging client: ", err)
		return
	}
}

// Handler for unsubscribing device(s) to a specified topic (Identical to subcribe)
func UnsubscribeDevicesToChannel(w http.ResponseWriter, r *http.Request) {
	log.Info("In unsubscribe devices to topic handler -------------------------")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// Read in form values
	device_tokens := r.FormValue("device_tokens")
	topic := r.FormValue("topic")

	// These params are required
	if device_tokens == "" {
		response.Message = "Missing required query parameter: device_tokens"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	if topic == "" {
		response.Message = "Missing required query parameter: topic"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	device_tokens_split := strings.Split(device_tokens, ",")
	if len(device_tokens_split) > 1000 {
		response.Message = "Too many device tokens, please limit to < 1000 tokens per call"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
	}

	log.Info("Processed form values")

	// Send unsubscribe message to FCM
	firebase_response, err := client.UnsubscribeFromTopic(r.Context(), device_tokens_split, topic)
	log.Trace("Raw firebase response: ", firebase_response)
	if err != nil {
		response.Message = "There was an error sending unsubscribe message to FCM: " + err.Error()
		log.Error(response.Message)
		api.Respond(w, response, http.StatusBadGateway)
		return
	}
	log.Trace("Devices Unsubscribed: ", device_tokens_split)
	log.Info("Unsubscribed devices to ", topic)

	api.RespondOK(w, Response{Success: true, Message: "Devices unsubscribed from topic: " + topic})
}

// Handler for subscribing device(s) to a specified topic (Identical to unsubcribe)
func SubscribeDevicesToTopic(w http.ResponseWriter, r *http.Request) {
	log.Info("In subscribe devices to topic handler -------------------------")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// Read in form values
	device_tokens := r.FormValue("device_tokens")
	topic := r.FormValue("topic")

	// These params are required
	if device_tokens == "" {
		response.Message = "Missing required query parameter: device_tokens"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	if topic == "" {
		response.Message = "Missing required query parameter: topic"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	device_tokens_split := strings.Split(device_tokens, ",")
	if len(device_tokens_split) > 1000 {
		response.Message = "Too many device tokens, please limit to < 1000 tokens per call"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
	}

	log.Info("Processed form values")

	// Send subscribe message to FCM
	firebase_response, err := client.SubscribeToTopic(r.Context(), device_tokens_split, topic)
	log.Trace("Raw firebase response: ", firebase_response)
	if err != nil {
		response.Message = "There was an error sending subscribe message to FCM: " + err.Error()
		log.Error(response.Message)
		api.Respond(w, response, http.StatusBadGateway)
		return
	}
	log.Trace("Devices subcribed: ", device_tokens_split)
	log.Info("Subscribed devices to ", topic)

	api.RespondOK(w, Response{Success: true, Message: "Devices subscribed to topic: " + topic})
}

// Sends a notification to the specified device
func SendNotificationToDevice(w http.ResponseWriter, r *http.Request) {
	log.Info("In send notification handler -------------------------")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// Read in form values
	device_token := r.FormValue("device_token")
	title := r.FormValue("title")
	body := r.FormValue("body")

	// These params are required
	if device_token == "" {
		response.Message = "Missing required query parameter: device_token"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// if title == "" {
	// 	response.Message = "Missing required query parameter: title"
	// 	log.Warning(response.Message)
	// 	api.Respond(w, response, http.StatusBadRequest)
	// 	return
	// }

	log.Info("Processed form values")

	// Create notification and include contents specified
	notification_message := &messaging.Message{
		Token: device_token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}

	log.Trace("Raw notification message: ", notification_message)

	// Send message to FCM to distribute to device
	firebase_response, err := client.Send(r.Context(), notification_message)
	log.Trace("Raw firebase response: ", firebase_response)
	if err != nil {
		response.Message = "There was an error sending message to FCM: " + err.Error()
		log.Error(response.Message)
		api.Respond(w, response, http.StatusBadGateway)
		return
	}

	api.RespondOK(w, Response{Success: true, Message: "Push notification sent to device"})
}

// Sends a notification to the specified topic
func SendNotificationToTopic(w http.ResponseWriter, r *http.Request) {
	log.Info("In send notification handler -------------------------")

	response := &Response{
		Success: false,
		Message: "none",
		Name:    "",
	}

	// Read in form values
	topic := r.FormValue("topic")
	title := r.FormValue("title")
	body := r.FormValue("body")

	// These params are required
	if topic == "" {
		response.Message = "Missing required query parameter: topic"
		log.Warning(response.Message)
		api.Respond(w, response, http.StatusBadRequest)
		return
	}

	// if title == "" {
	// 	response.Message = "Missing required query parameter: title"
	// 	log.Warning(response.Message)
	// 	api.Respond(w, response, http.StatusBadRequest)
	// 	return
	// }

	log.Info("Processed form values")

	// Create notification and include contents specified
	notification_message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}

	log.Trace("Raw notification message: ", notification_message)

	// Send message to FCM to distribute to topic
	firebase_response, err := client.Send(r.Context(), notification_message)
	log.Trace("Raw firebase response: ", firebase_response)
	if err != nil {
		response.Message = "There was an error sending message to FCM: " + err.Error()
		log.Error(response.Message)
		api.Respond(w, response, http.StatusBadGateway)
		return
	}

	api.RespondOK(w, Response{Success: true, Message: "Push notification sent to topic"})
}

// Cleans up any captured resources and frees them
func Cleanup() {
	cancel()
}
