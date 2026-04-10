package game

import "testing"

func TestBusinessCycleRevenueMultiplier(t *testing.T) {
	c := businessCycle{cyclePhase: "boom", cycleImpactBps: 2200}
	if got := businessCycleRevenueMultiplier(c); got <= 1.0 {
		t.Fatalf("expected boom multiplier > 1, got %f", got)
	}
}

func TestRollBusinessCycleCanCreateLossPeriod(t *testing.T) {
	phase, ticks, impact, _ := rollBusinessCycle(businessCycle{
		name:              "Acme",
		primaryRegion:     "americas",
		narrativeArc:      "fragile",
		narrativeFocus:    "finance",
		narrativePressure: 9000,
		strategy:          "aggressive",
		avgRiskBps:        7000,
	}, marketWorldState{RiskRewardBiasBps: -1600}, 0.10)
	if phase == "boom" {
		t.Fatalf("expected stressed setup to avoid boom, got %s", phase)
	}
	if ticks <= 0 {
		t.Fatalf("expected positive cycle duration, got %d", ticks)
	}
	if impact == 0 {
		t.Fatalf("expected non-zero cycle impact")
	}
}
