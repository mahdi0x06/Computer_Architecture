package cache

import (
	"fmt"

	"victimcacheproject/internal/config"
	"victimcacheproject/internal/model"
)

type l2Set struct {
	ways []model.Block
}

// L2 is a set-associative cache with deterministic FIFO replacement. FIFO is
// implemented with each block's Inserted sequence number. Hits do not change
// replacement order.
type L2 struct {
	cfg   config.Config
	sets  []l2Set
	clock uint64
}

func NewL2(cfg config.Config) *L2 {
	if err := cfg.ValidateL2(); err != nil {
		panic(fmt.Sprintf("invalid L2 configuration: %v", err))
	}

	numSets := int(cfg.L2SizeBytes / (cfg.BlockSizeBytes * uint64(cfg.L2Associativity)))
	sets := make([]l2Set, numSets)
	for i := range sets {
		sets[i].ways = make([]model.Block, cfg.L2Associativity)
	}

	return &L2{cfg: cfg, sets: sets}
}

func (c *L2) Name() string { return "L2" }

func (c *L2) Lookup(req model.Request) LookupResult {
	blockAddress := req.Address / c.cfg.BlockSizeBytes
	setIndex, tag := c.decode(blockAddress)

	for wayIndex := range c.sets[setIndex].ways {
		block := c.sets[setIndex].ways[wayIndex]
		if block.Valid && block.Tag == tag && block.Address == blockAddress {
			c.clock++
			c.sets[setIndex].ways[wayIndex].LastUsed = c.clock
			matched := c.sets[setIndex].ways[wayIndex]
			return LookupResult{Hit: true, Block: &matched}
		}
	}

	return LookupResult{}
}

func (c *L2) Insert(block model.Block) *model.Block {
	setIndex, tag := c.decode(block.Address)
	set := &c.sets[setIndex]

	// Updating an existing block does not change its FIFO position.
	for wayIndex := range set.ways {
		current := set.ways[wayIndex]
		if current.Valid && current.Tag == tag && current.Address == block.Address {
			c.clock++
			block.Valid = true
			block.Tag = tag
			block.Inserted = current.Inserted
			block.LastUsed = c.clock
			set.ways[wayIndex] = block
			return nil
		}
	}

	c.clock++
	block.Valid = true
	block.Tag = tag
	block.Inserted = c.clock
	block.LastUsed = c.clock

	// Fill the lowest-numbered empty way first for deterministic behavior.
	for wayIndex := range set.ways {
		if !set.ways[wayIndex].Valid {
			set.ways[wayIndex] = block
			return nil
		}
	}

	victimWay := 0
	oldestInsertion := set.ways[0].Inserted
	for wayIndex := 1; wayIndex < len(set.ways); wayIndex++ {
		if set.ways[wayIndex].Inserted < oldestInsertion {
			victimWay = wayIndex
			oldestInsertion = set.ways[wayIndex].Inserted
		}
	}

	evicted := set.ways[victimWay]
	set.ways[victimWay] = block
	return &evicted
}

func (c *L2) Invalidate(blockAddress uint64) bool {
	setIndex, tag := c.decode(blockAddress)
	set := &c.sets[setIndex]

	for wayIndex := range set.ways {
		block := set.ways[wayIndex]
		if block.Valid && block.Tag == tag && block.Address == blockAddress {
			set.ways[wayIndex] = model.Block{}
			return true
		}
	}

	return false
}

func (c *L2) decode(blockAddress uint64) (setIndex int, tag uint64) {
	numSets := uint64(len(c.sets))
	return int(blockAddress % numSets), blockAddress / numSets
}
