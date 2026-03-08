package game

import "testing"

func TestAnalyzeWorkforceRewardsBalancedTeams(t *testing.T) {
	impact := analyzeWorkforce(workforceProfile{
		EmployeeCount:   10,
		OpsCount:        2,
		EngineerCount:   3,
		ProductCount:    2,
		SalesCount:      2,
		GrowthCount:     2,
		FinanceCount:    1,
		LegalCount:      1,
		DesignCount:     1,
		MarketingLevel:  2,
		RDLevel:         2,
		AutomationLevel: 2,
		ComplianceLevel: 2,
	})

	if impact.RevenueMultiplier <= 1 {
		t.Fatalf("expected balanced team to improve revenue, got %f", impact.RevenueMultiplier)
	}
	if impact.RiskMultiplier >= 1 {
		t.Fatalf("expected balanced team to reduce risk, got %f", impact.RiskMultiplier)
	}
	if impact.MachineUpkeepFactor >= 1 {
		t.Fatalf("expected balanced team to cut upkeep, got %f", impact.MachineUpkeepFactor)
	}
}

func TestAnalyzeWorkforcePunishesMissingGovernance(t *testing.T) {
	impact := analyzeWorkforce(workforceProfile{
		EmployeeCount:   8,
		EngineerCount:   3,
		ProductCount:    2,
		SalesCount:      2,
		GrowthCount:     1,
		MarketingLevel:  1,
		RDLevel:         1,
		AutomationLevel: 1,
	})

	if impact.RiskMultiplier <= 1 {
		t.Fatalf("expected missing governance to increase risk, got %f", impact.RiskMultiplier)
	}
	if impact.CrisisChanceBonus <= 0 {
		t.Fatalf("expected missing governance to add crisis chance, got %f", impact.CrisisChanceBonus)
	}
}
