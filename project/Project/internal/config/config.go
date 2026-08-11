package config

import (
	"fmt"
	"strings"
)

const (
	ReplacementFIFO = "FIFO"
	ReplacementLRU  = "LRU"

	TopologyMemoryOnly = "memory"
	TopologyL1         = "l1"
	TopologyL1L2       = "l1-l2"
	TopologyFull       = "full"
)

type Config struct {
	Topology            string
	BlockSizeBytes      uint64
	L1SizeBytes         uint64
	L1Associativity     int
	VictimEntries       int
	VictimEnabled       bool
	VictimPolicy        string
	L2SizeBytes         uint64
	L2Associativity     int
	L1HitLatencyCycles  uint64
	VictimLatencyCycles uint64
	L2LatencyCycles     uint64
	MemoryLatencyCycles uint64
}

func Default() Config {
	return Config{Topology: TopologyFull, BlockSizeBytes: 64, L1SizeBytes: 4 * 1024, L1Associativity: 1,
		VictimEntries: 8, VictimEnabled: true, VictimPolicy: ReplacementFIFO,
		L2SizeBytes: 64 * 1024, L2Associativity: 8,
		L1HitLatencyCycles: 1, VictimLatencyCycles: 2, L2LatencyCycles: 12, MemoryLatencyCycles: 100}
}

func (c Config) NormalizedVictimPolicy() string {
	return strings.ToUpper(strings.TrimSpace(c.VictimPolicy))
}
func (c Config) NormalizedTopology() string { return strings.ToLower(strings.TrimSpace(c.Topology)) }
func (c Config) UsesL1() bool               { return c.NormalizedTopology() != TopologyMemoryOnly }
func (c Config) UsesL2() bool {
	t := c.NormalizedTopology()
	return t == TopologyL1L2 || t == TopologyFull
}
func (c Config) UsesVictim() bool {
	return c.NormalizedTopology() == TopologyFull && c.VictimEnabled && c.VictimEntries > 0
}

func (c Config) ValidateL1() error {
	if c.BlockSizeBytes == 0 {
		return fmt.Errorf("block size must be greater than zero")
	}
	if c.L1SizeBytes == 0 || c.L1SizeBytes%c.BlockSizeBytes != 0 {
		return fmt.Errorf("L1 size must be non-zero and divisible by block size")
	}
	if c.L1Associativity != 1 {
		return fmt.Errorf("L1 must be direct-mapped")
	}
	return nil
}
func (c Config) ValidateL2() error {
	if c.BlockSizeBytes == 0 {
		return fmt.Errorf("block size must be greater than zero")
	}
	if c.L2Associativity <= 0 {
		return fmt.Errorf("L2 associativity must be greater than zero")
	}
	x := c.BlockSizeBytes * uint64(c.L2Associativity)
	if c.L2SizeBytes == 0 || c.L2SizeBytes%x != 0 {
		return fmt.Errorf("L2 size must be divisible by block size times associativity")
	}
	return nil
}
func (c Config) ValidateVictim() error {
	if c.VictimEntries < 0 {
		return fmt.Errorf("victim entries cannot be negative")
	}
	p := c.NormalizedVictimPolicy()
	if p != ReplacementFIFO && p != ReplacementLRU {
		return fmt.Errorf("unsupported victim policy %q", c.VictimPolicy)
	}
	return nil
}
func (c Config) ValidateMemoryHierarchy() error {
	switch c.NormalizedTopology() {
	case TopologyMemoryOnly:
		return nil
	case TopologyL1:
		return c.ValidateL1()
	case TopologyL1L2:
		if err := c.ValidateL1(); err != nil {
			return err
		}
		return c.ValidateL2()
	case TopologyFull:
		if err := c.ValidateL1(); err != nil {
			return err
		}
		if err := c.ValidateL2(); err != nil {
			return err
		}
		return c.ValidateVictim()
	default:
		return fmt.Errorf("unsupported topology %q", c.Topology)
	}
}
