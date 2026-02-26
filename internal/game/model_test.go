package game

import "testing"

func TestValidateSymbol(t *testing.T) {
	valid := []string{"ABCDEF", "NIMBUS", "COBOLT"}
	for _, s := range valid {
		if err := ValidateSymbol(s); err != nil {
			t.Fatalf("expected symbol %q to be valid: %v", s, err)
		}
	}

	invalid := []string{"abc123", "ABC12", "TOOLONG7", "A_BCD1"}
	for _, s := range invalid {
		if err := ValidateSymbol(s); err == nil {
			t.Fatalf("expected symbol %q to fail", s)
		}
	}
}

func TestDebtLimitFromPeak(t *testing.T) {
	tests := []struct {
		peak int64
		want int64
	}{
		{peak: 0, want: MinDebtLimitMicros},
		{peak: 10_000 * MicrosPerStonky, want: MinDebtLimitMicros},
		{peak: 1_000_000 * MicrosPerStonky, want: MaxDebtLimitMicros},
	}
	for _, tc := range tests {
		got := DebtLimitFromPeak(tc.peak)
		if got != tc.want {
			t.Fatalf("peak=%d got=%d want=%d", tc.peak, got, tc.want)
		}
	}
}

func TestNotionalMicros(t *testing.T) {
	price := int64(150 * MicrosPerStonky)
	qty := int64(25 * ShareScale / 10) // 2.5 shares
	got, err := notionalMicros(price, qty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := int64(375 * MicrosPerStonky)
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestValidateEntityName(t *testing.T) {
	if err := validateEntityName("Acme Labs"); err != nil {
		t.Fatalf("expected valid entity name: %v", err)
	}
	if err := validateEntityName("admin empire"); err == nil {
		t.Fatalf("expected blocked name to fail")
	}
}

func TestMaxAffordableBuy(t *testing.T) {
	price := int64(840 * MicrosPerStonky)
	balance := int64(19_025) * MicrosPerStonky
	debt := DebtLimitFromPeak(25_000 * MicrosPerStonky)

	units, notional, fee := maxAffordableBuy(price, balance, debt)
	if units <= 0 {
		t.Fatalf("expected affordable units > 0")
	}
	total := notional + fee
	if total > balance+debt {
		t.Fatalf("total %d exceeds budget %d", total, balance+debt)
	}

	nextUnits := units + 1
	nextNotional, err := notionalMicros(price, nextUnits)
	if err != nil {
		t.Fatalf("notional error: %v", err)
	}
	nextFee := int64(0.0015*float64(nextNotional) + 0.5)
	if nextNotional+nextFee <= balance+debt {
		t.Fatalf("expected units+1 to be unaffordable")
	}
}
