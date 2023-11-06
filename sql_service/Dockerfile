FROM golang:1.21.3-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /app/sql_service
# RUN go build -o /app/database/migrate ./database/migrate

EXPOSE 50051

RUN apk --no-cache add postgresql-client

CMD ["/app/sql_service"]
# CMD ["/app/database/migrate", "/app/sql_service"]
