FROM golang:alpine
WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .
RUN go build

EXPOSE 8086

CMD ["go","run","vibe"]