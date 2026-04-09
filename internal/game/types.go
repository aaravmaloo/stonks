package game

import "time"

type Dashboard struct {
	SeasonID           int64          `json:"season_id"`
	ActiveBusinessID   *int64         `json:"active_business_id,omitempty"`
	BalanceMicros      int64          `json:"balance_micros"`
	NetWorthMicros     int64          `json:"net_worth_micros"`
	PeakNetWorthMicros int64          `json:"peak_net_worth_micros"`
	Progression        PlayerProgress `json:"progression"`
	World              WorldView      `json:"world"`
	Positions          []PositionView `json:"positions"`
	Businesses         []BusinessView `json:"businesses"`
	Stakes             []StakeView    `json:"stakes"`
}

type WalletSummary struct {
	SeasonID           int64  `json:"season_id"`
	ActiveBusinessID   *int64 `json:"active_business_id,omitempty"`
	BalanceMicros      int64  `json:"balance_micros"`
	PeakNetWorthMicros int64  `json:"peak_net_worth_micros"`
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
	StockSymbol           string `json:"stock_symbol,omitempty"`
	PrimaryRegion         string `json:"primary_region"`
	NarrativeArc          string `json:"narrative_arc"`
	NarrativeFocus        string `json:"narrative_focus"`
	NarrativePressureBps  int32  `json:"narrative_pressure_bps"`
	EmployeeLimit         int64  `json:"employee_limit"`
	EmployeeCount         int64  `json:"employee_count"`
	RevenuePerTickMicros  int64  `json:"revenue_per_tick_micros"`
	GrossRevenueMicros    int64  `json:"gross_revenue_micros"`
	OperatingCostsMicros  int64  `json:"operating_costs_micros"`
	EmployeeSalaryMicros  int64  `json:"employee_salary_micros"`
	MaintenanceMicros     int64  `json:"maintenance_micros"`
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
	OwnedStakeBps         int32  `json:"owned_stake_bps"`
}

type StakeView struct {
	BusinessID           int64  `json:"business_id"`
	BusinessName         string `json:"business_name"`
	ControllerUsername   string `json:"controller_username"`
	PrimaryRegion        string `json:"primary_region"`
	NarrativeArc         string `json:"narrative_arc"`
	StakeBps             int32  `json:"stake_bps"`
	RevenueShareMicros   int64  `json:"revenue_share_micros"`
	EstimatedValueMicros int64  `json:"estimated_value_micros"`
	CostBasisMicros      int64  `json:"cost_basis_micros"`
	UnrealizedPLMicros   int64  `json:"unrealized_pl_micros"`
	LastEvent            string `json:"last_event"`
}

type PlayerProgress struct {
	ReputationScore         int32  `json:"reputation_score"`
	ReputationTitle         string `json:"reputation_title"`
	CurrentProfitStreak     int32  `json:"current_profit_streak"`
	BestProfitStreak        int32  `json:"best_profit_streak"`
	RiskAppetiteBps         int32  `json:"risk_appetite_bps"`
	LastNetWorthDeltaMicros int64  `json:"last_net_worth_delta_micros"`
	LastRiskPayoutMicros    int64  `json:"last_risk_payout_micros"`
	LastStreakRewardMicros  int64  `json:"last_streak_reward_micros"`
	NextStreakTarget        int32  `json:"next_streak_target"`
}

type WorldView struct {
	Regime                 string           `json:"regime"`
	PoliticalClimate       string           `json:"political_climate"`
	PolicyFocus            string           `json:"policy_focus"`
	CatalystName           string           `json:"catalyst_name"`
	CatalystSummary        string           `json:"catalyst_summary"`
	CatalystTicksRemaining int32            `json:"catalyst_ticks_remaining"`
	Headline               string           `json:"headline"`
	RiskRewardBiasBps      int32            `json:"risk_reward_bias_bps"`
	Regions                []RegionView     `json:"regions"`
	RecentEvents           []WorldEventView `json:"recent_events"`
}

type RegionView struct {
	Name     string `json:"name"`
	TrendBps int32  `json:"trend_bps"`
}

type WorldEventView struct {
	Category      string    `json:"category"`
	Headline      string    `json:"headline"`
	ImpactSummary string    `json:"impact_summary"`
	CreatedAt     time.Time `json:"created_at"`
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

type BulkHireEmployeesInput struct {
	UserID         string
	SeasonID       int64
	BusinessID     int64
	Count          int
	Strategy       string
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

type TransferBusinessStakeInput struct {
	UserID            string
	SeasonID          int64
	BusinessID        int64
	RecipientUsername string
	StakeBps          int32
	IdempotencyKey    string
}

type LeaderboardRow struct {
	Rank           int64  `json:"rank"`
	Username       string `json:"username"`
	InviteCode     string `json:"invite_code"`
	NetWorthMicros int64  `json:"net_worth_micros"`
}
