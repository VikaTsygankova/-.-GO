package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"bank-service/internal/config"
	"bank-service/internal/models"
	"bank-service/internal/repository"
	"bank-service/internal/utils"
	"github.com/beevik/etree"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	gomail "gopkg.in/gomail.v2"
)

type Service struct {
	Repo *repository.Repository
	Cfg  config.Config
}

func New(r *repository.Repository, cfg config.Config) *Service { return &Service{Repo: r, Cfg: cfg} }

func (s *Service) Register(username, email, password string) (int64, error) {
	if len(password) < 6 {
		return 0, errors.New("password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	return s.Repo.CreateUser(username, email, string(hash))
}
func (s *Service) Login(email, password string) (string, error) {
	u, err := s.Repo.GetUserByEmail(email)
	if err != nil || u == nil {
		return "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}
	claims := jwt.RegisteredClaims{Subject: fmt.Sprint(u.ID), ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.Cfg.JWTSecret))
}
func (s *Service) CreateCard(userID, accountID int64) (map[string]any, error) {
	ok, err := s.Repo.OwnsAccount(userID, accountID)
	if err != nil || !ok {
		return nil, errors.New("account not found")
	}
	number := utils.GenerateCardNumber()
	expiry := time.Now().AddDate(3, 0, 0).Format("01/06")
	cvv := utils.RandomCVV()
	cvvHash, _ := bcrypt.GenerateFromPassword([]byte(cvv), bcrypt.DefaultCost)
	mac := utils.ComputeHMAC(number+expiry, []byte(s.Cfg.HMACSecret))
	id, err := s.Repo.SaveCard(userID, accountID, utils.EncryptDemo(number, s.Cfg.PGPKey), utils.EncryptDemo(expiry, s.Cfg.PGPKey), string(cvvHash), mac)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "account_id": accountID, "number": number, "expiry": expiry, "cvv": cvv, "hmac": mac}, nil
}
func (s *Service) Cards(userID int64) ([]models.Card, error) {
	raw, err := s.Repo.GetCards(userID)
	if err != nil {
		return nil, err
	}
	out := []models.Card{}
	for _, c := range raw {
		number := utils.DecryptDemo(c.EncNumber, s.Cfg.PGPKey)
		expiry := utils.DecryptDemo(c.EncExpiry, s.Cfg.PGPKey)
		out = append(out, models.Card{ID: c.ID, UserID: c.UserID, AccountID: c.AccountID, Number: number, Expiry: expiry, HMAC: c.HMAC})
	}
	return out, nil
}
func (s *Service) PayByCard(userID, cardID int64, amount float64, merchant string) error {
	acc, err := s.Repo.GetCardAccount(userID, cardID)
	if err != nil {
		return errors.New("card not found")
	}
	return s.Repo.Withdraw(userID, acc, amount, "card payment: "+merchant)
}
func (s *Service) IssueCredit(userID, accountID int64, amount float64, months int) (int64, error) {
	if months <= 0 || amount <= 0 {
		return 0, errors.New("invalid credit params")
	}
	ok, err := s.Repo.OwnsAccount(userID, accountID)
	if err != nil || !ok {
		return 0, errors.New("account not found")
	}
	rate, err := GetCentralBankRate()
	if err != nil {
		logrus.Warnf("CBR unavailable, fallback rate used: %v", err)
		rate = 21.0
	}
	payment := annuity(amount, rate, months)
	return s.Repo.CreateCredit(userID, accountID, amount, rate, months, payment)
}
func annuity(amount, annualRate float64, months int) float64 {
	m := annualRate / 100 / 12
	return math.Round((amount*m/(1-math.Pow(1+m, float64(-months))))*100) / 100
}
func (s *Service) Analytics(userID int64) (map[string]any, error) {
	inc, exp, err := s.Repo.Analytics(userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"month_income": inc, "month_expense": exp, "net": inc - exp}, nil
}
func (s *Service) Predict(userID, accountID int64, days int) (map[string]any, error) {
	if days > 365 {
		days = 365
	}
	b, err := s.Repo.Balance(userID, accountID)
	if err != nil {
		return nil, err
	}
	due, _ := s.Repo.DuePayments(userID, accountID, days)
	return map[string]any{"current_balance": b, "planned_credit_payments": due, "predicted_balance": b - due, "days": days}, nil
}

func buildSOAPRequest() string {
	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	to := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope"><soap12:Body><KeyRate xmlns="http://web.cbr.ru/"><fromDate>%s</fromDate><ToDate>%s</ToDate></KeyRate></soap12:Body></soap12:Envelope>`, from, to)
}
func GetCentralBankRate() (float64, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", "https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx", bytes.NewBuffer([]byte(buildSOAPRequest())))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://web.cbr.ru/KeyRate")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		return 0, err
	}
	els := doc.FindElements("//diffgram/KeyRate/KR")
	if len(els) == 0 {
		return 0, errors.New("rate not found")
	}
	rateEl := els[0].FindElement("./Rate")
	if rateEl == nil {
		return 0, errors.New("rate tag not found")
	}
	var rate float64
	_, err = fmt.Sscanf(rateEl.Text(), "%f", &rate)
	if err != nil {
		return 0, err
	}
	return rate + 5, nil
}
func (s *Service) SendPaymentEmail(to string, amount float64) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.Cfg.SMTPFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Payment notification")
	m.SetBody("text/html", fmt.Sprintf("<p>Payment processed: %.2f RUB</p>", amount))
	d := gomail.NewDialer(s.Cfg.SMTPHost, s.Cfg.SMTPPort, s.Cfg.SMTPUser, s.Cfg.SMTPPass)
	return d.DialAndSend(m)
}
func (s *Service) ProcessCreditPayments() { logrus.Info("scheduled credit payment processing tick") }
