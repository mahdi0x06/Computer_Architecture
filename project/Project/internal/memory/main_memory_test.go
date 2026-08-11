package memory

import (
	"testing"

	"victimcacheproject/internal/model"
)

func TestReadBlockCreatesValidBlock(t *testing.T) {
	memory := NewMainMemory()

	block := memory.ReadBlock(42)
	if !block.Valid {
		t.Fatal("expected a valid block")
	}
	if block.Address != 42 {
		t.Fatalf("got block address %d, want 42", block.Address)
	}
	if block.Dirty {
		t.Fatal("a block read from main memory must be clean")
	}
}

func TestWriteThenReadBlock(t *testing.T) {
	memory := NewMainMemory()
	memory.WriteBlock(model.Block{
		Address:  7,
		Valid:    true,
		Dirty:    true,
		LastUsed: 99,
	})

	block := memory.ReadBlock(7)
	if block.Address != 7 || block.LastUsed != 99 {
		t.Fatalf("stored block metadata was not preserved: %+v", block)
	}
	if block.Dirty {
		t.Fatal("main-memory write must persist and clear the dirty state")
	}
}
