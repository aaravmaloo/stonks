package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"stanks/internal/game"
)

func (s *adminStore) ListPlayers(ctx context.Context, query string) ([]playerRow, error) {
	path := "/v1/admin/players"
	if q := strings.TrimSpace(query); q != "" {
		path += "?q=" + url.QueryEscape(q)
	}
	var out struct {
		Players []playerRow `json:"players"`
	}
	if err := s.jsonRequest(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Players, nil
}

func (s *adminStore) PlayerByID(ctx context.Context, userID string) (playerRow, error) {
	var out playerRow
	err := s.jsonRequest(ctx, http.MethodGet, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID)), nil, &out)
	return out, err
}

func (s *adminStore) ChangeBalance(ctx context.Context, userID string, deltaMicros int64) (playerRow, error) {
	var out playerRow
	err := s.jsonRequest(ctx, http.MethodPost, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/balance/change", map[string]any{
		"delta_micros": deltaMicros,
	}, &out)
	return out, err
}

func (s *adminStore) SetBalance(ctx context.Context, userID string, amountMicros int64) (playerRow, error) {
	var out playerRow
	err := s.jsonRequest(ctx, http.MethodPost, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/balance/set", map[string]any{
		"amount_micros": amountMicros,
	}, &out)
	return out, err
}

func (s *adminStore) ChangePeak(ctx context.Context, userID string, deltaMicros int64) (playerRow, error) {
	var out playerRow
	err := s.jsonRequest(ctx, http.MethodPost, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/peak/change", map[string]any{
		"delta_micros": deltaMicros,
	}, &out)
	return out, err
}

func (s *adminStore) SetPeak(ctx context.Context, userID string, amountMicros int64) (playerRow, error) {
	var out playerRow
	err := s.jsonRequest(ctx, http.MethodPost, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/peak/set", map[string]any{
		"amount_micros": amountMicros,
	}, &out)
	return out, err
}

func (s *adminStore) SetPlayerProgress(ctx context.Context, userID string, reputationScore, currentStreak, bestStreak, riskAppetiteBps int32) (playerRow, error) {
	var out playerRow
	err := s.jsonRequest(ctx, http.MethodPost, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/progress", map[string]any{
		"reputation_score":      reputationScore,
		"current_profit_streak": currentStreak,
		"best_profit_streak":    bestStreak,
		"risk_appetite_bps":     riskAppetiteBps,
	}, &out)
	return out, err
}

func (s *adminStore) SetActiveBusiness(ctx context.Context, userID string, businessID int64) (playerRow, error) {
	var out playerRow
	err := s.jsonRequest(ctx, http.MethodPost, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/active-business", map[string]any{
		"business_id": businessID,
	}, &out)
	return out, err
}

func (s *adminStore) ListBusinessesByUser(ctx context.Context, userID string) ([]businessRow, error) {
	var out struct {
		Businesses []businessRow `json:"businesses"`
	}
	err := s.jsonRequest(ctx, http.MethodGet, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/businesses", nil, &out)
	return out.Businesses, err
}

func (s *adminStore) SetBusinessName(ctx context.Context, businessID int64, name string) (businessRow, error) {
	var out businessRow
	err := s.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/admin/businesses/%d/name", businessID), map[string]any{
		"name": name,
	}, &out)
	return out, err
}

func (s *adminStore) SetBusinessVisibility(ctx context.Context, businessID int64, visibility string) (businessRow, error) {
	var out businessRow
	err := s.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/admin/businesses/%d/visibility", businessID), map[string]any{
		"visibility": visibility,
	}, &out)
	return out, err
}

func (s *adminStore) SetBusinessListed(ctx context.Context, businessID int64, listed bool) (businessRow, error) {
	var out businessRow
	err := s.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/admin/businesses/%d/listed", businessID), map[string]any{
		"listed": listed,
	}, &out)
	return out, err
}

func (s *adminStore) SetBusinessRevenue(ctx context.Context, businessID int64, amountMicros int64) (businessRow, error) {
	var out businessRow
	err := s.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/admin/businesses/%d/revenue", businessID), map[string]any{
		"amount_micros": amountMicros,
	}, &out)
	return out, err
}

func (s *adminStore) SetBusinessNarrative(ctx context.Context, businessID int64, region, arc, focus string, pressureBps int32) (businessRow, error) {
	var out businessRow
	err := s.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/admin/businesses/%d/narrative", businessID), map[string]any{
		"primary_region":         region,
		"narrative_arc":          arc,
		"narrative_focus":        focus,
		"narrative_pressure_bps": pressureBps,
	}, &out)
	return out, err
}

func (s *adminStore) ListBusinessStakes(ctx context.Context, businessID int64) ([]stakeRow, error) {
	var out struct {
		Stakes []stakeRow `json:"stakes"`
	}
	err := s.jsonRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/admin/businesses/%d/stakes", businessID), nil, &out)
	return out.Stakes, err
}

func (s *adminStore) SetBusinessStake(ctx context.Context, businessID int64, username string, stakeBps int32) ([]stakeRow, error) {
	var out struct {
		Stakes []stakeRow `json:"stakes"`
	}
	err := s.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/admin/businesses/%d/stakes", businessID), map[string]any{
		"username":  username,
		"stake_bps": stakeBps,
	}, &out)
	return out.Stakes, err
}

func (s *adminStore) DeleteBusiness(ctx context.Context, businessID int64) error {
	return s.jsonRequest(ctx, http.MethodDelete, fmt.Sprintf("/v1/admin/businesses/%d", businessID), nil, nil)
}

func (s *adminStore) ListPositionsByUser(ctx context.Context, userID string) ([]positionRow, error) {
	var out struct {
		Positions []positionRow `json:"positions"`
	}
	err := s.jsonRequest(ctx, http.MethodGet, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/positions", nil, &out)
	return out.Positions, err
}

func (s *adminStore) SetPosition(ctx context.Context, userID, symbol string, shares float64, avgPriceMicros int64) (positionRow, error) {
	units, err := game.SharesToUnits(shares)
	if err != nil {
		return positionRow{}, err
	}
	var out positionRow
	err = s.jsonRequest(ctx, http.MethodPost, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/positions/"+url.PathEscape(strings.ToUpper(symbol)), map[string]any{
		"quantity_units":   units,
		"avg_price_micros": avgPriceMicros,
	}, &out)
	return out, err
}

func (s *adminStore) DeletePosition(ctx context.Context, userID, symbol string) error {
	return s.jsonRequest(ctx, http.MethodDelete, "/v1/admin/players/"+url.PathEscape(strings.TrimSpace(userID))+"/positions/"+url.PathEscape(strings.ToUpper(symbol)), nil, nil)
}

func (s *adminStore) ListStocks(ctx context.Context) ([]stockRow, error) {
	var out struct {
		Stocks []stockRow `json:"stocks"`
	}
	err := s.jsonRequest(ctx, http.MethodGet, "/v1/admin/stocks", nil, &out)
	return out.Stocks, err
}

func (s *adminStore) SetStockPrice(ctx context.Context, symbol string, priceMicros int64) (stockRow, error) {
	var out stockRow
	err := s.jsonRequest(ctx, http.MethodPost, "/v1/admin/stocks/"+url.PathEscape(strings.ToUpper(symbol))+"/price", map[string]any{
		"price_micros": priceMicros,
	}, &out)
	return out, err
}

func (s *adminStore) WorldState(ctx context.Context) (worldRow, error) {
	var out worldRow
	err := s.jsonRequest(ctx, http.MethodGet, "/v1/admin/world", nil, &out)
	return out, err
}

func (s *adminStore) SetWorldState(ctx context.Context, in worldRow) (worldRow, error) {
	var out worldRow
	err := s.jsonRequest(ctx, http.MethodPost, "/v1/admin/world", in, &out)
	return out, err
}

func (s *adminStore) jsonRequest(ctx context.Context, method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(s.baseURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.SetBasicAuth(s.username, s.password)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("api status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
