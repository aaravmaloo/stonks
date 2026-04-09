package api

import (
	"net/http"
	"strconv"
	"strings"

	"stanks/internal/admin"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleAdminPlayers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.admin.ListPlayers(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"players": rows})
}

func (s *Server) handleAdminPlayer(w http.ResponseWriter, r *http.Request) {
	row, err := s.admin.PlayerByID(r.Context(), chi.URLParam(r, "userID"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminChangeBalance(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DeltaMicros int64 `json:"delta_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.ChangeBalance(r.Context(), chi.URLParam(r, "userID"), in.DeltaMicros)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminSetBalance(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AmountMicros int64 `json:"amount_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetBalance(r.Context(), chi.URLParam(r, "userID"), in.AmountMicros)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminChangePeak(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DeltaMicros int64 `json:"delta_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.ChangePeak(r.Context(), chi.URLParam(r, "userID"), in.DeltaMicros)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminSetPeak(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AmountMicros int64 `json:"amount_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetPeak(r.Context(), chi.URLParam(r, "userID"), in.AmountMicros)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminSetPlayerProgress(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ReputationScore     int32 `json:"reputation_score"`
		CurrentProfitStreak int32 `json:"current_profit_streak"`
		BestProfitStreak    int32 `json:"best_profit_streak"`
		RiskAppetiteBps     int32 `json:"risk_appetite_bps"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetPlayerProgress(r.Context(), chi.URLParam(r, "userID"), in.ReputationScore, in.CurrentProfitStreak, in.BestProfitStreak, in.RiskAppetiteBps)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminSetActiveBusiness(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BusinessID int64 `json:"business_id"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetActiveBusiness(r.Context(), chi.URLParam(r, "userID"), in.BusinessID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminBusinesses(w http.ResponseWriter, r *http.Request) {
	rows, err := s.admin.ListBusinessesByUser(r.Context(), chi.URLParam(r, "userID"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"businesses": rows})
}

func (s *Server) handleAdminPositions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.admin.ListPositionsByUser(r.Context(), chi.URLParam(r, "userID"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"positions": rows})
}

func (s *Server) handleAdminSetPosition(w http.ResponseWriter, r *http.Request) {
	var in struct {
		QuantityUnits  int64 `json:"quantity_units"`
		AvgPriceMicros int64 `json:"avg_price_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetPosition(r.Context(), chi.URLParam(r, "userID"), strings.ToUpper(chi.URLParam(r, "symbol")), in.QuantityUnits, in.AvgPriceMicros)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminDeletePosition(w http.ResponseWriter, r *http.Request) {
	if err := s.admin.DeletePosition(r.Context(), chi.URLParam(r, "userID"), strings.ToUpper(chi.URLParam(r, "symbol"))); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminSetBusinessName(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseBusinessID(w, r)
	if !ok {
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetBusinessName(r.Context(), businessID, in.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminSetBusinessVisibility(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseBusinessID(w, r)
	if !ok {
		return
	}
	var in struct {
		Visibility string `json:"visibility"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetBusinessVisibility(r.Context(), businessID, strings.ToLower(strings.TrimSpace(in.Visibility)))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminSetBusinessListed(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseBusinessID(w, r)
	if !ok {
		return
	}
	var in struct {
		Listed bool `json:"listed"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetBusinessListed(r.Context(), businessID, in.Listed)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminSetBusinessRevenue(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseBusinessID(w, r)
	if !ok {
		return
	}
	var in struct {
		AmountMicros int64 `json:"amount_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetBusinessRevenue(r.Context(), businessID, in.AmountMicros)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminDeleteBusiness(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseBusinessID(w, r)
	if !ok {
		return
	}
	if err := s.admin.DeleteBusiness(r.Context(), businessID); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminSetBusinessNarrative(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseBusinessID(w, r)
	if !ok {
		return
	}
	var in struct {
		PrimaryRegion        string `json:"primary_region"`
		NarrativeArc         string `json:"narrative_arc"`
		NarrativeFocus       string `json:"narrative_focus"`
		NarrativePressureBps int32  `json:"narrative_pressure_bps"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetBusinessNarrative(r.Context(), businessID, in.PrimaryRegion, in.NarrativeArc, in.NarrativeFocus, in.NarrativePressureBps)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminBusinessStakes(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseBusinessID(w, r)
	if !ok {
		return
	}
	rows, err := s.admin.ListBusinessStakes(r.Context(), businessID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stakes": rows})
}

func (s *Server) handleAdminSetBusinessStake(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseBusinessID(w, r)
	if !ok {
		return
	}
	var in struct {
		Username string `json:"username"`
		StakeBps int32  `json:"stake_bps"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rows, err := s.admin.SetBusinessStake(r.Context(), businessID, in.Username, in.StakeBps)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stakes": rows})
}

func (s *Server) handleAdminStocks(w http.ResponseWriter, r *http.Request) {
	rows, err := s.admin.ListStocks(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stocks": rows})
}

func (s *Server) handleAdminSetStockPrice(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PriceMicros int64 `json:"price_micros"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetStockPrice(r.Context(), strings.ToUpper(chi.URLParam(r, "symbol")), in.PriceMicros)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminWorld(w http.ResponseWriter, r *http.Request) {
	row, err := s.admin.WorldState(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminSetWorld(w http.ResponseWriter, r *http.Request) {
	var in admin.WorldState
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	row, err := s.admin.SetWorldState(r.Context(), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func parseBusinessID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	businessID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid business id")
		return 0, false
	}
	return businessID, true
}
