package main

import (
	"bank-service/internal/config"
	"bank-service/internal/db"
	"bank-service/internal/handler"
	"bank-service/internal/middleware"
	"bank-service/internal/repository"
	"bank-service/internal/service"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func main() {
	cfg := config.Load()
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		logrus.Fatalf("database connection failed: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		logrus.Fatalf("migration failed: %v", err)
	}
	repo := repository.New(database)
	svc := service.New(repo, cfg)
	h := handler.New(svc)

	r := mux.NewRouter()
	r.Use(loggingMiddleware)
	r.HandleFunc("/register", h.Register).Methods("POST")
	r.HandleFunc("/login", h.Login).Methods("POST")

	auth := r.PathPrefix("/").Subrouter()
	auth.Use(middleware.Auth(cfg.JWTSecret))
	auth.HandleFunc("/accounts", h.CreateAccount).Methods("POST")
	auth.HandleFunc("/accounts", h.Accounts).Methods("GET")
	auth.HandleFunc("/accounts/{id}/deposit", h.Deposit).Methods("POST")
	auth.HandleFunc("/accounts/{id}/withdraw", h.Withdraw).Methods("POST")
	auth.HandleFunc("/transfer", h.Transfer).Methods("POST")
	auth.HandleFunc("/cards", h.CreateCard).Methods("POST")
	auth.HandleFunc("/cards", h.Cards).Methods("GET")
	auth.HandleFunc("/cards/pay", h.Pay).Methods("POST")
	auth.HandleFunc("/credits", h.CreateCredit).Methods("POST")
	auth.HandleFunc("/credits/{creditId}/schedule", h.Schedule).Methods("GET")
	auth.HandleFunc("/analytics", h.Analytics).Methods("GET")
	auth.HandleFunc("/accounts/{accountId}/predict", h.Predict).Methods("GET")

	go func() {
		ticker := time.NewTicker(12 * time.Hour)
		for range ticker.C {
			svc.ProcessCreditPayments()
		}
	}()
	logrus.Infof("server started on http://localhost:%s", cfg.ServerPort)
	logrus.Fatal(http.ListenAndServe(":"+cfg.ServerPort, r))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logrus.Infof("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
