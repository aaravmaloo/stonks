package game

import (
	"fmt"
	"math"
	"strings"
)

func businessCycleRevenueMultiplier(c businessCycle) float64 {
	return 1 + float64(c.cycleImpactBps)/10000.0
}

func businessCycleCostMultiplier(c businessCycle) float64 {
	switch c.cyclePhase {
	case "slump":
		return 1.08 + math.Abs(float64(c.cycleImpactBps))/22000.0
	case "squeeze":
		return 1.12 + math.Abs(float64(c.cycleImpactBps))/18000.0
	case "recovery":
		return 0.97
	case "boom":
		return 0.95
	default:
		return 1
	}
}

func rollBusinessCycle(c businessCycle, world marketWorldState, seed float64) (string, int32, int32, string) {
	label := strings.TrimSpace(c.name)
	if label == "" {
		label = "Business"
	}
	hotBias := regionTrend(world, c.primaryRegion) + float64(world.RiskRewardBiasBps)/14000.0
	pressureBias := float64(c.narrativePressure) / 14000.0
	riskBias := c.avgRiskBps / 15000.0
	score := hotBias + pressureBias - riskBias

	switch {
	case score > 0.28 || seed > 0.90:
		impact := int32(1200 + math.Round((seed-0.5)*1800))
		return "boom", 4 + int32(seed*3), clampBps(impact, 900, 3400), fmt.Sprintf("%s hit a profit boom cycle", label)
	case score < -0.10 && seed < 0.42:
		impact := -int32(1300 + math.Round((0.5-seed)*2200))
		return "slump", 3 + int32(seed*4), clampBps(impact, -3600, -1000), fmt.Sprintf("%s slipped into a loss period", label)
	case c.strategy == "aggressive" && (seed < 0.18 || score < 0):
		impact := -int32(1700 + math.Round(seed*1600))
		return "squeeze", 2 + int32(seed*3), clampBps(impact, -4200, -1500), fmt.Sprintf("%s is getting squeezed by its own pace", label)
	case c.narrativeArc == "turnaround" || score > 0.05:
		impact := int32(500 + math.Round(seed*1000))
		return "recovery", 3 + int32(seed*3), clampBps(impact, 300, 1800), fmt.Sprintf("%s is moving into recovery", label)
	default:
		impact := int32(math.Round((seed - 0.5) * 600))
		return "stable", 4 + int32(seed*3), clampBps(impact, -700, 700), fmt.Sprintf("%s is holding a stable cycle", label)
	}
}
