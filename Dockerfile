FROM golang:1

WORKDIR /app

# Install git
RUN apt-get update
RUN apt-get install -y git

# Install nvim
RUN curl -LO https://github.com/neovim/neovim/releases/download/v0.10.4/nvim-linux-x86_64.tar.gz
RUN tar -C /opt -xzf nvim-linux-x86_64.tar.gz
ENV PATH="/opt/nvim-linux-x86_64/bin:$PATH"
RUN nvim --version

ENV TERM xterm-256color

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o main ./cmd/worker

ENTRYPOINT [ "/app/main" ]

CMD [ "import" ]
