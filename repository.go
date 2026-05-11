package repository

import (
	"bank-service/internal/models"
	"database/sql"
	"errors"
)

type Repository struct{ DB *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{DB: db} }

func (r *Repository) CreateUser(username, email, hash string) (int64, error) {
	var id int64
	err := r.DB.QueryRow(`INSERT INTO users(username,email,password_hash) VALUES($1,$2,$3) RETURNING id`, username, email, hash).Scan(&id)
	return id, err
}
func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	u := &models.User{}
	err := r.DB.QueryRow(`SELECT id,username,email,password_hash,created_at FROM users WHERE email=$1`, email).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}
func (r *Repository) CreateAccount(userID int64, name string) (int64, error) {
	var id int64
	err := r.DB.QueryRow(`INSERT INTO accounts(user_id,name) VALUES($1,$2) RETURNING id`, userID, name).Scan(&id)
	return id, err
}
func (r *Repository) GetAccounts(userID int64) ([]models.Account, error) {
	rows, err := r.DB.Query(`SELECT id,user_id,name,balance,currency,created_at FROM accounts WHERE user_id=$1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Balance, &a.Currency, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *Repository) OwnsAccount(userID, accountID int64) (bool, error) {
	var n int
	err := r.DB.QueryRow(`SELECT count(*) FROM accounts WHERE id=$1 AND user_id=$2`, accountID, userID).Scan(&n)
	return n > 0, err
}
func (r *Repository) Deposit(userID, accountID int64, amount float64, desc string) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE accounts SET balance=balance+$1 WHERE id=$2 AND user_id=$3`, amount, accountID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("account not found")
	}
	_, err = tx.Exec(`INSERT INTO transactions(user_id,to_account_id,amount,type,description) VALUES($1,$2,$3,'INCOME',$4)`, userID, accountID, amount, desc)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) Withdraw(userID, accountID int64, amount float64, desc string) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE accounts SET balance=balance-$1 WHERE id=$2 AND user_id=$3 AND balance >= $1`, amount, accountID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("not enough funds or account not found")
	}
	_, err = tx.Exec(`INSERT INTO transactions(user_id,from_account_id,amount,type,description) VALUES($1,$2,$3,'EXPENSE',$4)`, userID, accountID, amount, desc)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) Transfer(userID, fromID, toID int64, amount float64) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE accounts SET balance=balance-$1 WHERE id=$2 AND user_id=$3 AND balance >= $1`, amount, fromID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("not enough funds or source account not found")
	}
	res, err = tx.Exec(`UPDATE accounts SET balance=balance+$1 WHERE id=$2`, amount, toID)
	if err != nil {
		return err
	}
	n, _ = res.RowsAffected()
	if n == 0 {
		return errors.New("target account not found")
	}
	_, err = tx.Exec(`INSERT INTO transactions(user_id,from_account_id,to_account_id,amount,type,description) VALUES($1,$2,$3,$4,'TRANSFER','transfer')`, userID, fromID, toID, amount)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) SaveCard(userID, accountID int64, encNumber, encExpiry, cvvHash, mac string) (int64, error) {
	var id int64
	err := r.DB.QueryRow(`INSERT INTO cards(user_id,account_id,encrypted_number,encrypted_expiry,cvv_hash,hmac) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, userID, accountID, encNumber, encExpiry, cvvHash, mac).Scan(&id)
	return id, err
}
func (r *Repository) GetCards(userID int64) ([]struct {
	ID, UserID, AccountID      int64
	EncNumber, EncExpiry, HMAC string
}, error) {
	rows, err := r.DB.Query(`SELECT id,user_id,account_id,encrypted_number,encrypted_expiry,hmac FROM cards WHERE user_id=$1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ID, UserID, AccountID      int64
		EncNumber, EncExpiry, HMAC string
	}
	for rows.Next() {
		var c struct {
			ID, UserID, AccountID      int64
			EncNumber, EncExpiry, HMAC string
		}
		if err := rows.Scan(&c.ID, &c.UserID, &c.AccountID, &c.EncNumber, &c.EncExpiry, &c.HMAC); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *Repository) GetCardAccount(userID, cardID int64) (int64, error) {
	var id int64
	err := r.DB.QueryRow(`SELECT account_id FROM cards WHERE id=$1 AND user_id=$2`, cardID, userID).Scan(&id)
	return id, err
}
func (r *Repository) CreateCredit(userID, accountID int64, amount, rate float64, months int, payment float64) (int64, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	if err := tx.QueryRow(`INSERT INTO credits(user_id,account_id,principal,annual_rate,months) VALUES($1,$2,$3,$4,$5) RETURNING id`, userID, accountID, amount, rate, months).Scan(&id); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`UPDATE accounts SET balance=balance+$1 WHERE id=$2 AND user_id=$3`, amount, accountID, userID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`INSERT INTO transactions(user_id,to_account_id,amount,type,description) VALUES($1,$2,$3,'INCOME','credit issued')`, userID, accountID, amount); err != nil {
		return 0, err
	}
	for i := 1; i <= months; i++ {
		if _, err := tx.Exec(`INSERT INTO payment_schedules(credit_id,due_date,amount) VALUES($1,current_date + ($2 || ' months')::interval,$3)`, id, i, payment); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
func (r *Repository) GetSchedule(userID, creditID int64) ([]models.PaymentSchedule, error) {
	rows, err := r.DB.Query(`SELECT ps.id,ps.credit_id,ps.due_date::text,ps.amount,ps.paid,ps.penalty FROM payment_schedules ps JOIN credits c ON c.id=ps.credit_id WHERE c.user_id=$1 AND c.id=$2 ORDER BY ps.due_date`, userID, creditID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PaymentSchedule
	for rows.Next() {
		var p models.PaymentSchedule
		if err := rows.Scan(&p.ID, &p.CreditID, &p.DueDate, &p.Amount, &p.Paid, &p.Penalty); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *Repository) Analytics(userID int64) (income, expense float64, err error) {
	_ = r.DB.QueryRow(`SELECT COALESCE(sum(amount),0) FROM transactions WHERE user_id=$1 AND type='INCOME' AND created_at >= date_trunc('month', now())`, userID).Scan(&income)
	_ = r.DB.QueryRow(`SELECT COALESCE(sum(amount),0) FROM transactions WHERE user_id=$1 AND type IN ('EXPENSE','TRANSFER') AND created_at >= date_trunc('month', now())`, userID).Scan(&expense)
	return income, expense, nil
}
func (r *Repository) Balance(userID, accountID int64) (float64, error) {
	var b float64
	err := r.DB.QueryRow(`SELECT balance FROM accounts WHERE user_id=$1 AND id=$2`, userID, accountID).Scan(&b)
	return b, err
}
func (r *Repository) DuePayments(userID, accountID int64, days int) (float64, error) {
	var s float64
	err := r.DB.QueryRow(`SELECT COALESCE(sum(ps.amount+ps.penalty),0) FROM payment_schedules ps JOIN credits c ON c.id=ps.credit_id WHERE c.user_id=$1 AND c.account_id=$2 AND ps.paid=false AND ps.due_date <= current_date + ($3 || ' days')::interval`, userID, accountID, days).Scan(&s)
	return s, err
}
