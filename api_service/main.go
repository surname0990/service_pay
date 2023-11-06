package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	pb "api_service/grpc/proto"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
)

type DepositRequest struct {
	WalletID int     `json:"wallet_id"`
	Amount   float64 `json:"amount"`
}

type WithdrawRequest struct {
	WalletID int     `json:"wallet_id"`
	Amount   float64 `json:"amount"`
}

type Transaction struct {
	TransactionID   string  `json:"transaction_id"`
	WalletID        int     `json:"wallet_id"`
	Value           float64 `json:"value"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	TransactionTime string  `json:"transaction_time"`
}

func main() {
	r := gin.Default()

	r.GET("/get-transaction/:id", getTransactionHandler)
	r.POST("/deposit", depositHandler)
	r.POST("/withdraw", withdrawHandler)

	err := r.Run(":8080")
	if err != nil {
		log.Fatal("HTTP server failed to start")
	}
}

func getTransactionHandler(c *gin.Context) {
	id := c.Param("id")

	fmt.Printf("Received ID: %s\n", id)

	transaction, err := getTransactionFromService(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction"})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func getTransactionFromService(id string) (Transaction, error) {
	connSQL, err := grpc.Dial("localhost:50051", grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Failed to connect to SQL service: %v", err)
		return Transaction{}, err
	}
	defer connSQL.Close()

	sqlServiceClient := pb.NewSQLServiceClient(connSQL)
	ctx := context.Background()

	request := &pb.TransactionId{
		TransactionId: id,
	}

	response, err := sqlServiceClient.GetTransactionID(ctx, request)
	if err != nil {
		return Transaction{}, err
	}

	result := Transaction{
		TransactionID:   response.TransactionId,
		WalletID:        int(response.WalletId),
		Value:           response.Amount,
		Type:            response.Type,
		Status:          response.Status,
		TransactionTime: response.GetRequestTime().String(),
	}

	return result, nil
}

func depositHandler(c *gin.Context) {
	var depositRequest DepositRequest

	if err := c.ShouldBindJSON(&depositRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := publishDeposit(depositRequest); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish deposit request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deposit request sent to RabbitMQ"})
}

func publishDeposit(depositRequest DepositRequest) error {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"deposit_requests",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	body, err := json.Marshal(depositRequest)
	if err != nil {

		return err
	}

	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return err
	}

	return nil
}

func withdrawHandler(c *gin.Context) {
	var withdrawRequest WithdrawRequest

	if err := c.ShouldBindJSON(&withdrawRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := publishWithdraw(withdrawRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish withdraw request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Withdraw request sent to RabbitMQ"})
}

func publishWithdraw(withdrawRequest WithdrawRequest) error {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")

	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"withdraw_requests",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	body, err := json.Marshal(withdrawRequest)
	if err != nil {
		return err
	}

	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return err
	}

	return nil
}
