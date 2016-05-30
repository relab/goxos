package batchpaxos

// The batchpointer structure is used to maintaining batchpoints at the leader. Previous
// batchpoints are stored so that they can be used to compute future batchpoints.
type BatchPointer struct {
	BatchPoints []BatchPoint
}

// Construct a new batchpointer, with empty list of previous batchpoints.
func NewBatchPointer() *BatchPointer {
	return &BatchPointer{
		BatchPoints: make([]BatchPoint, 0),
	}
}

// Given a batchpoint, make a range map from it. In this case the ranges all start at
// 0 since there are no previous batchpoints.
func makeRangeMap1(bp BatchPoint) RangeMap {
	rm := make(RangeMap)
	for cid, hs := range bp {
		rm[cid] = ClientRange{Start: 0, Stop: hs}
	}
	return rm
}

// Given two batchpoints, make a range from them such that all requests between each
// of the ranges are included.
func makeRangeMap2(bp1, bp2 BatchPoint) RangeMap {
	rm := make(RangeMap)
	for cid, hs := range bp2 {
		phs, exist := bp1[cid]
		if !exist {
			rm[cid] = ClientRange{Start: 0, Stop: hs}
		} else if hs > phs {
			rm[cid] = ClientRange{Start: phs + 1, Stop: hs}
		}
	}
	return rm
}

// Given an intersection from the batch acceptor, check if it is possible to do a batchpoint
// operation. If so, return true and the ranges for the batch. If not, return false and nil.
func (bp *BatchPointer) TryBatchPoint(nbp BatchPoint) (bool, RangeMap) {
	if len(bp.BatchPoints) == 0 {
		if len(nbp) > 0 {
			bp.BatchPoints = append(bp.BatchPoints, nbp)
			return true, makeRangeMap1(nbp)
		}
		return false, nil
	}
	last := bp.BatchPoints[len(bp.BatchPoints)-1]
	grtr := make(BatchPoint)
	rmable := false
	for cid, hs := range nbp {
		phs, exist := last[cid]
		if !exist {
			grtr[cid] = hs
			rmable = true
		} else if hs > phs {
			grtr[cid] = hs
			rmable = true
		} else {
			grtr[cid] = phs
		}
	}
	if rmable {
		bp.BatchPoints = append(bp.BatchPoints, grtr)
		return true, makeRangeMap2(last, grtr)
	}
	return false, nil
}
