package model

type Operation uint8

const (
	Read Operation = iota
	Write
)

// Request is simulator-independent on purpose.
type Request struct {
	ID      uint64
	Address uint64
	Op      Operation
	Size    uint64
}
