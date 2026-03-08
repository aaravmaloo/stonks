package game

import "testing"

func TestCandidatePoolScales(t *testing.T) {
	got := candidatePool(seededCandidatePoolSize)
	if len(got) != seededCandidatePoolSize {
		t.Fatalf("candidate pool size = %d, want %d", len(got), seededCandidatePoolSize)
	}

	if got[0].Role == "" || got[len(got)-1].Trait == "" {
		t.Fatalf("expected generated candidates to be populated")
	}
}
