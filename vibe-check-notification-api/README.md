# Vibecheck Notification API
This service is responsible for remote push notifications. Can call the API to send notifications to device(s).

<!-- ![example workflow](https://github.com/vibe-tech-co/vibe-check-core-api/actions/workflows/docker-image.yml/badge.svg) -->

# Available API Calls
| TYPE                                   | API URL                               | Description                                                                 |
|--------------------------------------- | ------------------------------------- | --------------------------------------------------------------------------- |
| $\color{green}{\textsf{GET}}$          | ```/test-no-auth```                   | A test method for sanity checking                                           |
| $\color{orange}{\textsf{POST}}$        | ```/send-notification-to-device```    | Sends a notification to a device give a specifed device_token               |
|                                        |                                       | Form data:                                                                  |
|                                        |                                       | `device_token: <string>`                                                    |
|                                        |                                       | `(optional) title: <string>`                                                |
|                                        |                                       | `(optional) body: <string>`                                                 |
| $\color{orange}{\textsf{POST}}$        | ```/send-notification-to-topic```     | Sends a notification to a specified topic                                   |
|                                        |                                       | Form data:                                                                  |
|                                        |                                       | `topic: <string>`                                                           |
|                                        |                                       | `(optional) title: <string>`                                                |
|                                        |                                       | `(optional body: <string>`                                                  |
| $\color{orange}{\textsf{POST}}$        | ```/subscribe-devices-to-topic```     | Subscribes device token(s) to a topic                                       |
|                                        |                                       | Form data:                                                                  |
|                                        |                                       | `device_tokens: <float> (CSV of device tokens, can also be 1 device token)` |
|                                        |                                       | `topic: <string>`                                                           |
| $\color{orange}{\textsf{POST}}$        | ```/unsubscribe-devices-from-topic``` | Unsubscribes device token(s) from a topic                                   |
|                                        |                                       | Form data:                                                                  |
|                                        |                                       | `device_tokens: <float> (CSV of device tokens, can also be 1 device token)` |
|                                        |                                       | `topic: <string>`                                                           |



# Folder Structure
| Folder Name      | Description                                                                                                        |
| -----------------| ------------------------------------------------------------------------------------------------------------------ |
| api              | Contains the overall logic for the different API requests mapped for the microservice                              |
| model            | Contains the models for the different API requests and model schema for storing in the database for each data type |
| store            | Contains the database logic, configuration, and setup                                                              |


# How to run

## Create
### .env Creation
To select which config to use; `dev` or `prod`, create a `.env` file locally in the cloned repo folder with the following contents:
```
APP_ENV=dev //or prod
```
Change `APP_ENV` to which ever is needed. Additional fields will also be required in the `.env` file to run the microservice successfully. Here is a basic template of the `.env`. Customize to your liking. This template will change as the microservice matures and implements new features.

```
APP_ENV=dev
APP_PORT=:8088 // Standard port for this microservice
LOG_LEVEL=trace
METHOD_LOGGING=false
```

### serviceAccountKey.json Creation
This file holds the service account keys (private) for the service to interface FCM (Firebase Cloud Messaging). These do not have a template and are only created once at service account setup. You will need to ask for the service account keys or create a new service account if you would like to use the service locally where the keys are not stored.

## Build
```
go build
```
## Run
```
go run vibe
```
or if you dont want to build
```
go run main.go
```
## (Optional) Update package checksums and download dependencies
```
go mod tidy
``` 
