FROM golang:1

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o main ./cmd/worker

ENTRYPOINT [ "/app/main" ]

CMD [ "import" ]
