package game

import "testing"

func TestNextRushVaultTarget(t *testing.T) {
	if got := nextRushVaultTarget(0); got != 90 {
		t.Fatalf("expected base vault target 90, got %d", got)
	}
	if got := nextRushVaultTarget(3); got != 225 {
		t.Fatalf("expected scaled vault target 225, got %d", got)
	}
}

func TestRushMilestoneReward(t *testing.T) {
	if got := rushMilestoneReward(3, 50*MicrosPerStonky); got < 120*MicrosPerStonky {
		t.Fatalf("expected streak reward floor, got %d", got)
	}
	if got := rushMilestoneReward(4, 50*MicrosPerStonky); got != 0 {
		t.Fatalf("expected no milestone reward at streak 4, got %d", got)
	}
}
