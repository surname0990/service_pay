package structs

import "time"

var Config Configuration

type Configuration struct {
	LogLevel           string
	ListenAddress      string
	RabbitMQURL        string
	DBConnectionString string
}

type Transaction struct {
	ID              int
	WalletID        int
	Amount          float64
	Type            string
	Status          string
	RequestTime     time.Time
	TransactionTime time.Time
}

type Wallet struct {
	ID      int
	Balance float64
}

type DepositRequest struct {
	WalletID int     `json:"wallet_id"`
	Amount   float64 `json:"amount"`
}
