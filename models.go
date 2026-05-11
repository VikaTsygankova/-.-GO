package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Account struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

type Card struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	AccountID int64     `json:"account_id"`
	Number    string    `json:"number,omitempty"`
	Expiry    string    `json:"expiry,omitempty"`
	HMAC      string    `json:"hmac,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Transaction struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	FromAccountID *int64    `json:"from_account_id,omitempty"`
	ToAccountID   *int64    `json:"to_account_id,omitempty"`
	Amount        float64   `json:"amount"`
	Type          string    `json:"type"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

type Credit struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	AccountID  int64     `json:"account_id"`
	Principal  float64   `json:"principal"`
	AnnualRate float64   `json:"annual_rate"`
	Months     int       `json:"months"`
	CreatedAt  time.Time `json:"created_at"`
}

type PaymentSchedule struct {
	ID       int64   `json:"id"`
	CreditID int64   `json:"credit_id"`
	DueDate  string  `json:"due_date"`
	Amount   float64 `json:"amount"`
	Paid     bool    `json:"paid"`
	Penalty  float64 `json:"penalty"`
}
