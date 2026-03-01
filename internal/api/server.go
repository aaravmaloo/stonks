package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stanks/internal/auth"
	"stanks/internal/config"
	"stanks/internal/game"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "user"

type UserContext struct {
	UserID string
	Email  string
	Token  string
}

type Server struct {
	cfg  config.APIConfig
	log  *slog.Logger
	auth *auth.SupabaseClient
	game *game.Service
	mux  *chi.Mux
}

func New(cfg config.APIConfig, logger *slog.Logger, authClient *auth.SupabaseClient, gameSvc *game.Service) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		cfg:  cfg,
		log:  logger,
		auth: authClient,
		game: gameSvc,
		mux:  chi.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	r := s.mux
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Post("/auth/signup", s.handleSignup)
		r.Post("/auth/login", s.handleLogin)

		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Get("/dashboard", s.handleDashboard)
			r.Get("/stocks", s.handleStocksList)
			r.Get("/stocks/{symbol}", s.handleStockDetail)
			r.Post("/orders", s.handleOrder)

			r.Post("/businesses", s.handleCreateBusiness)
			r.Get("/businesses/{id}", s.handleBusinessState)
			r.Get("/businesses/{id}/employees", s.handleBusinessEmployees)
			r.Get("/businesses/employees/candidates", s.handleEmployeeCandidates)
			r.Post("/businesses/{id}/employees/hire", s.handleHireEmployee)
			r.Post("/businesses/{id}/employees/{employee_id}/train", s.handleTrainProfessional)
			r.Get("/businesses/{id}/machinery", s.handleBusinessMachinery)
			r.Get("/businesses/{id}/loans", s.handleBusinessLoans)
			r.Post("/businesses/{id}/machinery/buy", s.handleBuyMachinery)
			r.Post("/businesses/{id}/loans/take", s.handleTakeBusinessLoan)
			r.Post("/businesses/{id}/loans/repay", s.handleRepayBusinessLoan)
			r.Post("/businesses/{id}/strategy", s.handleSetBusinessStrategy)
			r.Post("/businesses/{id}/upgrades/buy", s.handleBuyBusinessUpgrade)
			r.Post("/businesses/{id}/reserve/deposit", s.handleBusinessReserveDeposit)
			r.Post("/businesses/{id}/reserve/withdraw", s.handleBusinessReserveWithdraw)
			r.Post("/businesses/{id}/visibility", s.handleBusinessVisibility)
			r.Post("/businesses/{id}/ipo", s.handleBusinessIPO)
			r.Post("/businesses/{id}/sell", s.handleSellBusiness)

			r.Post("/stocks/custom", s.handleCreateCustomStock)
			r.Post("/stocks/{symbol}/ipo", s.handleIPOStock)
			r.Get("/funds", s.handleFundsList)
			r.Post("/funds/{code}/buy", s.handleFundBuy)
			r.Post("/funds/{code}/sell", s.handleFundSell)

			r.Get("/leaderboard/global", s.handleLeaderboardGlobal)
			r.Get("/leaderboard/friends", s.handleLeaderboardFriends)
			r.Post("/friends", s.handleFriendAdd)
			r.Delete("/friends/{invite_code}", s.handleFriendDelete)

			r.Post("/sync/replay", s.handleSyncReplay)
		})
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		user, err := s.auth.VerifyAccessToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err))
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, UserContext{
			UserID: user.ID,
			Email:  user.Email,
			Token:  token,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userFromContext(ctx context.Context) (UserContext, error) {
	v := ctx.Value(userContextKey)
	user, ok := v.(UserContext)
	if !ok || user.UserID == "" {
		return UserContext{}, errors.New("missing auth context")
	}
	return user, nil
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Username string `json:"username"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	session, err := s.auth.SignUp(r.Context(), strings.TrimSpace(in.Email), strings.TrimSpace(in.Password))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if session.User.ID != "" {
		if err := s.game.EnsurePlayer(r.Context(), session.User.ID, session.User.Email, in.Username); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	session, err := s.auth.Login(r.Context(), strings.TrimSpace(in.Email), strings.TrimSpace(in.Password))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err := s.game.EnsurePlayer(r.Context(), session.User.ID, session.User.Email, ""); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out, err := s.game.Dashboard(r.Context(), user.UserID, seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleStocksList(w http.ResponseWriter, r *http.Request) {
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	includeUnlisted := r.URL.Query().Get("all") == "1"
	out, err := s.game.ListStocks(r.Context(), seasonID, includeUnlisted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stocks": out})
}

func (s *Server) handleStockDetail(w http.ResponseWriter, r *http.Request) {
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	symbol := chi.URLParam(r, "symbol")
	out, err := s.game.StockDetail(r.Context(), seasonID, symbol)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleOrder(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var in struct {
		Symbol        string `json:"symbol"`
		Side          string `json:"side"`
		QuantityUnits int64  `json:"quantity_units"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.game.PlaceOrder(r.Context(), game.OrderInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		Symbol:         in.Symbol,
		Side:           in.Side,
		QuantityUnits:  in.QuantityUnits,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCreateBusiness(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var in struct {
		Name       string `json:"name"`
		Visibility string `json:"visibility"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := s.game.CreateBusiness(r.Context(), game.CreateBusinessInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		Name:           in.Name,
		Visibility:     in.Visibility,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleBusinessState(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	out, err := s.game.BusinessState(r.Context(), user.UserID, seasonID, businessID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleBusinessEmployees(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	employees, err := s.game.ListBusinessEmployees(r.Context(), user.UserID, seasonID, businessID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"employees": employees})
}

func (s *Server) handleEmployeeCandidates(w http.ResponseWriter, r *http.Request) {
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	candidates, err := s.game.ListEmployeeCandidates(r.Context(), seasonID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": candidates})
}

func (s *Server) handleHireEmployee(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		CandidateID int64 `json:"candidate_id"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	err = s.game.HireEmployee(r.Context(), game.HireEmployeeInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		CandidateID:    in.CandidateID,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleTrainProfessional(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	employeeID, err := strconv.ParseInt(chi.URLParam(r, "employee_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid employee id")
		return
	}
	out, err := s.game.TrainProfessional(r.Context(), game.TrainProfessionalInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		EmployeeID:     employeeID,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleBusinessMachinery(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	out, err := s.game.ListBusinessMachinery(r.Context(), user.UserID, seasonID, businessID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"machinery": out})
}

func (s *Server) handleBusinessLoans(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	out, err := s.game.ListBusinessLoans(r.Context(), user.UserID, seasonID, businessID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"loans": out})
}

func (s *Server) handleBuyMachinery(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		MachineType string `json:"machine_type"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.game.BuyBusinessMachinery(r.Context(), game.BuyMachineryInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		MachineType:    in.MachineType,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleTakeBusinessLoan(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		AmountMicros int64 `json:"amount_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.game.TakeBusinessLoan(r.Context(), game.BusinessLoanInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		AmountMicros:   in.AmountMicros,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRepayBusinessLoan(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		AmountMicros int64 `json:"amount_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.game.RepayBusinessLoan(r.Context(), game.BusinessLoanInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		AmountMicros:   in.AmountMicros,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSetBusinessStrategy(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		Strategy string `json:"strategy"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.game.SetBusinessStrategy(r.Context(), game.BusinessStrategyInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		Strategy:       in.Strategy,
		IdempotencyKey: idempotencyKey(r),
	}); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBuyBusinessUpgrade(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		Upgrade string `json:"upgrade"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.game.BuyBusinessUpgrade(r.Context(), game.BusinessUpgradeInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		Upgrade:        in.Upgrade,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleBusinessReserveDeposit(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		AmountMicros int64 `json:"amount_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.game.BusinessReserveDeposit(r.Context(), game.BusinessReserveInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		AmountMicros:   in.AmountMicros,
		IdempotencyKey: idempotencyKey(r),
	}); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBusinessReserveWithdraw(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		AmountMicros int64 `json:"amount_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.game.BusinessReserveWithdraw(r.Context(), game.BusinessReserveInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		BusinessID:     businessID,
		AmountMicros:   in.AmountMicros,
		IdempotencyKey: idempotencyKey(r),
	}); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBusinessVisibility(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		Visibility string `json:"visibility"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.game.SetBusinessVisibility(r.Context(), user.UserID, seasonID, businessID, in.Visibility); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBusinessIPO(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	var in struct {
		Symbol      string `json:"symbol"`
		PriceMicros int64  `json:"price_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.game.BusinessIPO(r.Context(), user.UserID, seasonID, businessID, in.Symbol, in.PriceMicros, idempotencyKey(r)); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSellBusiness(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return
	}
	out, err := s.game.SellBusinessToBank(r.Context(), user.UserID, seasonID, businessID, idempotencyKey(r))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateCustomStock(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var in struct {
		Symbol      string `json:"symbol"`
		DisplayName string `json:"display_name"`
		BusinessID  int64  `json:"business_id"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.game.CreateCustomStock(r.Context(), game.CreateStockInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		Symbol:         in.Symbol,
		DisplayName:    in.DisplayName,
		BusinessID:     in.BusinessID,
		IdempotencyKey: idempotencyKey(r),
	}); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

func (s *Server) handleIPOStock(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var in struct {
		PriceMicros int64 `json:"price_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.game.IPOStock(r.Context(), game.IPOInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		Symbol:         chi.URLParam(r, "symbol"),
		PriceMicros:    in.PriceMicros,
		IdempotencyKey: idempotencyKey(r),
	}); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleFundsList(w http.ResponseWriter, r *http.Request) {
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out, err := s.game.ListFunds(r.Context(), seasonID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"funds": out})
}

func (s *Server) handleFundBuy(w http.ResponseWriter, r *http.Request) {
	s.handleFundTrade("buy", w, r)
}

func (s *Server) handleFundSell(w http.ResponseWriter, r *http.Request) {
	s.handleFundTrade("sell", w, r)
}

func (s *Server) handleFundTrade(side string, w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var in struct {
		Units int64 `json:"units"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.game.TradeFund(r.Context(), game.FundOrderInput{
		UserID:         user.UserID,
		SeasonID:       seasonID,
		FundCode:       chi.URLParam(r, "code"),
		Side:           side,
		Units:          in.Units,
		IdempotencyKey: idempotencyKey(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleLeaderboardGlobal(w http.ResponseWriter, r *http.Request) {
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out, err := s.game.GlobalLeaderboard(r.Context(), seasonID, 100)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": out})
}

func (s *Server) handleLeaderboardFriends(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out, err := s.game.FriendsLeaderboard(r.Context(), seasonID, user.UserID, 100)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": out})
}

func (s *Server) handleFriendAdd(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var in struct {
		InviteCode string `json:"invite_code"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.game.AddFriend(r.Context(), user.UserID, in.InviteCode); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleFriendDelete(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err := s.game.RemoveFriend(r.Context(), user.UserID, chi.URLParam(r, "invite_code")); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSyncReplay(w http.ResponseWriter, r *http.Request) {
	user, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	seasonID, err := s.game.ActiveSeasonID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var in struct {
		Commands []map[string]any `json:"commands"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.game.ReplaySync(r.Context(), user.UserID, seasonID, in.Commands)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": out})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, game.ErrDuplicateIdempotency):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, game.ErrInsufficientFunds), errors.Is(err, game.ErrInsufficientShares):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, game.ErrBusinessLocked), errors.Is(err, game.ErrUnauthorized):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, game.ErrInvalidSymbol):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, game.ErrStockNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, game.ErrTxConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func decodeJSON(r *http.Request, out any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": strings.TrimSpace(message)})
}

func idempotencyKey(r *http.Request) string {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key != "" {
		return key
	}
	return uuid.NewString()
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
