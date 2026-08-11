package simadapter

import (
	"github.com/sarchlab/akita/v4/sim"

	"victimcacheproject/internal/model"
)

const (
	accessRequestTrafficClass  = "VictimCacheAccessRequest"
	accessResponseTrafficClass = "VictimCacheAccessResponse"
)

// accessRequestMsg carries one simulator-independent request between the
// Akita request driver and the hierarchy executor.
type accessRequestMsg struct {
	sim.MsgMeta
	Request model.Request
}

func newAccessRequestMsg(
	req model.Request,
	src, dst sim.RemotePort,
) *accessRequestMsg {
	return &accessRequestMsg{
		MsgMeta: sim.MsgMeta{
			ID:           sim.GetIDGenerator().Generate(),
			Src:          src,
			Dst:          dst,
			TrafficClass: accessRequestTrafficClass,
			TrafficBytes: uint64ToInt(req.Size),
		},
		Request: req,
	}
}

func (m *accessRequestMsg) Meta() *sim.MsgMeta { return &m.MsgMeta }

func (m *accessRequestMsg) Clone() sim.Msg {
	clone := *m
	clone.ID = sim.GetIDGenerator().Generate()
	return &clone
}

// accessResponseMsg returns the unchanged functional response through Akita.
type accessResponseMsg struct {
	sim.MsgMeta
	RspTo    string
	Response model.Response
}

func newAccessResponseMsg(
	req *accessRequestMsg,
	rsp model.Response,
	src sim.RemotePort,
) *accessResponseMsg {
	return &accessResponseMsg{
		MsgMeta: sim.MsgMeta{
			ID:           sim.GetIDGenerator().Generate(),
			Src:          src,
			Dst:          req.Src,
			TrafficClass: accessResponseTrafficClass,
			TrafficBytes: req.TrafficBytes,
		},
		RspTo:    req.ID,
		Response: rsp,
	}
}

func (m *accessResponseMsg) Meta() *sim.MsgMeta { return &m.MsgMeta }

func (m *accessResponseMsg) Clone() sim.Msg {
	clone := *m
	clone.ID = sim.GetIDGenerator().Generate()
	return &clone
}

func (m *accessResponseMsg) GetRspTo() string { return m.RspTo }

func uint64ToInt(value uint64) int {
	maxInt := uint64(^uint(0) >> 1)
	if value > maxInt {
		return int(maxInt)
	}
	return int(value)
}

var (
	_ sim.Msg = (*accessRequestMsg)(nil)
	_ sim.Rsp = (*accessResponseMsg)(nil)
)
