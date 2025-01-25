# Vibecheck Machine Learning API Microservice
Machine Learning API Microservice for the Vibecheck App. Currently only supports Dynamic Tagging for locations. Acts as an interface between current/future Machine Learning architectures.

![example workflow](https://github.com/vibe-tech-co/vibe-check-ml-api/actions/workflows/docker-image.yml/badge.svg)

# Available API Calls
| TYPE                                   | API URL               | Description                                          |
|--------------------------------------- | --------------------  | ---------------------------------------------------- |
| $\color{orange}{\textsf{POST}}$        | ```/tags```           | Returns the predicted tags of the location           |
|                                        |                       | Form data:                                           |
|                                        |                       | `lat: <float>`                                       |
|                                        |                       | `lon: <float>`                                       |
|                                        |                       | `location_name: <string>`                            |
| $\color{orange}{\textsf{POST}}$        | ```/tagsCenterPos```  | Returns the predicted tags of the location           |
|                                        |                       | Form data:                                           |
|                                        |                       | `lat: <float>`                                       |
|                                        |                       | `lon: <float>`                                       |
|                                        |                       | `(optional) filter: <string> (CSV of tags to filter)`|
| $\color{green}{\textsf{GET}}$          | ```/test-no-auth```   | A test method for sanity checking                    |


# Folder Structure
| Folder Name      | Description                                                                                                        |
| -----------------| ------------------------------------------------------------------------------------------------------------------ |
| api              | Contains the overall logic for the different API requests mapped for the microservice                              |
| model            | Contains the models for the different API requests and model schema for storing in the database for each data type |
| store            | Contains the database logic, configuration, and setup                                                              |


# How to run

## Create
To select which config to use; `dev` or `prod`, create a `.env` file locally in the cloned repo folder with the following contents:
```
APP_ENV=dev //or prod
```
Change `APP_ENV` to which ever is needed. Additional fields will also be required in the `.env` file to run the microservice successfully. Here is a basic template of the `.env`. Customize to your liking. This template will change as the microservice matures and implements new features.

```
APP_ENV=dev
APP_PORT=:8087 // Standard port for this micro-service
OPEN_API_KEY=<key goes here>
GOOGLE_API_KEY=<key goes here>
LOG_LEVEL=trace
LOCATIONS_API_PORT=6464
LOCATIONS_API_URL=https://locations-api.vibecheck.tech/get_vibes/
NUM_WORKERS=5,k7u jn5r6et4df
METHOD_LOGGING=false
MARIA_DB_USERNAME=<user goes here>
MARIA_DB_PASSWORD=<pass goes here>
MARIA_DB_PORT=3306
MARIA_DB_HOST=127.0.0.1
MARIA_DB_NAME=vibe_db	
```

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
