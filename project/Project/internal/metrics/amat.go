package metrics

// AMATInputs follows the project report formula.
type AMATInputs struct {
	L1HitTime      float64
	L1MissRate     float64
	VictimHitRate  float64
	VictimPenalty  float64
	VictimMissRate float64
	L2Penalty      float64
}

func CalculateAMAT(in AMATInputs) float64 {
	return in.L1HitTime + in.L1MissRate*(in.VictimHitRate*in.VictimPenalty+in.VictimMissRate*in.L2Penalty)
}

func CalculateBaselineAMAT(l1HitTime, l1MissRate, l2Penalty float64) float64 {
	return l1HitTime + l1MissRate*l2Penalty
}
