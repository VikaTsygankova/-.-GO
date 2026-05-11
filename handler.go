package handler

import (
	"bank-service/internal/middleware"
	"bank-service/internal/service"
	"bank-service/internal/utils"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

type Handler struct{ S *service.Service }

func New(s *service.Service) *Handler { return &Handler{S: s} }
func userID(r *http.Request) int64 {
	v, _ := r.Context().Value(middleware.UserIDKey).(string)
	id, _ := strconv.ParseInt(v, 10, 64)
	return id
}
func decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct{ Username, Email, Password string }
	if err := decode(r, &req); err != nil {
		utils.Error(w, 400, "bad json")
		return
	}
	id, err := h.S.Register(req.Username, req.Email, req.Password)
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 201, map[string]any{"id": id})
}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email, Password string }
	if err := decode(r, &req); err != nil {
		utils.Error(w, 400, "bad json")
		return
	}
	t, err := h.S.Login(req.Email, req.Password)
	if err != nil {
		utils.Error(w, 401, err.Error())
		return
	}
	utils.JSON(w, 200, map[string]string{"token": t})
}
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name string }
	_ = decode(r, &req)
	if req.Name == "" {
		req.Name = "Account"
	}
	id, err := h.S.Repo.CreateAccount(userID(r), req.Name)
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 201, map[string]any{"id": id})
}
func (h *Handler) Accounts(w http.ResponseWriter, r *http.Request) {
	a, err := h.S.Repo.GetAccounts(userID(r))
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, a)
}
func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	var req struct{ Amount float64 }
	_ = decode(r, &req)
	err := h.S.Repo.Deposit(userID(r), id, req.Amount, "deposit")
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, map[string]bool{"ok": true})
}
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	var req struct{ Amount float64 }
	_ = decode(r, &req)
	err := h.S.Repo.Withdraw(userID(r), id, req.Amount, "withdraw")
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, map[string]bool{"ok": true})
}
func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromAccountID int64   `json:"from_account_id"`
		ToAccountID   int64   `json:"to_account_id"`
		Amount        float64 `json:"amount"`
	}
	_ = decode(r, &req)
	err := h.S.Repo.Transfer(userID(r), req.FromAccountID, req.ToAccountID, req.Amount)
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, map[string]bool{"ok": true})
}
func (h *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID int64 `json:"account_id"`
	}
	_ = decode(r, &req)
	c, err := h.S.CreateCard(userID(r), req.AccountID)
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 201, c)
}
func (h *Handler) Cards(w http.ResponseWriter, r *http.Request) {
	c, err := h.S.Cards(userID(r))
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, c)
}
func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CardID   int64   `json:"card_id"`
		Amount   float64 `json:"amount"`
		Merchant string  `json:"merchant"`
	}
	_ = decode(r, &req)
	err := h.S.PayByCard(userID(r), req.CardID, req.Amount, req.Merchant)
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, map[string]bool{"ok": true})
}
func (h *Handler) CreateCredit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID int64   `json:"account_id"`
		Amount    float64 `json:"amount"`
		Months    int     `json:"months"`
	}
	_ = decode(r, &req)
	id, err := h.S.IssueCredit(userID(r), req.AccountID, req.Amount, req.Months)
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 201, map[string]any{"id": id})
}
func (h *Handler) Schedule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(mux.Vars(r)["creditId"], 10, 64)
	s, err := h.S.Repo.GetSchedule(userID(r), id)
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, s)
}
func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	a, err := h.S.Analytics(userID(r))
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, a)
}
func (h *Handler) Predict(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(mux.Vars(r)["accountId"], 10, 64)
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days == 0 {
		days = 30
	}
	p, err := h.S.Predict(userID(r), id, days)
	if err != nil {
		utils.Error(w, 400, err.Error())
		return
	}
	utils.JSON(w, 200, p)
}
