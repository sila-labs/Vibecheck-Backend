FROM golang:alpine
WORKDIR /usr/src/app


# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .
#RUN apt-get -y update && apt-get -y upgrade && apt-get install -y --no-install-recommends ffmpeg
RUN go build

CMD ["go","run","vibe"]