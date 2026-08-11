package cache

import (
	"errors"
	"testing"

	"victimcacheproject/internal/config"
	"victimcacheproject/internal/model"
)

func victimConfig(entries int, policy string) config.Config {
	cfg := config.Default()
	cfg.BlockSizeBytes = 64
	cfg.VictimEnabled = true
	cfg.VictimEntries = entries
	cfg.VictimPolicy = policy
	return cfg
}

func TestVictimMissOnEmptyAndHitAfterInsert(t *testing.T) {
	cfg := victimConfig(2, config.ReplacementFIFO)
	cache := NewVictim(cfg)

	if cache.Lookup(requestForBlock(1, 10, cfg.BlockSizeBytes)).Hit {
		t.Fatal("empty Victim Cache must miss")
	}
	cache.Insert(model.NewBlock(10))
	result := cache.Lookup(requestForBlock(2, 10, cfg.BlockSizeBytes))
	if !result.Hit || result.Block == nil || result.Block.Address != 10 {
		t.Fatalf("expected Victim Cache hit, got %+v", result)
	}
}

func TestVictimIsFullyAssociativeAndUsesFIFO(t *testing.T) {
	cfg := victimConfig(2, config.ReplacementFIFO)
	cache := NewVictim(cfg)
	cache.Insert(model.NewBlock(1))
	cache.Insert(model.NewBlock(1001))

	// Both unrelated addresses coexist because there are no sets or indices.
	if !cache.Lookup(requestForBlock(1, 1, cfg.BlockSizeBytes)).Hit ||
		!cache.Lookup(requestForBlock(2, 1001, cfg.BlockSizeBytes)).Hit {
		t.Fatal("Victim Cache must search all entries")
	}

	evicted := cache.Insert(model.NewBlock(55))
	if evicted == nil || evicted.Address != 1 {
		t.Fatalf("FIFO should evict block 1, got %+v", evicted)
	}
}

func TestVictimLRUEvictsLeastRecentlyUsed(t *testing.T) {
	cfg := victimConfig(2, config.ReplacementLRU)
	cache := NewVictim(cfg)
	cache.Insert(model.NewBlock(10))
	cache.Insert(model.NewBlock(20))
	cache.Lookup(requestForBlock(1, 10, cfg.BlockSizeBytes)) // Make 10 most recently used.

	evicted := cache.Insert(model.NewBlock(30))
	if evicted == nil || evicted.Address != 20 {
		t.Fatalf("LRU should evict block 20, got %+v", evicted)
	}
}

func TestVictimLRUPreservesRecencyFromL1Eviction(t *testing.T) {
	cfg := victimConfig(2, config.ReplacementLRU)
	cache := NewVictim(cfg)

	// Block 10 enters first but was used much more recently while resident in
	// L1. Block 20 enters later but carries an older L1 LastUsed timestamp.
	recent := model.NewBlock(10)
	recent.LastUsed = 100
	stale := model.NewBlock(20)
	stale.LastUsed = 10
	cache.Insert(recent)
	cache.Insert(stale)

	incoming := model.NewBlock(30)
	incoming.LastUsed = 200
	evicted := cache.Insert(incoming)
	if evicted == nil || evicted.Address != 20 {
		t.Fatalf("LRU should preserve recently used block 10 and evict stale block 20, got %+v", evicted)
	}
}

func TestVictimRemove(t *testing.T) {
	cfg := victimConfig(2, config.ReplacementFIFO)
	cache := NewVictim(cfg)
	cache.Insert(model.NewBlock(8))

	removed, found := cache.Remove(8)
	if !found || removed.Address != 8 {
		t.Fatalf("unexpected remove result: block=%+v found=%v", removed, found)
	}
	if cache.Lookup(requestForBlock(1, 8, cfg.BlockSizeBytes)).Hit {
		t.Fatal("removed block must miss")
	}
}

func TestVictimSwap(t *testing.T) {
	cfg := victimConfig(2, config.ReplacementFIFO)
	cache := NewVictim(cfg)
	cache.Insert(model.NewBlock(10))
	cache.Insert(model.NewBlock(20))

	requested, err := cache.Swap(10, model.NewBlock(30))
	if err != nil {
		t.Fatalf("swap failed: %v", err)
	}
	if requested.Address != 10 {
		t.Fatalf("swap returned block %d, want 10", requested.Address)
	}
	if cache.Lookup(requestForBlock(1, 10, cfg.BlockSizeBytes)).Hit {
		t.Fatal("requested block must be removed from Victim Cache")
	}
	if !cache.Lookup(requestForBlock(2, 20, cfg.BlockSizeBytes)).Hit ||
		!cache.Lookup(requestForBlock(3, 30, cfg.BlockSizeBytes)).Hit {
		t.Fatal("Victim Cache must retain block 20 and receive L1 eviction 30")
	}
}

func TestVictimSwapErrors(t *testing.T) {
	cfg := victimConfig(1, config.ReplacementFIFO)
	cache := NewVictim(cfg)

	if _, err := cache.Swap(99, model.NewBlock(1)); !errors.Is(err, ErrBlockNotFound) {
		t.Fatalf("got error %v, want ErrBlockNotFound", err)
	}

	cfg.VictimEnabled = false
	disabled := NewVictim(cfg)
	if _, err := disabled.Swap(1, model.NewBlock(2)); !errors.Is(err, ErrVictimDisabled) {
		t.Fatalf("got error %v, want ErrVictimDisabled", err)
	}
}

func TestDisabledVictimReturnsIncomingBlock(t *testing.T) {
	cfg := victimConfig(2, config.ReplacementFIFO)
	cfg.VictimEnabled = false
	cache := NewVictim(cfg)
	incoming := model.NewBlock(12)

	overflow := cache.Insert(incoming)
	if overflow == nil || overflow.Address != 12 {
		t.Fatalf("disabled Victim Cache must return incoming block, got %+v", overflow)
	}
	if cache.Lookup(requestForBlock(1, 12, cfg.BlockSizeBytes)).Hit {
		t.Fatal("disabled Victim Cache must never hit")
	}
}
