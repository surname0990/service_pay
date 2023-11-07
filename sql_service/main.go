package main

import (
	"log"
	"net"
	"os"
	"sql_service/database"
	sql_service "sql_service/grpc"
	api "sql_service/grpc/proto"

	"github.com/joho/godotenv"

	"google.golang.org/grpc"
)

func main() {
	InitConfig()
	database.InitDB()
	ListenerGrpcServer()
}

func InitConfig() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func ListenerGrpcServer() {
	server := grpc.NewServer()
	sqlService := &sql_service.Server{}

	api.RegisterSQLServiceServer(server, sqlService)

	listener, err := net.Listen("tcp", (os.Getenv("SQL_SERVICE_ADDRESS")))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
