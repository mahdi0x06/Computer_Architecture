package cache

import (
	"testing"

	"victimcacheproject/internal/config"
	"victimcacheproject/internal/model"
)

func smallL2Config() config.Config {
	cfg := config.Default()
	cfg.BlockSizeBytes = 64
	cfg.L2Associativity = 2
	cfg.L2SizeBytes = 4 * cfg.BlockSizeBytes // Two sets, two ways per set.
	return cfg
}

func TestL2HitAndMiss(t *testing.T) {
	cfg := smallL2Config()
	cache := NewL2(cfg)

	if cache.Lookup(requestForBlock(1, 2, cfg.BlockSizeBytes)).Hit {
		t.Fatal("empty L2 must miss")
	}
	cache.Insert(model.NewBlock(2))
	if !cache.Lookup(requestForBlock(2, 2, cfg.BlockSizeBytes)).Hit {
		t.Fatal("inserted L2 block must hit")
	}
}

func TestL2SetAssociativityAndFIFOEviction(t *testing.T) {
	cfg := smallL2Config()
	cache := NewL2(cfg)

	// Two sets means blocks 0, 2, and 4 all map to set 0.
	if evicted := cache.Insert(model.NewBlock(0)); evicted != nil {
		t.Fatalf("unexpected first eviction: %+v", evicted)
	}
	if evicted := cache.Insert(model.NewBlock(2)); evicted != nil {
		t.Fatalf("unexpected second eviction: %+v", evicted)
	}

	// A hit must not change FIFO insertion order.
	if !cache.Lookup(requestForBlock(1, 0, cfg.BlockSizeBytes)).Hit {
		t.Fatal("expected block 0 hit before replacement")
	}

	evicted := cache.Insert(model.NewBlock(4))
	if evicted == nil || evicted.Address != 0 {
		t.Fatalf("FIFO should evict block 0, got %+v", evicted)
	}
	if cache.Lookup(requestForBlock(2, 0, cfg.BlockSizeBytes)).Hit {
		t.Fatal("evicted block 0 must miss")
	}
	if !cache.Lookup(requestForBlock(3, 2, cfg.BlockSizeBytes)).Hit {
		t.Fatal("block 2 must remain in the set")
	}
	if !cache.Lookup(requestForBlock(4, 4, cfg.BlockSizeBytes)).Hit {
		t.Fatal("new block 4 must hit")
	}
}

func TestL2Invalidate(t *testing.T) {
	cfg := smallL2Config()
	cache := NewL2(cfg)
	cache.Insert(model.NewBlock(1))

	if !cache.Invalidate(1) {
		t.Fatal("expected L2 invalidation to succeed")
	}
	if cache.Lookup(requestForBlock(1, 1, cfg.BlockSizeBytes)).Hit {
		t.Fatal("invalidated L2 block must miss")
	}
}
