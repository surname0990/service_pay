package sql_service

import (
	"context"
	"fmt"
	"log"
	db "sql_service/database"
	api "sql_service/grpc/proto"
	"time"
)

type Server struct {
}

func (s *Server) GetTransactionID(ctx context.Context, req *api.TransactionId) (*api.Transaction, error) {
	transaction_id := req.TransactionId
	transaction, err := db.GetTransactionID(transaction_id)
	if err != nil {
		log.Printf("Error getting transaction from the database: %v", err)
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	response := &api.Transaction{
		TransactionId: transaction.TransactionId,
		WalletId:      int32(transaction.WalletId),
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		RequestTime:   transaction.RequestTime,
		Status:        transaction.Status,
	}
	log.Printf("Transaction Details:\nTransaction Id: %s\nWallet ID: %d\nAmount: %.2f\nTransaction Type: %s\nTransaction Time: %s\nTransaction Status: %s", transaction.TransactionId, int32(transaction.WalletId), transaction.Amount, transaction.Type, transaction.RequestTime, transaction.Status)

	return response, nil
}

func (s *Server) GetBalance(ctx context.Context, req *api.WalletIdRequest) (*api.BalanceResponse, error) {
	balance, err := db.GetBalanceByIDwallet(int(req.WalletId))
	if err != nil {
		return nil, err
	}
	// fmt.Printf("Balance Wallet ID - %d: %.2f \n", req.WalletId, balance)
	return &api.BalanceResponse{Balance: balance}, nil
}

func (s *Server) CreateTransaction(ctx context.Context, req *api.Transaction) (*api.Empty, error) {
	TransactionId := req.TransactionId
	walletID := int(req.WalletId)
	amount := req.Amount
	typeTx := req.Type
	statusTx := req.Status
	transactionTime := time.Now()
	formattedTransactionTime := transactionTime.Format("2006-01-02 15:04:05.00")

	fmt.Printf("\nNew Transaction:\nTransaction Id: %s\nWallet ID: %d\nAmount: %.2f\nTransaction Type: %s\nTransaction Status: %s\nTransaction Time: %s\n\n", TransactionId, walletID, amount, typeTx, statusTx, formattedTransactionTime)

	if err := db.NewTransaction(TransactionId, walletID, amount, typeTx, statusTx, formattedTransactionTime); err != nil {
		return nil, err
	}

	return &api.Empty{}, nil
}

func (s *Server) UpdateBalance(ctx context.Context, req *api.UpdateBalanceRequest) (*api.Empty, error) {
	walletID := int(req.WalletId)
	newBalance := req.NewBalance

	if err := db.UpdateBalance(walletID, newBalance); err != nil {
		log.Printf("Failed to update balance: %v", err)
		return nil, err
	}
	// fmt.Printf("UpdateBalance Wallet ID - %d: %.2f \n", req.WalletId, newBalance)

	return &api.Empty{}, nil
}
