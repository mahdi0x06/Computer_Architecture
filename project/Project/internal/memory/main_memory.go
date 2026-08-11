package memory

import "victimcacheproject/internal/model"

// MainMemory is a minimal deterministic backing store indexed by block address.
// It models functional block persistence but deliberately does not model Akita
// events or latency; those belong to the system/adapter layers.
type MainMemory struct {
	blocks map[uint64]model.Block
}

func NewMainMemory() *MainMemory {
	return &MainMemory{blocks: make(map[uint64]model.Block)}
}

func (m *MainMemory) ReadBlock(blockAddress uint64) model.Block {
	if block, found := m.blocks[blockAddress]; found {
		block.Valid = true
		block.Dirty = false
		return block
	}

	block := model.NewBlock(blockAddress)
	m.blocks[blockAddress] = block
	return block
}

func (m *MainMemory) WriteBlock(block model.Block) {
	block.Valid = true
	// Main Memory is the point where dirty data is considered persisted.
	block.Dirty = false
	m.blocks[block.Address] = block
}
