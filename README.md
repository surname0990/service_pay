
Services:
- api-service: Сервис принимает HTTP-запросы от клиентов для создания и получения данных о транзакциях (grpc-->sql-service) и отправляет в очереди (ввод, вывод средств) RabbitMQ 
  

- transaction-service: Обрабатывает данные о транзакциях (ввод, вывод), получая их из RabbitMQ, запрашивает данные баланса sql-service для осуществления транзакции(проверка баланса) и отправляет данные о готовой транзакции в sql-service по gRPC


- sql-service: Cервис для обмена данными между transaction-service и PostgreSQL. 

*RabbitMQ: 
- для обмена данными: api-service -- Transaction Service

*Postrges: 
- для обмена данными: sql-service

*gRPC:  
- для обмена данными: api-service -- sql-service

- для обмена данными: transaction-service -- sql-service


Must create tables:

- TABLE wallets 

  balance DECIMAL(10, 2) NOT NULL

  wallet_id SERIAL PRIMARY KEY, 


- TABLE transactions 

  transaction_id UUID PRIMARY KEY,

  wallet_id INT,  

  value DECIMAL(10, 2) NOT NULL,

  type VARCHAR(255) NOT NULL,

  status VARCHAR(255) NOT NULL,

  transaction_time VARCHAR(255) NOT NULL, 

  FOREIGN KEY (wallet_id) REFERENCES wallets(id)
  

Command:

- docker build -t api_service .

- docker build -t transaction_service .

- docker build -t sql_service .

- docker-compose build

- docker-compose up 




