package system

import (
	"fmt"
	"victimcacheproject/internal/cache"
	"victimcacheproject/internal/config"
	"victimcacheproject/internal/memory"
	"victimcacheproject/internal/metrics"
	"victimcacheproject/internal/model"
)

type System struct {
	Config config.Config
	L1     *cache.L1
	Victim *cache.Victim
	L2     *cache.L2
	Memory *memory.MainMemory
	Stats  metrics.Stats
}

func New(cfg config.Config) *System {
	s := &System{Config: cfg, Memory: memory.NewMainMemory()}
	if cfg.UsesL1() {
		s.L1 = cache.NewL1(cfg)
	}
	if cfg.UsesL2() {
		s.L2 = cache.NewL2(cfg)
	}
	if cfg.NormalizedTopology() == config.TopologyFull {
		s.Victim = cache.NewVictim(cfg)
	}
	return s
}
func (s *System) Validate() error { return s.Config.ValidateMemoryHierarchy() }
func (s *System) ResetStats()     { s.Stats = metrics.Stats{} }

// Run is the synchronous correctness oracle. User-facing commands execute the
// same requests through simadapter's Akita components and compare against this
// behavior in integration tests.
func (s *System) Run(reqs []model.Request) []model.Response {
	out := make([]model.Response, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, s.Access(r))
	}
	return out
}

func (s *System) Access(req model.Request) model.Response {
	s.Stats.TotalRequests++
	if !s.Config.UsesL1() {
		return s.fromMemory(req, 0)
	}
	latency := s.Config.L1HitLatencyCycles
	if hit := s.L1.Lookup(req); hit.Hit {
		s.Stats.L1Hits++
		b := *hit.Block
		if req.Op == model.Write {
			b.Dirty = true
			s.L1.Insert(b)
		}
		s.Stats.TotalCycles += latency
		return model.Response{RequestID: req.ID, Location: model.HitL1, LatencyCycles: latency}
	}
	s.Stats.L1Misses++
	blockAddr := req.Address / s.Config.BlockSizeBytes
	if s.Config.UsesVictim() {
		latency += s.Config.VictimLatencyCycles
		if hit := s.Victim.Lookup(req); hit.Hit {
			s.Stats.VictimHits++
			requested, _ := s.Victim.Remove(blockAddr)
			ev := s.L1.Insert(requested)
			if ev != nil && ev.Valid {
				if overflow := s.Victim.Insert(*ev); overflow != nil {
					s.forwardEviction(*overflow)
				}
			}
			if req.Op == model.Write {
				requested.Dirty = true
				s.L1.Insert(requested)
			}
			s.Stats.VictimSwaps++
			s.Stats.TotalCycles += latency
			return model.Response{RequestID: req.ID, Location: model.HitVictim, LatencyCycles: latency}
		}
		s.Stats.VictimMisses++
	}
	if s.Config.UsesL2() {
		latency += s.Config.L2LatencyCycles
		s.Stats.L2ReadRequests++
		if hit := s.L2.Lookup(req); hit.Hit {
			s.Stats.L2Hits++
			b := *hit.Block
			if req.Op == model.Write {
				b.Dirty = true
				s.L2.Insert(b)
			}
			s.installL1(b)
			s.Stats.TotalCycles += latency
			return model.Response{RequestID: req.ID, Location: model.HitL2, LatencyCycles: latency}
		}
		s.Stats.L2Misses++
	}
	return s.fromMemory(req, latency)
}

func (s *System) fromMemory(req model.Request, latency uint64) model.Response {
	blockAddr := req.Address / s.Config.BlockSizeBytes
	latency += s.Config.MemoryLatencyCycles
	s.Stats.MemoryAccesses++
	b := s.Memory.ReadBlock(blockAddr)
	if req.Op == model.Write {
		b.Dirty = true
	}
	if s.Config.UsesL2() {
		if ev := s.L2.Insert(b); ev != nil && ev.Valid && ev.Dirty {
			s.Memory.WriteBlock(*ev)
			s.Stats.MemoryAccesses++
			s.Stats.L2WriteRequests++
		}
	}
	if s.Config.UsesL1() {
		s.installL1(b)
	} else if req.Op == model.Write {
		s.Memory.WriteBlock(b)
	}
	s.Stats.TotalCycles += latency
	return model.Response{RequestID: req.ID, Location: model.HitMemory, LatencyCycles: latency}
}

func (s *System) installL1(b model.Block) {
	if s.L1 == nil {
		return
	}
	ev := s.L1.Insert(b)
	if ev == nil || !ev.Valid {
		return
	}
	if s.Config.UsesVictim() {
		if overflow := s.Victim.Insert(*ev); overflow != nil {
			s.forwardEviction(*overflow)
		}
	} else {
		s.forwardEviction(*ev)
	}
}
func (s *System) forwardEviction(b model.Block) {
	if s.Config.UsesL2() {
		if ev := s.L2.Insert(b); ev != nil && ev.Valid && ev.Dirty {
			s.Memory.WriteBlock(*ev)
			s.Stats.MemoryAccesses++
			s.Stats.L2WriteRequests++
		}
		return
	}
	if b.Dirty {
		s.Memory.WriteBlock(b)
		s.Stats.MemoryAccesses++
	}
}

func (s *System) Summary() string {
	return fmt.Sprintf("topology=%s requests=%d cycles=%d avg=%.2f L1=%d/%d VC=%d/%d L2=%d/%d memory=%d", s.Config.NormalizedTopology(), s.Stats.TotalRequests, s.Stats.TotalCycles, s.Stats.AverageCyclesPerRequest(), s.Stats.L1Hits, s.Stats.L1Misses, s.Stats.VictimHits, s.Stats.VictimMisses, s.Stats.L2Hits, s.Stats.L2Misses, s.Stats.MemoryAccesses)
}
