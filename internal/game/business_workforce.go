package game

import (
	"fmt"
	"math"
)

type workforceProfile struct {
	EmployeeCount   int64
	OpsCount        int64
	EngineerCount   int64
	ProductCount    int64
	SalesCount      int64
	GrowthCount     int64
	FinanceCount    int64
	LegalCount      int64
	DesignCount     int64
	MarketingLevel  int32
	RDLevel         int32
	AutomationLevel int32
	ComplianceLevel int32
}

type workforceImpact struct {
	RevenueMultiplier   float64
	MachineUpkeepFactor float64
	RiskMultiplier      float64
	ReserveYieldFactor  float64
	LaunchChanceBonus   float64
	DemandChanceBonus   float64
	ViralChanceBonus    float64
	CrisisChanceBonus   float64
}

func bulkHireOrder(strategy string) (string, error) {
	switch strategy {
	case "", "best_value":
		return "(ec.revenue_per_tick_micros::numeric / NULLIF(ec.hire_cost_micros, 0)) DESC, ec.risk_bps ASC, ec.id ASC", nil
	case "high_output":
		return "ec.revenue_per_tick_micros DESC, ec.risk_bps ASC, ec.id ASC", nil
	case "low_risk":
		return "ec.risk_bps ASC, ec.revenue_per_tick_micros DESC, ec.id ASC", nil
	default:
		return "", fmt.Errorf("strategy must be one of: best_value, high_output, low_risk")
	}
}

func analyzeWorkforce(p workforceProfile) workforceImpact {
	impact := workforceImpact{
		RevenueMultiplier:   1,
		MachineUpkeepFactor: 1,
		RiskMultiplier:      1,
		ReserveYieldFactor:  1,
	}

	executionPairs := min64(p.OpsCount, p.EngineerCount)
	productPairs := min64(p.EngineerCount, p.ProductCount)
	goToMarketPairs := min64(p.SalesCount, p.GrowthCount)

	impact.RevenueMultiplier *= 1 + math.Min(0.18, float64(executionPairs)*0.025+float64(p.ProductCount)*0.008)
	impact.RevenueMultiplier *= 1 + math.Min(0.16, float64(productPairs)*0.02+float64(p.RDLevel)*0.004)
	impact.RevenueMultiplier *= 1 + math.Min(0.20, float64(goToMarketPairs)*0.03+float64(p.DesignCount)*0.006+float64(p.MarketingLevel)*0.005)

	if p.EmployeeCount >= 6 && p.OpsCount == 0 {
		impact.RevenueMultiplier *= 0.91
		impact.CrisisChanceBonus += 0.018
	}
	if p.EmployeeCount >= 6 && (p.FinanceCount+p.LegalCount) == 0 {
		impact.RevenueMultiplier *= 0.94
		impact.RiskMultiplier += 0.22
		impact.CrisisChanceBonus += 0.025
	}
	if p.EmployeeCount >= 5 && p.EngineerCount == 0 {
		impact.RevenueMultiplier *= 0.93
	}
	if p.EmployeeCount >= 5 && (p.SalesCount+p.GrowthCount) == 0 {
		impact.RevenueMultiplier *= 0.92
	}

	upkeepReduction := math.Min(0.18, float64(p.OpsCount)*0.012+float64(p.AutomationLevel)*0.005)
	impact.MachineUpkeepFactor = 1 - upkeepReduction

	riskReduction := math.Min(0.24, float64(p.FinanceCount)*0.03+float64(p.LegalCount)*0.035+float64(p.ComplianceLevel)*0.015)
	impact.RiskMultiplier *= 1 - riskReduction
	if impact.RiskMultiplier < 0.68 {
		impact.RiskMultiplier = 0.68
	}

	impact.ReserveYieldFactor = 1 + math.Min(0.16, float64(p.FinanceCount)*0.018+float64(p.RDLevel)*0.006)
	impact.LaunchChanceBonus = math.Min(0.025, float64(productPairs)*0.0025+float64(p.RDLevel)*0.001)
	impact.DemandChanceBonus = math.Min(0.030, float64(goToMarketPairs)*0.003+float64(p.MarketingLevel)*0.0008)
	impact.ViralChanceBonus = math.Min(0.020, float64(p.DesignCount)*0.0012+float64(p.MarketingLevel)*0.0006)

	return impact
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
