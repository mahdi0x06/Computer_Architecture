package cache

import (
	"fmt"

	"victimcacheproject/internal/config"
	"victimcacheproject/internal/model"
)

// Victim is a small fully-associative cache. It supports FIFO and LRU
// replacement, selected through Config.VictimPolicy.
type Victim struct {
	cfg     config.Config
	entries []model.Block
	clock   uint64
}

func NewVictim(cfg config.Config) *Victim {
	if err := cfg.ValidateVictim(); err != nil {
		panic(fmt.Sprintf("invalid victim cache configuration: %v", err))
	}

	return &Victim{
		cfg:     cfg,
		entries: make([]model.Block, cfg.VictimEntries),
	}
}

func (c *Victim) Name() string { return "VictimCache" }

func (c *Victim) Lookup(req model.Request) LookupResult {
	if !c.cfg.VictimEnabled {
		return LookupResult{}
	}

	blockAddress := req.Address / c.cfg.BlockSizeBytes
	for entryIndex := range c.entries {
		entry := c.entries[entryIndex]
		if entry.Valid && entry.Address == blockAddress {
			c.clock++
			c.entries[entryIndex].LastUsed = c.clock
			matched := c.entries[entryIndex]
			return LookupResult{Hit: true, Block: &matched}
		}
	}

	return LookupResult{}
}

func (c *Victim) Insert(block model.Block) *model.Block {
	// A disabled or zero-capacity Victim Cache must not swallow an L1 eviction.
	// Returning the incoming block lets the caller forward it to L2.
	if !c.cfg.VictimEnabled || len(c.entries) == 0 {
		overflow := block
		return &overflow
	}

	// Updating an existing block is not an eviction and preserves FIFO age.
	for entryIndex := range c.entries {
		current := c.entries[entryIndex]
		if current.Valid && current.Address == block.Address {
			c.clock++
			block.Valid = true
			block.Tag = block.Address
			block.Inserted = current.Inserted
			block.LastUsed = c.clock
			c.entries[entryIndex] = block
			return nil
		}
	}

	c.clock++
	block.Valid = true
	// Victim Cache is fully associative, so the complete block address serves
	// as a convenient tag.
	block.Tag = block.Address
	block.Inserted = c.clock
	// For a real LRU distinction, preserve the recency carried by an L1
	// eviction. FIFO ignores this field and uses Inserted. Direct cache-level
	// tests may insert a fresh block with no recency metadata, so fall back to
	// the Victim Cache clock in that case.
	if block.LastUsed == 0 {
		block.LastUsed = c.clock
	}

	for entryIndex := range c.entries {
		if !c.entries[entryIndex].Valid {
			c.entries[entryIndex] = block
			return nil
		}
	}

	victimIndex := c.replacementIndex()
	evicted := c.entries[victimIndex]
	c.entries[victimIndex] = block
	return &evicted
}

func (c *Victim) Invalidate(blockAddress uint64) bool {
	_, found := c.Remove(blockAddress)
	return found
}

// Remove extracts a block from the Victim Cache. It is the small extra API
// needed by the system layer to orchestrate an L1/Victim swap.
func (c *Victim) Remove(blockAddress uint64) (model.Block, bool) {
	if !c.cfg.VictimEnabled {
		return model.Block{}, false
	}

	for entryIndex := range c.entries {
		entry := c.entries[entryIndex]
		if entry.Valid && entry.Address == blockAddress {
			c.entries[entryIndex] = model.Block{}
			return entry, true
		}
	}

	return model.Block{}, false
}

// Swap atomically removes the requested Victim block and inserts the block
// evicted from L1. The returned block is the one that the caller should insert
// into L1. If l1Evicted is invalid, Swap simply removes the requested block.
func (c *Victim) Swap(requestedBlockAddress uint64, l1Evicted model.Block) (model.Block, error) {
	if !c.cfg.VictimEnabled {
		return model.Block{}, ErrVictimDisabled
	}

	requested, found := c.Remove(requestedBlockAddress)
	if !found {
		return model.Block{}, ErrBlockNotFound
	}

	if l1Evicted.Valid {
		// Remove created one empty entry, so a valid insertion cannot overflow.
		_ = c.Insert(l1Evicted)
	}

	return requested, nil
}

func (c *Victim) replacementIndex() int {
	victimIndex := 0

	switch c.cfg.NormalizedVictimPolicy() {
	case config.ReplacementLRU:
		oldestUse := c.entries[0].LastUsed
		for entryIndex := 1; entryIndex < len(c.entries); entryIndex++ {
			if c.entries[entryIndex].LastUsed < oldestUse {
				victimIndex = entryIndex
				oldestUse = c.entries[entryIndex].LastUsed
			}
		}
	default: // FIFO is validated by Config and is the project baseline.
		oldestInsertion := c.entries[0].Inserted
		for entryIndex := 1; entryIndex < len(c.entries); entryIndex++ {
			if c.entries[entryIndex].Inserted < oldestInsertion {
				victimIndex = entryIndex
				oldestInsertion = c.entries[entryIndex].Inserted
			}
		}
	}

	return victimIndex
}
