package cache

import (
	"fmt"

	"victimcacheproject/internal/config"
	"victimcacheproject/internal/model"
)

// L1 is a direct-mapped cache. It models only lookup, insertion, eviction, and
// invalidation. It never calls L2 or Main Memory directly.
type L1 struct {
	cfg   config.Config
	lines []model.Block
	clock uint64
}

func NewL1(cfg config.Config) *L1 {
	if err := cfg.ValidateL1(); err != nil {
		panic(fmt.Sprintf("invalid L1 configuration: %v", err))
	}

	numLines := int(cfg.L1SizeBytes / cfg.BlockSizeBytes)
	return &L1{
		cfg:   cfg,
		lines: make([]model.Block, numLines),
	}
}

func (c *L1) Name() string { return "L1" }

func (c *L1) Lookup(req model.Request) LookupResult {
	blockAddress := req.Address / c.cfg.BlockSizeBytes
	index, tag := c.decode(blockAddress)
	line := c.lines[index]

	if !line.Valid || line.Tag != tag || line.Address != blockAddress {
		return LookupResult{}
	}

	c.clock++
	c.lines[index].LastUsed = c.clock
	matched := c.lines[index]
	return LookupResult{Hit: true, Block: &matched}
}

func (c *L1) Insert(block model.Block) *model.Block {
	index, tag := c.decode(block.Address)
	current := c.lines[index]

	c.clock++
	block.Valid = true
	block.Tag = tag
	block.LastUsed = c.clock

	// Updating the same block is not an eviction.
	if current.Valid && current.Address == block.Address && current.Tag == tag {
		block.Inserted = current.Inserted
		c.lines[index] = block
		return nil
	}

	block.Inserted = c.clock
	c.lines[index] = block

	if !current.Valid {
		return nil
	}

	evicted := current
	return &evicted
}

func (c *L1) Invalidate(blockAddress uint64) bool {
	index, tag := c.decode(blockAddress)
	line := c.lines[index]
	if !line.Valid || line.Tag != tag || line.Address != blockAddress {
		return false
	}

	c.lines[index] = model.Block{}
	return true
}

func (c *L1) decode(blockAddress uint64) (index int, tag uint64) {
	numLines := uint64(len(c.lines))
	return int(blockAddress % numLines), blockAddress / numLines
}
