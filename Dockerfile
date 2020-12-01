FROM golang:1

WORKDIR /app

RUN apt-get update && apt-get install -y \
  vim \
  git

ENV TERM xterm-256color

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o main ./cmd/worker

ENTRYPOINT [ "/app/main" ]

CMD [ "import" ]
