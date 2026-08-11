package cache

import (
	"testing"

	"victimcacheproject/internal/config"
	"victimcacheproject/internal/model"
)

func smallL1Config() config.Config {
	cfg := config.Default()
	cfg.BlockSizeBytes = 64
	cfg.L1SizeBytes = 4 * cfg.BlockSizeBytes // Four direct-mapped lines.
	cfg.L1Associativity = 1
	return cfg
}

func requestForBlock(id, blockAddress, blockSize uint64) model.Request {
	return model.Request{
		ID:      id,
		Address: blockAddress*blockSize + 3, // Include a byte offset deliberately.
		Op:      model.Read,
		Size:    4,
	}
}

func TestL1MissOnEmptyCache(t *testing.T) {
	cfg := smallL1Config()
	cache := NewL1(cfg)

	result := cache.Lookup(requestForBlock(1, 2, cfg.BlockSizeBytes))
	if result.Hit || result.Block != nil {
		t.Fatalf("empty L1 lookup must miss, got %+v", result)
	}
}

func TestL1HitAfterInsert(t *testing.T) {
	cfg := smallL1Config()
	cache := NewL1(cfg)

	if evicted := cache.Insert(model.NewBlock(2)); evicted != nil {
		t.Fatalf("first insertion unexpectedly evicted %+v", evicted)
	}

	result := cache.Lookup(requestForBlock(1, 2, cfg.BlockSizeBytes))
	if !result.Hit || result.Block == nil || result.Block.Address != 2 {
		t.Fatalf("expected L1 hit for block 2, got %+v", result)
	}
}

func TestL1ConflictReplacementReturnsEvictedBlock(t *testing.T) {
	cfg := smallL1Config()
	cache := NewL1(cfg)

	// Four lines means block 1 and block 5 map to the same index.
	cache.Insert(model.NewBlock(1))
	evicted := cache.Insert(model.NewBlock(5))
	if evicted == nil || evicted.Address != 1 {
		t.Fatalf("got eviction %+v, want block 1", evicted)
	}

	if cache.Lookup(requestForBlock(1, 1, cfg.BlockSizeBytes)).Hit {
		t.Fatal("replaced block 1 must miss")
	}
	if !cache.Lookup(requestForBlock(2, 5, cfg.BlockSizeBytes)).Hit {
		t.Fatal("replacement block 5 must hit")
	}
}

func TestL1UpdatingSameBlockDoesNotEvict(t *testing.T) {
	cfg := smallL1Config()
	cache := NewL1(cfg)
	cache.Insert(model.NewBlock(3))

	updated := model.NewBlock(3)
	updated.Dirty = true
	if evicted := cache.Insert(updated); evicted != nil {
		t.Fatalf("updating the same block unexpectedly evicted %+v", evicted)
	}

	result := cache.Lookup(requestForBlock(1, 3, cfg.BlockSizeBytes))
	if !result.Hit || result.Block == nil || !result.Block.Dirty {
		t.Fatalf("updated block was not retained: %+v", result)
	}
}

func TestL1Invalidate(t *testing.T) {
	cfg := smallL1Config()
	cache := NewL1(cfg)
	cache.Insert(model.NewBlock(2))

	if !cache.Invalidate(2) {
		t.Fatal("expected block 2 to be invalidated")
	}
	if cache.Invalidate(2) {
		t.Fatal("invalidating block 2 twice must return false the second time")
	}
	if cache.Lookup(requestForBlock(1, 2, cfg.BlockSizeBytes)).Hit {
		t.Fatal("invalidated block must miss")
	}
}
