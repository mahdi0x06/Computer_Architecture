package benchmark

import "victimcacheproject/internal/model"

type ConflictConfig struct {
	BaseAddress     uint64
	CacheSizeBytes  uint64
	NumberOfBlocks  int
	Repetitions     int
	AccessSizeBytes uint64
}

// GenerateConflictTrace will create addresses separated by the L1 capacity,
// making them map to the same direct-mapped L1 index.
func GenerateConflictTrace(cfg ConflictConfig) []model.Request {
	if cfg.CacheSizeBytes == 0 || cfg.NumberOfBlocks <= 0 || cfg.Repetitions <= 0 {
		return []model.Request{}
	}

	requests := make([]model.Request, 0, cfg.NumberOfBlocks*cfg.Repetitions)
	for repetition := 0; repetition < cfg.Repetitions; repetition++ {
		for block := 0; block < cfg.NumberOfBlocks; block++ {
			requests = append(requests, model.Request{
				ID:      uint64(len(requests) + 1),
				Address: cfg.BaseAddress + uint64(block)*cfg.CacheSizeBytes,
				Op:      model.Read,
				Size:    cfg.AccessSizeBytes,
			})
		}
	}

	return requests
}
