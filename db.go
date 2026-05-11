package db

import (
	"database/sql"
	_ "github.com/lib/pq"
)

func Connect(databaseURL string) (*sql.DB, error) {
	d, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := d.Ping(); err != nil {
		return nil, err
	}
	return d, nil
}

func Migrate(d *sql.DB) error {
	schema := `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    balance NUMERIC(14,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'RUB',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS cards (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id INT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    encrypted_number TEXT NOT NULL,
    encrypted_expiry TEXT NOT NULL,
    cvv_hash TEXT NOT NULL,
    hmac TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_account_id INT REFERENCES accounts(id) ON DELETE SET NULL,
    to_account_id INT REFERENCES accounts(id) ON DELETE SET NULL,
    amount NUMERIC(14,2) NOT NULL,
    type TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS credits (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id INT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    principal NUMERIC(14,2) NOT NULL,
    annual_rate NUMERIC(8,4) NOT NULL,
    months INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS payment_schedules (
    id SERIAL PRIMARY KEY,
    credit_id INT NOT NULL REFERENCES credits(id) ON DELETE CASCADE,
    due_date DATE NOT NULL,
    amount NUMERIC(14,2) NOT NULL,
    paid BOOLEAN NOT NULL DEFAULT false,
    penalty NUMERIC(14,2) NOT NULL DEFAULT 0
);`
	_, err := d.Exec(schema)
	return err
}
