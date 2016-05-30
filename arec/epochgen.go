package arec

import "github.com/relab/goxos/grp"

func getNewEpoch(id grp.ID) grp.Epoch {
	return grp.Epoch(uint64(id.PaxosID) + (uint64(id.Epoch)/10+1)*10)
}

func cantorPairing(k1, k2 uint64) uint64 {
	return (k1+k2)*(k1+k2+1)/2 + k2
}
