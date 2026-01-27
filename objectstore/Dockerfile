FROM golang:1.24.9-alpine

WORKDIR /app

# Copy Go modules first for caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary pointing to your main.go
RUN go build -o app ./cmd/server

# Run the built binary
CMD ["./app"]
