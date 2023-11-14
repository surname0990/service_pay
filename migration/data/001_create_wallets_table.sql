-- Write your migrate up statements here
CREATE TABLE IF NOT EXISTS wallets (
  balance DECIMAL(10, 2) NOT NULL
  wallet_id SERIAL PRIMARY KEY,
);

---- create above / drop below ----

DROP TABLE wallets