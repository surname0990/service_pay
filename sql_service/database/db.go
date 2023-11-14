package sql_service

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
	for {
		dbPool, err = pgxpool.Connect(context.Background(), os.Getenv("POSTRGRES_STRING"))
		if err != nil {
			log.Printf("Unable to connect to database: %v. Retrying...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("DB connected")
		break
	}

	err = CreateTables()
	if err != nil {
		log.Fatalf("Error creating tables: %v", err)
	}
	err = InsertTestData()
	if err != nil {
		log.Fatalf("Error inserting test data in tables: %v", err)
	}
}

func CreateTables() error {
	if dbPool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	walletsQuery := `
        CREATE TABLE IF NOT EXISTS wallets (
            wallet_id SERIAL PRIMARY KEY,
            balance DECIMAL(10, 2) NOT NULL
        );
    `
	_, err = tx.Exec(context.Background(), walletsQuery)
	if err != nil {
		return err
	}

	transactionsQuery := `
        CREATE TABLE IF NOT EXISTS transactions (
            transaction_id UUID PRIMARY KEY,
            wallet_id INT,
            value DECIMAL(10, 2) NOT NULL,
            type VARCHAR(255) NOT NULL,
            status VARCHAR(255) NOT NULL,
            transaction_time VARCHAR(255) NOT NULL,
            FOREIGN KEY (wallet_id) REFERENCES wallets (wallet_id)
        );
    `
	_, err = tx.Exec(context.Background(), transactionsQuery)
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func InsertTestData() error {
	if dbPool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	walletsInsertQuery := `
        INSERT INTO wallets (balance) VALUES (100.00), (200.00), (75.00)
		ON CONFLICT (wallet_id) DO NOTHING;
    `
	_, err = tx.Exec(context.Background(), walletsInsertQuery)
	if err != nil {
		return err
	}

	transactionsInsertQuery := `
    INSERT INTO transactions (transaction_id, wallet_id, value, type, status, transaction_time)
    VALUES
        ('f47cbde3-98d8-47cb-a30b-1046b1f70b75', 1, 100.00, 'deposit', 'Success', '2023-11-06 12:00:00'),
        ('5e7e68f0-30d6-4d4b-8411-7f75e3b63f27', 2, 200.00, 'deposit', 'Success', '2023-11-06 12:15:00'),
        ('f4c94427-d2a9-49ac-bb5a-e7a7d2db6d4d', 3, 75.00, 'deposit', 'Success', '2023-11-06 12:30:00')
    ON CONFLICT (transaction_id) DO NOTHING;
`

	_, err = tx.Exec(context.Background(), transactionsInsertQuery)
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}

	return nil
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
		WHERE wallet_id = $2
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
		WHERE wallet_id  = $1
	`

	var balance float64
	err := dbPool.QueryRow(context.Background(), query, walletID).Scan(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func GetTransactionID(TransactionId string) (*api.Transaction, error) {
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
	// time to TimeProto
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
