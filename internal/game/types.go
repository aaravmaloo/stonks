package game

import "time"

type Dashboard struct {
	SeasonID           int64          `json:"season_id"`
	BalanceMicros      int64          `json:"balance_micros"`
	NetWorthMicros     int64          `json:"net_worth_micros"`
	PeakNetWorthMicros int64          `json:"peak_net_worth_micros"`
	Positions          []PositionView `json:"positions"`
	Businesses         []BusinessView `json:"businesses"`
}

type PositionView struct {
	Symbol             string `json:"symbol"`
	DisplayName        string `json:"display_name"`
	QuantityUnits      int64  `json:"quantity_units"`
	AvgPriceMicros     int64  `json:"avg_price_micros"`
	CurrentPriceMicros int64  `json:"current_price_micros"`
	UnrealizedMicros   int64  `json:"unrealized_micros"`
}

type BusinessView struct {
	ID                    int64  `json:"id"`
	Name                  string `json:"name"`
	Visibility            string `json:"visibility"`
	IsListed              bool   `json:"is_listed"`
	EmployeeCount         int64  `json:"employee_count"`
	RevenuePerTickMicros  int64  `json:"revenue_per_tick_micros"`
	MachineryCount        int64  `json:"machinery_count"`
	MachineryOutputMicros int64  `json:"machinery_output_micros"`
	MachineryUpkeepMicros int64  `json:"machinery_upkeep_micros"`
	LoanOutstandingMicros int64  `json:"loan_outstanding_micros"`
	Strategy              string `json:"strategy"`
	MarketingLevel        int32  `json:"marketing_level"`
	RDLevel               int32  `json:"rd_level"`
	AutomationLevel       int32  `json:"automation_level"`
	ComplianceLevel       int32  `json:"compliance_level"`
	BrandBps              int32  `json:"brand_bps"`
	OperationalHealthBps  int32  `json:"operational_health_bps"`
	CashReserveMicros     int64  `json:"cash_reserve_micros"`
	LastEvent             string `json:"last_event"`
}

type StockView struct {
	Symbol             string `json:"symbol"`
	DisplayName        string `json:"display_name"`
	CurrentPriceMicros int64  `json:"current_price_micros"`
	ListedPublic       bool   `json:"listed_public"`
}

type StockDetail struct {
	StockView
	Series []PricePoint `json:"series"`
}

type PricePoint struct {
	TickAt      time.Time `json:"tick_at"`
	PriceMicros int64     `json:"price_micros"`
}

type OrderInput struct {
	UserID         string
	SeasonID       int64
	Symbol         string
	Side           string
	QuantityUnits  int64
	IdempotencyKey string
}

type OrderResult struct {
	OrderID        int64 `json:"order_id"`
	PriceMicros    int64 `json:"price_micros"`
	NotionalMicros int64 `json:"notional_micros"`
	FeeMicros      int64 `json:"fee_micros"`
	BalanceMicros  int64 `json:"balance_micros"`
}

type CreateBusinessInput struct {
	UserID         string
	SeasonID       int64
	Name           string
	Visibility     string
	IdempotencyKey string
}

type CreateStockInput struct {
	UserID         string
	SeasonID       int64
	Symbol         string
	DisplayName    string
	BusinessID     int64
	IdempotencyKey string
}

type IPOInput struct {
	UserID         string
	SeasonID       int64
	Symbol         string
	PriceMicros    int64
	IdempotencyKey string
}

type HireEmployeeInput struct {
	UserID         string
	SeasonID       int64
	BusinessID     int64
	CandidateID    int64
	IdempotencyKey string
}

type BuyMachineryInput struct {
	UserID         string
	SeasonID       int64
	BusinessID     int64
	MachineType    string
	IdempotencyKey string
}

type TrainProfessionalInput struct {
	UserID         string
	SeasonID       int64
	BusinessID     int64
	EmployeeID     int64
	IdempotencyKey string
}

type BusinessLoanInput struct {
	UserID         string
	SeasonID       int64
	BusinessID     int64
	AmountMicros   int64
	IdempotencyKey string
}

type BusinessStrategyInput struct {
	UserID         string
	SeasonID       int64
	BusinessID     int64
	Strategy       string
	IdempotencyKey string
}

type BusinessUpgradeInput struct {
	UserID         string
	SeasonID       int64
	BusinessID     int64
	Upgrade        string
	IdempotencyKey string
}

type BusinessReserveInput struct {
	UserID         string
	SeasonID       int64
	BusinessID     int64
	AmountMicros   int64
	IdempotencyKey string
}

type FundOrderInput struct {
	UserID         string
	SeasonID       int64
	FundCode       string
	Side           string
	Units          int64
	IdempotencyKey string
}

type LeaderboardRow struct {
	Rank           int64  `json:"rank"`
	Username       string `json:"username"`
	InviteCode     string `json:"invite_code"`
	NetWorthMicros int64  `json:"net_worth_micros"`
}
