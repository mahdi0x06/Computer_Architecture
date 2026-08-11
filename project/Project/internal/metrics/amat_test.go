package metrics

import "testing"

func TestCalculateAMAT(t *testing.T) {
	got := CalculateAMAT(AMATInputs{
		L1HitTime: 1, L1MissRate: 0.2,
		VictimHitRate: 0.5, VictimPenalty: 2,
		VictimMissRate: 0.5, L2Penalty: 10,
	})
	want := 2.2
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCalculateBaselineAMAT(t *testing.T) {
	got := CalculateBaselineAMAT(1, 0.25, 32)
	want := 9.0
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
