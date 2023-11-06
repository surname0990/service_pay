package main

import (
	"log"
	"net"
	"sql_service/database"
	sql_service "sql_service/grpc"
	api "sql_service/grpc/proto"

	"google.golang.org/grpc"
)

func main() {
	database.InitDB()

	server := grpc.NewServer()
	sqlService := &sql_service.Server{}

	api.RegisterSQLServiceServer(server, sqlService)

	listener, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
	log.Println("SQL service is running")
}
