package model

type HitLocation string

const (
	HitL1     HitLocation = "L1"
	HitVictim HitLocation = "VictimCache"
	HitL2     HitLocation = "L2"
	HitMemory HitLocation = "MainMemory"
)

type Response struct {
	RequestID     uint64
	Location      HitLocation
	LatencyCycles uint64
}
