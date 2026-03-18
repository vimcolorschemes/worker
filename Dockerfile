FROM golang:1.23

WORKDIR /app

# Install git
RUN apt-get update && apt-get install -y --no-install-recommends git && rm -rf /var/lib/apt/lists/*

# Install CGO build dependencies
RUN apt-get install -y gcc libc6-dev
ENV CGO_ENABLED=1

# Install nvim
RUN curl -LO https://github.com/neovim/neovim/releases/download/v0.10.4/nvim-linux-x86_64.tar.gz
RUN tar -C /opt -xzf nvim-linux-x86_64.tar.gz
ENV PATH="/opt/nvim-linux-x86_64/bin:$PATH"
RUN nvim --version

ENV TERM=xterm-256color

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o main ./cmd/worker

ENTRYPOINT [ "/app/main" ]

CMD [ "import" ]
