package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"stanks/internal/auth"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Signup(ctx context.Context, email, password, username string) (auth.Session, error) {
	var out auth.Session
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/auth/signup", "", map[string]any{
		"email":    email,
		"password": password,
		"username": username,
	}, &out, "")
	return out, err
}

func (c *Client) Login(ctx context.Context, email, password string) (auth.Session, error) {
	var out auth.Session
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	}, &out, "")
	return out, err
}

func (c *Client) Dashboard(ctx context.Context, accessToken string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, "/v1/dashboard", accessToken, nil, &out, "")
	return out, err
}

func (c *Client) ListStocks(ctx context.Context, accessToken string, all bool) (map[string]any, error) {
	path := "/v1/stocks"
	if all {
		path = "/v1/stocks?all=1"
	}
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, path, accessToken, nil, &out, "")
	return out, err
}

func (c *Client) StockDetail(ctx context.Context, accessToken, symbol string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, "/v1/stocks/"+url.PathEscape(symbol), accessToken, nil, &out, "")
	return out, err
}

func (c *Client) PlaceOrder(ctx context.Context, accessToken, symbol, side, idem string, qtyUnits int64) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/orders", accessToken, map[string]any{
		"symbol":         symbol,
		"side":           side,
		"quantity_units": qtyUnits,
	}, &out, idem)
	return out, err
}

func (c *Client) CreateBusiness(ctx context.Context, accessToken, name, visibility, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/businesses", accessToken, map[string]any{
		"name":       name,
		"visibility": visibility,
	}, &out, idem)
	return out, err
}

func (c *Client) BusinessState(ctx context.Context, accessToken string, businessID int64) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/businesses/%d", businessID), accessToken, nil, &out, "")
	return out, err
}

func (c *Client) SetBusinessVisibility(ctx context.Context, accessToken string, businessID int64, visibility, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/visibility", businessID), accessToken, map[string]any{
		"visibility": visibility,
	}, &out, idem)
	return out, err
}

func (c *Client) ListEmployeeCandidates(ctx context.Context, accessToken string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, "/v1/businesses/employees/candidates", accessToken, nil, &out, "")
	return out, err
}

func (c *Client) ListBusinessEmployees(ctx context.Context, accessToken string, businessID int64) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/businesses/%d/employees", businessID), accessToken, nil, &out, "")
	return out, err
}

func (c *Client) HireEmployee(ctx context.Context, accessToken string, businessID, candidateID int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/employees/hire", businessID), accessToken, map[string]any{
		"candidate_id": candidateID,
	}, &out, idem)
	return out, err
}

func (c *Client) TrainProfessional(ctx context.Context, accessToken string, businessID, employeeID int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/employees/%d/train", businessID, employeeID), accessToken, map[string]any{}, &out, idem)
	return out, err
}

func (c *Client) ListBusinessMachinery(ctx context.Context, accessToken string, businessID int64) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/businesses/%d/machinery", businessID), accessToken, nil, &out, "")
	return out, err
}

func (c *Client) ListBusinessLoans(ctx context.Context, accessToken string, businessID int64) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/businesses/%d/loans", businessID), accessToken, nil, &out, "")
	return out, err
}

func (c *Client) BuyBusinessMachinery(ctx context.Context, accessToken string, businessID int64, machineType, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/machinery/buy", businessID), accessToken, map[string]any{
		"machine_type": machineType,
	}, &out, idem)
	return out, err
}

func (c *Client) TakeBusinessLoan(ctx context.Context, accessToken string, businessID int64, amountMicros int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/loans/take", businessID), accessToken, map[string]any{
		"amount_micros": amountMicros,
	}, &out, idem)
	return out, err
}

func (c *Client) RepayBusinessLoan(ctx context.Context, accessToken string, businessID int64, amountMicros int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/loans/repay", businessID), accessToken, map[string]any{
		"amount_micros": amountMicros,
	}, &out, idem)
	return out, err
}

func (c *Client) SetBusinessStrategy(ctx context.Context, accessToken string, businessID int64, strategy, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/strategy", businessID), accessToken, map[string]any{
		"strategy": strategy,
	}, &out, idem)
	return out, err
}

func (c *Client) BuyBusinessUpgrade(ctx context.Context, accessToken string, businessID int64, upgrade, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/upgrades/buy", businessID), accessToken, map[string]any{
		"upgrade": upgrade,
	}, &out, idem)
	return out, err
}

func (c *Client) BusinessReserveDeposit(ctx context.Context, accessToken string, businessID int64, amountMicros int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/reserve/deposit", businessID), accessToken, map[string]any{
		"amount_micros": amountMicros,
	}, &out, idem)
	return out, err
}

func (c *Client) BusinessReserveWithdraw(ctx context.Context, accessToken string, businessID int64, amountMicros int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/reserve/withdraw", businessID), accessToken, map[string]any{
		"amount_micros": amountMicros,
	}, &out, idem)
	return out, err
}

func (c *Client) SellBusinessToBank(ctx context.Context, accessToken string, businessID int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/sell", businessID), accessToken, map[string]any{}, &out, idem)
	return out, err
}

func (c *Client) BusinessIPO(ctx context.Context, accessToken string, businessID int64, symbol string, priceMicros int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/businesses/%d/ipo", businessID), accessToken, map[string]any{
		"symbol":       symbol,
		"price_micros": priceMicros,
	}, &out, idem)
	return out, err
}

func (c *Client) CreateStock(ctx context.Context, accessToken, symbol, display string, businessID int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/stocks/custom", accessToken, map[string]any{
		"symbol":       symbol,
		"display_name": display,
		"business_id":  businessID,
	}, &out, idem)
	return out, err
}

func (c *Client) IPOStock(ctx context.Context, accessToken, symbol string, priceMicros int64, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/stocks/"+url.PathEscape(symbol)+"/ipo", accessToken, map[string]any{
		"price_micros": priceMicros,
	}, &out, idem)
	return out, err
}

func (c *Client) ListFunds(ctx context.Context, accessToken string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, "/v1/funds", accessToken, nil, &out, "")
	return out, err
}

func (c *Client) BuyFund(ctx context.Context, accessToken, fundCode, idem string, units int64) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/funds/"+url.PathEscape(fundCode)+"/buy", accessToken, map[string]any{
		"units": units,
	}, &out, idem)
	return out, err
}

func (c *Client) SellFund(ctx context.Context, accessToken, fundCode, idem string, units int64) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/funds/"+url.PathEscape(fundCode)+"/sell", accessToken, map[string]any{
		"units": units,
	}, &out, idem)
	return out, err
}

func (c *Client) LeaderboardGlobal(ctx context.Context, accessToken string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, "/v1/leaderboard/global", accessToken, nil, &out, "")
	return out, err
}

func (c *Client) LeaderboardFriends(ctx context.Context, accessToken string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodGet, "/v1/leaderboard/friends", accessToken, nil, &out, "")
	return out, err
}

func (c *Client) AddFriend(ctx context.Context, accessToken, inviteCode, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/friends", accessToken, map[string]any{
		"invite_code": inviteCode,
	}, &out, idem)
	return out, err
}

func (c *Client) RemoveFriend(ctx context.Context, accessToken, inviteCode string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodDelete, "/v1/friends/"+url.PathEscape(inviteCode), accessToken, nil, &out, "")
	return out, err
}

func (c *Client) SyncReplay(ctx context.Context, accessToken string, commands []map[string]any) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, http.MethodPost, "/v1/sync/replay", accessToken, map[string]any{
		"commands": commands,
	}, &out, "")
	return out, err
}

func (c *Client) Do(ctx context.Context, method, path, accessToken string, body map[string]any, idem string) (map[string]any, error) {
	var out map[string]any
	err := c.jsonRequest(ctx, method, path, accessToken, body, &out, idem)
	return out, err
}

func (c *Client) jsonRequest(ctx context.Context, method, path, accessToken string, in any, out any, idem string) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	resp, err := c.HTTP.Do(req)
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
