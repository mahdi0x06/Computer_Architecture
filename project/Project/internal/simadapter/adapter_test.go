package simadapter

import (
	"reflect"
	"testing"

	"victimcacheproject/internal/benchmark"
	"victimcacheproject/internal/config"
	"victimcacheproject/internal/model"
	"victimcacheproject/internal/system"
)

func adapterSuiteConfig(cfg config.Config) benchmark.SuiteConfig {
	return benchmark.SuiteConfig{
		BaseAddress:     0,
		BlockSizeBytes:  cfg.BlockSizeBytes,
		L1SizeBytes:     cfg.L1SizeBytes,
		L2SizeBytes:     cfg.L2SizeBytes,
		L2Associativity: cfg.L2Associativity,
		VictimEntries:   cfg.VictimEntries,
		NumberOfBlocks:  4,
		Repetitions:     8,
		SequentialWords: 32,
		WordSizeBytes:   4,
		AccessSizeBytes: 8,
	}
}

func adapterConfigurations() []struct {
	name string
	cfg  config.Config
} {
	base := config.Default()

	memoryOnly := base
	memoryOnly.Topology = config.TopologyMemoryOnly

	l1 := base
	l1.Topology = config.TopologyL1

	l1L2 := base
	l1L2.Topology = config.TopologyL1L2

	fullFIFO := base
	fullFIFO.Topology = config.TopologyFull
	fullFIFO.VictimPolicy = config.ReplacementFIFO

	fullLRU := base
	fullLRU.Topology = config.TopologyFull
	fullLRU.VictimPolicy = config.ReplacementLRU

	return []struct {
		name string
		cfg  config.Config
	}{
		{name: "memory", cfg: memoryOnly},
		{name: "l1", cfg: l1},
		{name: "l1-l2", cfg: l1L2},
		{name: "full-fifo", cfg: fullFIFO},
		{name: "full-lru", cfg: fullLRU},
	}
}

func TestAkitaAdapterMatchesFunctionalReferenceForCompleteSuite(t *testing.T) {
	base := config.Default()
	scenarios, err := benchmark.GenerateSuite(adapterSuiteConfig(base))
	if err != nil {
		t.Fatal(err)
	}

	for _, scenario := range scenarios {
		for _, configuration := range adapterConfigurations() {
			name := string(scenario.Kind) + "/" + configuration.name
			t.Run(name, func(t *testing.T) {
				reference := system.New(configuration.cfg)
				wantResponses := reference.Run(scenario.Requests)
				wantStats := reference.Stats

				akitaSystem := system.New(configuration.cfg)
				adapter := New(akitaSystem)
				adapter.SetRequests(scenario.Requests)
				if err := adapter.Build(); err != nil {
					t.Fatal(err)
				}
				if err := adapter.Run(); err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(adapter.Responses, wantResponses) {
					t.Fatalf("Akita responses differ from the functional reference")
				}
				if akitaSystem.Stats != wantStats {
					t.Fatalf("Akita stats=%+v, functional stats=%+v", akitaSystem.Stats, wantStats)
				}
				if adapter.engine == nil {
					t.Fatal("Akita engine was not retained after the run")
				}
				if got := simulationFrequency.Cycle(adapter.engine.CurrentTime()); got < wantStats.TotalCycles {
					t.Fatalf("Akita engine ended at cycle %d before %d service cycles completed", got, wantStats.TotalCycles)
				}
			})
		}
	}
}

func TestAkitaAdapterHandlesEmptyTrace(t *testing.T) {
	simulator := system.New(config.Default())
	adapter := New(simulator)
	adapter.SetRequests(nil)

	if err := adapter.Build(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Run(); err != nil {
		t.Fatal(err)
	}
	if len(adapter.Responses) != 0 {
		t.Fatalf("responses=%d, want 0", len(adapter.Responses))
	}
	if got := simulationFrequency.Cycle(adapter.engine.CurrentTime()); got != 0 {
		t.Fatalf("engine cycles=%d, want 0", got)
	}
}

func TestAkitaAdapterCopiesRequestInput(t *testing.T) {
	requests := []model.Request{{
		ID: 1, Address: 64, Op: model.Read, Size: 8,
	}}
	adapter := New(system.New(config.Default()))
	adapter.SetRequests(requests)
	requests[0].Address = 4096

	if adapter.Requests[0].Address != 64 {
		t.Fatalf("adapter request address=%d, want 64", adapter.Requests[0].Address)
	}
}

func TestAkitaAdapterRequiresBuild(t *testing.T) {
	adapter := New(system.New(config.Default()))
	if err := adapter.Run(); err == nil {
		t.Fatal("Run must reject an adapter that was not built")
	}
}

func TestAkitaAdapterRejectsNilSystem(t *testing.T) {
	adapter := New(nil)
	if err := adapter.Build(); err == nil {
		t.Fatal("Build must reject a nil system")
	}
}

func TestAkitaMessagesCloneWithNewIdentity(t *testing.T) {
	request := newAccessRequestMsg(
		model.Request{ID: 7, Address: 128, Op: model.Read, Size: 8},
		"Source.Port", "Destination.Port",
	)
	requestClone := request.Clone().(*accessRequestMsg)
	if requestClone.ID == request.ID {
		t.Fatal("cloned request must have a new Akita message ID")
	}
	if requestClone.Request != request.Request {
		t.Fatal("cloned request changed the functional payload")
	}

	response := newAccessResponseMsg(
		request,
		model.Response{RequestID: 7, Location: model.HitL1, LatencyCycles: 1},
		"Destination.Port",
	)
	responseClone := response.Clone().(*accessResponseMsg)
	if responseClone.ID == response.ID {
		t.Fatal("cloned response must have a new Akita message ID")
	}
	if responseClone.GetRspTo() != response.GetRspTo() || responseClone.Response != response.Response {
		t.Fatal("cloned response changed correlation or functional payload")
	}
}
