// Package simadapter runs the functional memory-hierarchy model through an
// Akita event engine. The System remains the correctness authority for cache
// behavior, while Akita provides components, ports, messages, connections, and
// scheduled completion events.
package simadapter

import (
	"fmt"

	"github.com/sarchlab/akita/v4/sim"
	"github.com/sarchlab/akita/v4/sim/directconnection"

	"victimcacheproject/internal/model"
	"victimcacheproject/internal/system"
)

const simulationFrequency = 1 * sim.GHz

type Adapter struct {
	System    *system.System
	Requests  []model.Request
	Responses []model.Response

	engine sim.Engine
	built  bool
}

func New(sys *system.System) *Adapter { return &Adapter{System: sys} }
func (a *Adapter) SetRequests(reqs []model.Request) {
	a.Requests = append([]model.Request(nil), reqs...)
}
func (a *Adapter) Build() error {
	if a.System == nil {
		return fmt.Errorf("system is nil")
	}
	if err := a.System.Validate(); err != nil {
		return err
	}
	a.built = true
	return nil
}
func (a *Adapter) Run() error {
	if !a.built {
		return fmt.Errorf("adapter must be built before run")
	}

	engine := sim.NewSerialEngine()
	driver := newRequestDriver(engine, a.Requests)
	hierarchy := newHierarchyExecutor(engine, simulationFrequency, a.System)
	connection := directconnection.MakeBuilder().
		WithEngine(engine).
		WithFreq(simulationFrequency).
		Build("MemoryHierarchyConnection")

	connection.PlugIn(driver.Port)
	connection.PlugIn(hierarchy.Port)
	driver.Destination = hierarchy.Port.AsRemote()

	driver.Start()
	if err := engine.Run(); err != nil {
		return err
	}
	if driver.err != nil {
		return driver.err
	}
	if hierarchy.err != nil {
		return hierarchy.err
	}
	if len(driver.Responses) != len(a.Requests) {
		return fmt.Errorf(
			"Akita simulation completed %d responses for %d requests",
			len(driver.Responses), len(a.Requests),
		)
	}

	a.engine = engine
	a.Responses = append([]model.Response(nil), driver.Responses...)
	return nil
}
