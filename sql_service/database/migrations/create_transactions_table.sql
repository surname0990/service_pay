CREATE TABLE transactions (
  transaction_id UUID PRIMARY KEY,
  wallet_id INT,
  value DECIMAL(10, 2) NOT NULL,
  type VARCHAR(255) NOT NULL,
  status VARCHAR(255) NOT NULL,
  transaction_time VARCHAR(255) NOT NULL,
  FOREIGN KEY (wallet_id) REFERENCES wallets(wallet_id)
);