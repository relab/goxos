package ringreplacer

import "github.com/relab/goxos/grp"

func epochGenerator(id grp.ID) func(grp.Epoch) grp.Epoch {
	var n uint64
	baseForNode := cantorPairing(uint64(id.PaxosID), uint64(id.Epoch))
	return func(oldEpoch grp.Epoch) grp.Epoch {
		var replacerEpoch uint64
		for replacerEpoch <= uint64(oldEpoch) {
			replacerEpoch = cantorPairing(baseForNode, n)
			n++
		}
		return grp.Epoch(replacerEpoch)
	}
}

func cantorPairing(k1, k2 uint64) uint64 {
	return (k1+k2)*(k1+k2+1)/2 + k2
}
