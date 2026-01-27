FROM golang:1.24.9-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o app
CMD ["./app"]
