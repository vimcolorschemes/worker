FROM golang:1

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

ARG job

RUN go build -o main ./cmd/$job

ENTRYPOINT [ "/app/main" ]
