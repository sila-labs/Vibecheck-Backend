# Vibecheck Go Template API
This is a template API used as an example for any new Go API we may spin up in the future.

<!-- ![example workflow](https://github.com/vibe-tech-co/vibe-check-core-api/actions/workflows/docker-image.yml/badge.svg) -->

# Available API Calls
```TEMPLATE: put all available api calls for this service here and what is needed to call them```
| TYPE                                   | API URL               | Description                                          |
|--------------------------------------- | --------------------  | ---------------------------------------------------- |
| $\color{green}{\textsf{GET}}$          | ```/test-no-auth```   | A test method for sanity checking                    |
| $\color{orange}{\textsf{POST}}$        | ```<url here>```      | Example post method with expected form value data    |
|                                        |                       | Form data:                                           |
|                                        |                       | `<field name>: <float>`                              |
|                                        |                       | `<field name>: <float>`                              |
| $\color{cyan}{\textsf{PUT}}$          | ```<url here>```      | Example PUT method                                   |
| $\color{grey}{\textsf{PATCH}}$        | ```<url here>```      | Example PATCH method                                 |
| $\color{red}{\textsf{DELETE}}$       | ```<url here>```      | Example DELETE method                                |



# Folder Structure
```TEMPLATE: detail folder structure of project. May not change for most services```
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

```TEMPLATE: add an example .env to run your service, please keep this up to date```
```
APP_ENV=dev
APP_PORT=:8099 // Standard port for this micro-service
LOG_LEVEL=trace
METHOD_LOGGING=false
MARIA_DB_USERNAME=<user goes here>
MARIA_DB_PASSWORD=<pass goes here>
MARIA_DB_PORT=3306
MARIA_DB_HOST=127.0.0.1
MARIA_DB_NAME=vibe_db	
```

```TEMPLATE: these shoudln't change across each service unless you include something new like such as a cache that needs to be setup before service is launched. MAKE SURE YOU HAVE A .ENV FILE CREATED BEFORE YOU RUN THE SERVICE, otherwise it wont run.```
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
