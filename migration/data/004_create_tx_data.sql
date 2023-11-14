INSERT INTO transactions (transaction_id, wallet_id, value, type, status, transaction_time)
VALUES
    ('f47cbde3-98d8-47cb-a30b-1046b1f70b75', 1, 100.00, 'deposit', 'Success', '2023-11-06 12:00:00'),
    ('5e7e68f0-30d6-4d4b-8411-7f75e3b63f27', 2, 200.00, 'deposit', 'Success', '2023-11-06 12:15:00'),
    ('f4c94427-d2a9-49ac-bb5a-e7a7d2db6d4d', 3, 75.00, 'deposit', 'Success', '2023-11-06 12:30:00');

---- create above / drop below ----

DROP TABLE transactions
