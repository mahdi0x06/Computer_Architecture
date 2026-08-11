package cpu

import "victimcacheproject/internal/model"

// Generator emits deterministic memory requests.
type Generator struct {
	requests []model.Request
	next     int
}

func NewGenerator(requests []model.Request) *Generator {
	// Copy input slice.
	reqCopy := make([]model.Request, len(requests))
	copy(reqCopy, requests)

	return &Generator{
		requests: reqCopy,
		next:     0,
	}
}

func (g *Generator) HasNext() bool {
	return g.next < len(g.requests)
}

func (g *Generator) Next() (model.Request, bool) {
	if !g.HasNext() {
		return model.Request{}, false
	}
	req := g.requests[g.next]
	g.next++
	return req, true
}

// Reset state.
func (g *Generator) Reset() {
	g.next = 0
}

// Remaining count.
func (g *Generator) Remaining() int {
	return len(g.requests) - g.next
}

// Total count.
func (g *Generator) Total() int {
	return len(g.requests)
}
