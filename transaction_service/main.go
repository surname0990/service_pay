package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	pb "transaction_service/grpc/proto"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
)

// structs
type WithdrawRequest struct {
	WalletID int     `json:"wallet_id"`
	Amount   float64 `json:"amount"`
}

type DepositRequest struct {
	WalletID int     `json:"wallet_id"`
	Amount   float64 `json:"amount"`
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// gRPC Connect to sql-service
	sqlServiceConnString := os.Getenv("SQL_SERVICE_ADDRESS")
	log.Printf("Sql-service: %s\n", sqlServiceConnString)
	connSQL, err := grpc.Dial(sqlServiceConnString, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to SQL service: %v", err)
		return
	}
	defer connSQL.Close()
	sqlServiceClient := pb.NewSQLServiceClient(connSQL)

	// RabbitMQ Connect
	rabbitConnString := os.Getenv("RABBITMQ_ADDRESS")
	log.Printf("RABBIT: %s\n", rabbitConnString)
	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ")
		return
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open a channel")
		return
	}
	defer ch.Close()

	qDeposit, err := ch.QueueDeclare(
		"deposit_requests",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to declare a queue for deposit requests")
		return
	}

	qWithdraw, err := ch.QueueDeclare(
		"withdraw_requests",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to declare a queue for withdraw requests")
		return
	}

	messagesDeposit, err := ch.Consume(
		qDeposit.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to register a consumer for deposit requests")
		return
	}

	messagesWithdraw, err := ch.Consume(
		qWithdraw.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to register a consumer for withdraw requests")
		return
	}

	// Withdraw
	go func() {
		for msg := range messagesWithdraw {
			var withdrawRequest WithdrawRequest
			err := json.Unmarshal(msg.Body, &withdrawRequest)
			if err != nil {
				log.Println("Failed withdraw request:", err)
				continue
			}

			if withdrawRequest.Amount <= 0 {
				log.Println("Error: Withdraw amount must be greater than 0.")
				continue
			}

			balanceRequest := &pb.WalletIdRequest{WalletId: int32(withdrawRequest.WalletID)}
			balanceResponse, err := sqlServiceClient.GetBalance(context.Background(), balanceRequest)
			if err != nil {
				log.Printf("Failed to get balance: %v", err)
				continue
			}

			currentBalance := balanceResponse.Balance
			amountToWithdraw := float64(withdrawRequest.Amount)

			if currentBalance < amountToWithdraw {
				log.Println("Error: Insufficient balance for the withdrawal.")

				newTransaction := &pb.Transaction{
					TransactionId: uuid.New().String(),
					WalletId:      int32(withdrawRequest.WalletID),
					Amount:        amountToWithdraw,
					Type:          "withdraw",
					RequestTime:   &timestamp.Timestamp{},
					Status:        "error",
				}

				if _, err := sqlServiceClient.CreateTransaction(context.Background(), newTransaction); err != nil {
					log.Printf("Failed to create an error withdraw transaction: %v", err)
				} else {
					log.Println("New error withdraw transaction created successfully")
				}

				continue
			}

			newBalance := currentBalance - amountToWithdraw

			// Update balance
			updateBalanceRequest := &pb.UpdateBalanceRequest{
				WalletId:   int32(withdrawRequest.WalletID),
				NewBalance: newBalance,
			}

			if _, err := sqlServiceClient.UpdateBalance(context.Background(), updateBalanceRequest); err != nil {
				log.Printf("Failed to update balance: %v", err)
				continue
			}

			// Successful withdrawal transaction
			newTransaction := &pb.Transaction{
				TransactionId: uuid.New().String(),
				WalletId:      int32(withdrawRequest.WalletID),
				Amount:        amountToWithdraw,
				Type:          "withdraw",
				RequestTime:   &timestamp.Timestamp{},
				Status:        "Success",
			}

			if _, err := sqlServiceClient.CreateTransaction(context.Background(), newTransaction); err != nil {
				log.Printf("Failed to create a new withdraw transaction: %v", err)
			} else {
				log.Println("New withdraw transaction created successfully")
			}
		}
	}()

	// Deposit
	go func() {
		for msg := range messagesDeposit {
			var depositRequest DepositRequest
			err := json.Unmarshal(msg.Body, &depositRequest)
			if err != nil {
				log.Println("Failed to unmarshal deposit request:", err)
				continue
			}

			if depositRequest.Amount < 0 {
				log.Println("Error: Deposit amount cannot be negative.")
				continue
			}

			if depositRequest.Amount > 1000000000 {
				log.Println("Error: Deposit amount exceeds the limit of 1,000,000,000.")

				// Transaction "error"
				newTransaction := &pb.Transaction{
					TransactionId: uuid.New().String(),
					WalletId:      int32(depositRequest.WalletID),
					Amount:        float64(depositRequest.Amount),
					Type:          "deposit",
					RequestTime:   &timestamp.Timestamp{},
					Status:        "error",
				}

				if _, err := sqlServiceClient.CreateTransaction(context.Background(), newTransaction); err != nil {
					log.Printf("Failed to create an error deposit transaction: %v", err)
				} else {
					log.Println("New error deposit transaction created successfully")
				}
				continue
			}

			balanceRequest := &pb.WalletIdRequest{WalletId: int32(depositRequest.WalletID)}
			balanceResponse, err := sqlServiceClient.GetBalance(context.Background(), balanceRequest)
			if err != nil {
				log.Printf("Failed to get balance: %v", err)
				continue
			}

			currentBalance := balanceResponse.Balance
			amountToDeposit := float64(depositRequest.Amount)
			newBalance := currentBalance + amountToDeposit

			if newBalance > 1000000000 {
				log.Println("Error: Deposit would exceed the balance limit of 1,000,000,000.")

				// Transaction "error"
				newTransaction := &pb.Transaction{
					TransactionId: uuid.New().String(),
					WalletId:      int32(depositRequest.WalletID),
					Amount:        amountToDeposit,
					Type:          "deposit",
					RequestTime:   &timestamp.Timestamp{},
					Status:        "error",
				}

				if _, err := sqlServiceClient.CreateTransaction(context.Background(), newTransaction); err != nil {
					log.Printf("Failed to create an error deposit transaction: %v", err)
				} else {
					log.Println("New error deposit transaction created successfully")
				}

				continue
			}

			// Update balance
			updateBalanceRequest := &pb.UpdateBalanceRequest{
				WalletId:   int32(depositRequest.WalletID),
				NewBalance: newBalance,
			}

			if _, err := sqlServiceClient.UpdateBalance(context.Background(), updateBalanceRequest); err != nil {
				log.Printf("Failed to update balance: %v", err)
				continue
			}

			// Successful deposit
			newTransaction := &pb.Transaction{
				TransactionId: uuid.New().String(),
				WalletId:      int32(depositRequest.WalletID),
				Amount:        amountToDeposit,
				Type:          "deposit",
				RequestTime:   &timestamp.Timestamp{},
				Status:        "Success",
			}

			if _, err := sqlServiceClient.CreateTransaction(context.Background(), newTransaction); err != nil {
				log.Printf("Failed to create a new deposit transaction: %v", err)
			} else {
				log.Println("New deposit transaction created successfully")
			}
		}
	}()

	log.Println("Transaction service is running")
	select {}
}
