FROM golang:1

WORKDIR /app

RUN apt-get update
RUN apt-get install -y \
  vim \
  git

RUN mkdir /neovim
RUN wget -q "https://github.com/neovim/neovim/releases/download/v0.7.2/nvim-linux64.deb" -O /neovim/nvim-linux64.deb
RUN apt install /neovim/nvim-linux64.deb

ENV TERM xterm-256color

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o main ./cmd/worker

ENTRYPOINT [ "/app/main" ]

CMD [ "import" ]
