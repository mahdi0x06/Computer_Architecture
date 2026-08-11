package cache

import "victimcacheproject/internal/model"

// Cache defines logical cache behavior independently from Akita messaging and
// independently from the next level in the memory hierarchy.
type Cache interface {
	Name() string
	Lookup(req model.Request) LookupResult
	Insert(block model.Block) (evicted *model.Block)
	Invalidate(blockAddress uint64) bool
}

// LookupResult returns a copy of the matched block. Callers cannot mutate the
// cache's internal storage through the returned pointer.
type LookupResult struct {
	Hit   bool
	Block *model.Block
}

var (
	_ Cache = (*L1)(nil)
	_ Cache = (*L2)(nil)
	_ Cache = (*Victim)(nil)
)
