package cpu

import (
	"testing"
	"victimcacheproject/internal/model"
)

// 1. Empty Trace
func TestGenerator_EmptyTrace(t *testing.T) {
	g := NewGenerator([]model.Request{})

	if g.HasNext() {
		t.Errorf("Expected HasNext() to be false")
	}

	req, ok := g.Next()
	if ok {
		t.Errorf("Expected Next() to return false")
	}
	if req.ID != 0 || req.Address != 0 {
		t.Errorf("Expected zero-value request")
	}
}

// 2. Single Request
func TestGenerator_SingleRequest(t *testing.T) {
	trace := []model.Request{
		{ID: 1, Address: 0x100, Op: model.Read, Size: 4},
	}
	g := NewGenerator(trace)

	if !g.HasNext() {
		t.Errorf("Expected HasNext() to be true")
	}

	req, ok := g.Next()
	if !ok {
		t.Errorf("Expected Next() to return true")
	}
	if req != trace[0] {
		t.Errorf("Expected request %v, got %v", trace[0], req)
	}

	if g.HasNext() {
		t.Errorf("Expected HasNext() to be false")
	}

	_, ok = g.Next()
	if ok {
		t.Errorf("Expected Next() to return false")
	}
}

// 3. Multiple Requests
func TestGenerator_MultipleRequests(t *testing.T) {
	trace := []model.Request{
		{ID: 1, Address: 0x100, Op: model.Read, Size: 4},
		{ID: 2, Address: 0x200, Op: model.Write, Size: 8},
		{ID: 3, Address: 0x300, Op: model.Read, Size: 4},
	}
	g := NewGenerator(trace)

	for i, expected := range trace {
		if !g.HasNext() {
			t.Errorf("Expected HasNext() to be true at step %d", i)
		}
		req, ok := g.Next()
		if !ok {
			t.Errorf("Expected Next() to return true at step %d", i)
		}
		if req != expected {
			t.Errorf("Expected request %v, got %v", expected, req)
		}
	}

	if g.HasNext() {
		t.Errorf("Expected HasNext() to be false")
	}
}

// 4. Extra Calls
func TestGenerator_ExtraCallsAfterCompletion(t *testing.T) {
	trace := []model.Request{
		{ID: 1, Address: 0x100, Op: model.Read, Size: 4},
	}
	g := NewGenerator(trace)

	g.Next() // Consume

	for i := 0; i < 5; i++ {
		req, ok := g.Next()
		if ok {
			t.Errorf("Expected false on extra call %d", i)
		}
		if req.ID != 0 {
			t.Errorf("Expected zero-value on extra call %d", i)
		}
	}
}

// 5. Independence
func TestGenerator_IndependenceFromInputSlice(t *testing.T) {
	trace := []model.Request{
		{ID: 1, Address: 0x100, Op: model.Read, Size: 4},
	}
	g := NewGenerator(trace)

	// Mutate external slice.
	trace[0].Address = 0x999
	trace[0].Op = model.Write

	req, ok := g.Next()
	if !ok {
		t.Fatalf("Expected true")
	}

	// Verify internal copy.
	if req.Address == 0x999 || req.Op == model.Write {
		t.Errorf("Internal copy failed")
	}
	if req.Address != 0x100 || req.Op != model.Read {
		t.Errorf("Expected original values")
	}
}

// 6. Optional Features
func TestGenerator_OptionalFeatures(t *testing.T) {
	trace := []model.Request{
		{ID: 1, Address: 0x100, Op: model.Read, Size: 4},
		{ID: 2, Address: 0x200, Op: model.Write, Size: 8},
	}
	g := NewGenerator(trace)

	if g.Total() != 2 {
		t.Errorf("Expected Total() 2, got %d", g.Total())
	}

	if g.Remaining() != 2 {
		t.Errorf("Expected Remaining() 2, got %d", g.Remaining())
	}

	g.Next() // Consume one.

	if g.Remaining() != 1 {
		t.Errorf("Expected Remaining() 1, got %d", g.Remaining())
	}

	g.Reset() // Reset state.

	if g.Remaining() != 2 {
		t.Errorf("Expected Remaining() 2 after Reset(), got %d", g.Remaining())
	}

	req, _ := g.Next()
	if req.ID != 1 {
		t.Errorf("Expected first request, got ID %d", req.ID)
	}
}
