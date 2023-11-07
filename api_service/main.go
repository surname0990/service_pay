package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	pb "api_service/grpc/proto"

	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/joho/godotenv"
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
type Api struct {
	SQLServiceConn *grpc.ClientConn
	RabbitConn     *amqp.Connection
}

func main() {
	InitConfig()
	api, err := NewApi(os.Getenv("SQL_SERVICE_ADDRESS"), os.Getenv("RABBITMQ_ADDRESS"))
	if err != nil {
		log.Fatalf("Failed to init api-service: %v", err)
		return
	}
	defer api.Close()

	r := gin.Default()

	r.GET("/get-transaction/:id", api.getTransactionHandler)
	r.POST("/deposit", api.depositHandler)
	r.POST("/withdraw", api.withdrawHandler)

	httpPort := os.Getenv("HTTP_PORT")
	err = r.Run(httpPort)
	if err != nil {
		log.Fatal("HTTP server failed to start")
	}
}

// API
func (a *Api) getTransactionHandler(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Received ID: %s\n", id)

	transaction, err := a.getTransactionFromService(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction"})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (a *Api) getTransactionFromService(id string) (Transaction, error) {
	// defer a.SQLServiceConn.Close() //
	sqlServiceClient := pb.NewSQLServiceClient(a.SQLServiceConn)
	ctx := context.Background()
	request := &pb.TransactionId{
		TransactionId: id,
	}

	response, err := sqlServiceClient.GetTransactionID(ctx, request)
	if err != nil {
		log.Println("Error request GetTransactionID:", err)
		return Transaction{}, err
	}

	formattedTime, err := ProtoTimestampToFormattedTime(response.GetRequestTime())
	if err != nil {
		log.Println("Error FormattedTime:", err)
		return Transaction{}, err
	}

	result := Transaction{
		TransactionID:   response.TransactionId,
		WalletID:        int(response.WalletId),
		Value:           response.Amount,
		Type:            response.Type,
		Status:          response.Status,
		TransactionTime: formattedTime,
	}

	return result, nil
}

func (a *Api) depositHandler(c *gin.Context) {
	var depositRequest DepositRequest

	if err := c.ShouldBindJSON(&depositRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := a.publishDeposit(depositRequest); err != nil {
		log.Println("Error publishDeposit:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish deposit request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deposit request sent to RabbitMQ"})
}

func (a *Api) publishDeposit(depositRequest DepositRequest) error {
	ch, err := a.RabbitConn.Channel()
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

func (a *Api) withdrawHandler(c *gin.Context) {
	var withdrawRequest WithdrawRequest

	if err := c.ShouldBindJSON(&withdrawRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := a.publishWithdraw(withdrawRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish withdraw request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Withdraw request sent to RabbitMQ"})
}

func (a *Api) publishWithdraw(withdrawRequest WithdrawRequest) error {
	ch, err := a.RabbitConn.Channel()
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

// utils
func ProtoTimestampToFormattedTime(ts *timestamp.Timestamp) (string, error) {
	seconds := int64(ts.GetSeconds())
	nanos := int64(ts.GetNanos())

	duration := time.Second*time.Duration(seconds) + time.Nanosecond*time.Duration(nanos)
	t := time.Unix(0, duration.Nanoseconds())
	formattedTime := t.Format("2006-01-02 15:04:05.00")

	return formattedTime, nil
}

func InitConfig() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func NewApi(sqlServiceConnString, rabbitConnString string) (*Api, error) {
	sqlServiceConn, err := grpc.Dial(sqlServiceConnString, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	log.Println("Sql-service Connected.")

	rabbitConn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		// sqlServiceConn.Close()
		log.Println(err)
		return nil, err
	}
	log.Println("Rabbit Connected.\n")

	return &Api{
		SQLServiceConn: sqlServiceConn,
		RabbitConn:     rabbitConn,
	}, nil
}
func (a *Api) Close() {
	a.SQLServiceConn.Close()
	a.RabbitConn.Close()

}
