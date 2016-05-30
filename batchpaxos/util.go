package batchpaxos

import (
	"sort"
)

// Given a range map, sort the keys (client ids) of the map and return
// them.
func SortedKeysRange(m RangeMap) []string {
	// Sort for debugging
	var ids []string
	for cid := range m {
		ids = append(ids, cid)
	}
	sort.Strings(ids)
	return ids
}

func SortedKeysHSS(m BatchPoint) []string {
	var ids []string
	for cid := range m {
		ids = append(ids, cid)
	}
	sort.Strings(ids)
	return ids
}
