package game

import "testing"

func TestCandidatePoolScales(t *testing.T) {
	got := candidatePool(0, seededCandidatePoolSize)
	if len(got) != seededCandidatePoolSize {
		t.Fatalf("candidate pool size = %d, want %d", len(got), seededCandidatePoolSize)
	}

	if got[0].Role == "" || got[len(got)-1].Trait == "" {
		t.Fatalf("expected generated candidates to be populated")
	}
}

func TestCandidatePoolOffsetChangesOutput(t *testing.T) {
	got := candidatePool(10, 2)
	if len(got) != 2 {
		t.Fatalf("candidate pool size = %d, want 2", len(got))
	}
	if got[0].Name == got[1].Name && got[0].Trait == got[1].Trait {
		t.Fatalf("expected offset-generated candidates to vary")
	}
}
