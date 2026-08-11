package simadapter

import (
	"fmt"

	"github.com/sarchlab/akita/v4/sim"

	"victimcacheproject/internal/model"
	"victimcacheproject/internal/system"
)

const componentPortName = "Port"

type startDriverEvent struct {
	*sim.EventBase
}

type completeAccessEvent struct {
	*sim.EventBase
	response *accessResponseMsg
}

// requestDriver is an Akita component that injects one request and waits for
// its response before injecting the next. Serial issue preserves the exact
// ordering and cache state transitions of the established reference runner.
type requestDriver struct {
	*sim.ComponentBase

	Engine      sim.Engine
	Port        sim.Port
	Destination sim.RemotePort
	Requests    []model.Request
	Responses   []model.Response

	nextRequest int
	outstanding *accessRequestMsg
	err         error
}

func newRequestDriver(
	engine sim.Engine,
	requests []model.Request,
) *requestDriver {
	driver := &requestDriver{
		Engine:   engine,
		Requests: append([]model.Request(nil), requests...),
	}
	driver.ComponentBase = sim.NewComponentBase("MemoryRequestDriver")
	driver.Port = sim.NewPort(driver, 1, 1, driver.Name()+"."+componentPortName)
	driver.AddPort(componentPortName, driver.Port)
	return driver
}

func (d *requestDriver) Start() {
	d.Engine.Schedule(&startDriverEvent{
		EventBase: sim.NewEventBase(0, d),
	})
}

func (d *requestDriver) Handle(event sim.Event) error {
	if _, ok := event.(*startDriverEvent); !ok {
		d.fail(fmt.Errorf("request driver cannot handle event %T", event))
		return nil
	}
	d.sendNext()
	return nil
}

func (d *requestDriver) NotifyRecv(port sim.Port) {
	if d.err != nil {
		return
	}

	msg := port.RetrieveIncoming()
	response, ok := msg.(*accessResponseMsg)
	if !ok {
		d.fail(fmt.Errorf("request driver received message %T", msg))
		return
	}
	if d.outstanding == nil {
		d.fail(fmt.Errorf("request driver received response without an outstanding request"))
		return
	}
	if response.GetRspTo() != d.outstanding.ID {
		d.fail(fmt.Errorf(
			"response targets request %q, want %q",
			response.GetRspTo(), d.outstanding.ID,
		))
		return
	}
	if response.Response.RequestID != d.outstanding.Request.ID {
		d.fail(fmt.Errorf(
			"response request ID %d, want %d",
			response.Response.RequestID, d.outstanding.Request.ID,
		))
		return
	}

	d.Responses = append(d.Responses, response.Response)
	d.outstanding = nil
	d.sendNext()
}

func (d *requestDriver) NotifyPortFree(_ sim.Port) {}

func (d *requestDriver) sendNext() {
	if d.err != nil || d.outstanding != nil || d.nextRequest >= len(d.Requests) {
		return
	}
	if d.Destination == "" {
		d.fail(fmt.Errorf("request driver destination is not configured"))
		return
	}

	msg := newAccessRequestMsg(
		d.Requests[d.nextRequest],
		d.Port.AsRemote(),
		d.Destination,
	)
	if sendErr := d.Port.Send(msg); sendErr != nil {
		d.fail(fmt.Errorf("request driver output port is full"))
		return
	}

	d.outstanding = msg
	d.nextRequest++
}

func (d *requestDriver) fail(err error) {
	if d.err == nil {
		d.err = err
	}
}

// hierarchyExecutor is the Akita-facing component for the unchanged
// functional System. It executes one logical access and schedules delivery of
// that response after the exact number of cycles calculated by the System.
type hierarchyExecutor struct {
	*sim.ComponentBase

	Engine    sim.Engine
	Frequency sim.Freq
	Port      sim.Port
	System    *system.System

	busy bool
	err  error
}

func newHierarchyExecutor(
	engine sim.Engine,
	frequency sim.Freq,
	sys *system.System,
) *hierarchyExecutor {
	executor := &hierarchyExecutor{
		Engine:    engine,
		Frequency: frequency,
		System:    sys,
	}
	executor.ComponentBase = sim.NewComponentBase("MemoryHierarchyExecutor")
	executor.Port = sim.NewPort(executor, 1, 1, executor.Name()+"."+componentPortName)
	executor.AddPort(componentPortName, executor.Port)
	return executor
}

func (e *hierarchyExecutor) Handle(event sim.Event) error {
	completion, ok := event.(*completeAccessEvent)
	if !ok {
		e.fail(fmt.Errorf("hierarchy executor cannot handle event %T", event))
		return nil
	}
	if !e.busy {
		e.fail(fmt.Errorf("hierarchy executor completed an access while idle"))
		return nil
	}
	if sendErr := e.Port.Send(completion.response); sendErr != nil {
		e.fail(fmt.Errorf("hierarchy executor output port is full"))
		return nil
	}

	e.busy = false
	return nil
}

func (e *hierarchyExecutor) NotifyRecv(port sim.Port) {
	if e.err != nil {
		return
	}
	if e.busy {
		e.fail(fmt.Errorf("hierarchy executor received overlapping requests"))
		return
	}

	msg := port.RetrieveIncoming()
	request, ok := msg.(*accessRequestMsg)
	if !ok {
		e.fail(fmt.Errorf("hierarchy executor received message %T", msg))
		return
	}

	response := e.System.Access(request.Request)
	cycles := response.LatencyCycles
	maxInt := uint64(^uint(0) >> 1)
	if cycles > maxInt {
		e.fail(fmt.Errorf("response latency %d exceeds the Akita cycle range", cycles))
		return
	}

	e.busy = true
	completionTime := e.Frequency.NCyclesLater(
		int(cycles), e.Engine.CurrentTime(),
	)
	e.Engine.Schedule(&completeAccessEvent{
		EventBase: sim.NewEventBase(completionTime, e),
		response: newAccessResponseMsg(
			request, response, e.Port.AsRemote(),
		),
	})
}

func (e *hierarchyExecutor) NotifyPortFree(_ sim.Port) {}

func (e *hierarchyExecutor) fail(err error) {
	if e.err == nil {
		e.err = err
	}
}

var (
	_ sim.Component = (*requestDriver)(nil)
	_ sim.Component = (*hierarchyExecutor)(nil)
)
