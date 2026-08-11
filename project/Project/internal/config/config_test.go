package config

import "testing"

func TestDefaultMemoryHierarchyConfigurationIsValid(t *testing.T) {
	if err := Default().ValidateMemoryHierarchy(); err != nil {
		t.Fatalf("default configuration must be valid: %v", err)
	}
}

func TestVictimPolicyIsCaseInsensitive(t *testing.T) {
	cfg := Default()
	cfg.VictimPolicy = " lru "
	if err := cfg.ValidateVictim(); err != nil {
		t.Fatalf("normalized LRU policy should be valid: %v", err)
	}
	if got := cfg.NormalizedVictimPolicy(); got != ReplacementLRU {
		t.Fatalf("got policy %q, want %q", got, ReplacementLRU)
	}
}
