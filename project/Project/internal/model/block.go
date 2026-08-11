package model

// Block represents one cache block throughout the project.
//
// Address is a block address (byteAddress / blockSize), not a byte address.
// Each cache computes its own index and tag from this value. Keeping one shared
// representation prevents L1, L2, Victim Cache, and Main Memory from drifting
// into incompatible data models.
type Block struct {
	Address  uint64
	Tag      uint64
	Valid    bool
	Dirty    bool
	LastUsed uint64
	Inserted uint64
}

// NewBlock creates a valid, clean block with the supplied block address.
func NewBlock(blockAddress uint64) Block {
	return Block{
		Address: blockAddress,
		Valid:   true,
	}
}
