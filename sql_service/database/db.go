package database

import (
	"context"
	"fmt"
	"log"
	"os"
	api "sql_service/grpc/proto"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var dbPool *pgxpool.Pool

func InitDB() {
	var err error
	dbPool, err = pgxpool.Connect(context.Background(), os.Getenv("POSTRGRES_STRING"))
	if err != nil {
		log.Fatalf("Unable to connection to database: %v", err)
	}
	log.Println("DB connected")
}

func UpdateBalance(walletID int, newBalance float64) error {
	if dbPool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	updateWalletQuery := `
		UPDATE wallets
		SET balance = $1
		WHERE id = $2
	`
	_, err = tx.Exec(context.Background(), updateWalletQuery, newBalance, walletID)
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func NewTransaction(TransactionId string, walletID int, amount float64, typeTx string, statusTx string, transactionTime string) error {
	if dbPool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %v", err)
	}

	insertTransactionQuery := `
		INSERT INTO transactions (transaction_id, wallet_id, value, type, status, transaction_time)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(context.Background(), insertTransactionQuery, TransactionId, walletID, amount, typeTx, statusTx, transactionTime)
	if err != nil {
		tx.Rollback(context.Background())
		return fmt.Errorf("unable to insert transaction record: %v", err)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return fmt.Errorf("unable to commit transaction: %v", err)
	}

	return nil
}

func GetBalanceByIDwallet(walletID int) (float64, error) {
	if dbPool == nil {
		return 0, fmt.Errorf("database pool is not initialized")
	}

	query := `
		SELECT balance
		FROM wallets
		WHERE id  = $1
	`

	var balance float64
	err := dbPool.QueryRow(context.Background(), query, walletID).Scan(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func GetTransactionID(TransactionId string) (*api.Transaction, error) {
	fmt.Printf("str: %s", TransactionId)
	if dbPool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	TransactionIdUuid, err := StrToUuid(TransactionId)
	if err != nil {
		return nil, err
	}

	query := `
        SELECT wallet_id, value, type, status, transaction_time
        FROM transactions
        WHERE transaction_id = $1
    `

	var walletID int
	var amount float64
	var typeTx string
	var statusTx string
	var transactionTimeStr string

	err = dbPool.QueryRow(context.Background(), query, TransactionIdUuid).Scan(&walletID, &amount, &typeTx, &statusTx, &transactionTimeStr)
	if err != nil {
		return nil, err
	}

	transactionIDStr := TransactionIdUuid.String()

	// string to time
	transactionTime, err := time.Parse("2006-01-02 15:04:05", transactionTimeStr)
	//time to TimeProto
	requestTimeProto := timestamppb.New(transactionTime)

	transaction := &api.Transaction{
		TransactionId: transactionIDStr,
		WalletId:      int32(walletID),
		Amount:        amount,
		Type:          typeTx,
		RequestTime:   requestTimeProto,
		Status:        statusTx,
	}

	return transaction, nil
}

// utils
func StrToUuid(Str string) (uuid.UUID, error) {
	uuid, err := uuid.Parse(Str)
	if err != nil {
		return uuid, err
	}
	return uuid, nil
}
