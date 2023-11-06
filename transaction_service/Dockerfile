FROM golang:1.21.3-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o transaction_service

CMD ["./transaction_service"]
