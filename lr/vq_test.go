package lr

import (
	"testing"

	"github.com/relab/goxos/grp"
)

var (
	id00 = grp.NewIDFromInt(0, 0)
	id10 = grp.NewIDFromInt(1, 0)
	id20 = grp.NewIDFromInt(2, 0)
	id30 = grp.NewIDFromInt(3, 0)
	id40 = grp.NewIDFromInt(4, 0)

	id02 = grp.NewIDFromInt(0, 2)
	id23 = grp.NewIDFromInt(2, 3)
)

var (
	qs2ev1 = []grp.Epoch{0, 0, 0}
	qs2ev2 = []grp.Epoch{2, 0, 0}

	qs2msg1 = &EpochPromise{ID: id00, EpochVector: qs2ev1}
	qs2msg2 = &EpochPromise{ID: id10, EpochVector: qs2ev1}
	qs2msg3 = &EpochPromise{ID: id20, EpochVector: qs2ev1}

	qs2msg4 = &EpochPromise{ID: id02, EpochVector: qs2ev2}
	qs2msg5 = &EpochPromise{ID: id10, EpochVector: qs2ev2}

	qs2msgs1 = []*EpochPromise{qs2msg1}
	qs2msgs2 = []*EpochPromise{qs2msg1, qs2msg2}
	qs2msgs3 = []*EpochPromise{qs2msg1, qs2msg2, qs2msg3}
	qs2msgs4 = []*EpochPromise{qs2msg1, qs2msg4}
	qs2msgs5 = []*EpochPromise{qs2msg3, qs2msg5}
	qs2msgs6 = []*EpochPromise{qs2msg4, qs2msg3, qs2msg5}
	qs2msgs7 = []*EpochPromise{qs2msg4, qs2msg1, qs2msg2}
)

var (
	qs3ev1 = []grp.Epoch{0, 0, 0, 0, 0}
	qs3ev2 = []grp.Epoch{0, 0, 3, 0, 0}
	qs3ev3 = []grp.Epoch{1, 0, 3, 0, 0}

	qs3msg1  = &EpochPromise{ID: id00, EpochVector: qs3ev1}
	qs3msg2  = &EpochPromise{ID: id10, EpochVector: qs3ev1}
	qs3msg3  = &EpochPromise{ID: id20, EpochVector: qs3ev1}
	qs3msg4  = &EpochPromise{ID: id30, EpochVector: qs3ev1}
	qs3msg5  = &EpochPromise{ID: id40, EpochVector: qs3ev1}
	qs3msg7  = &EpochPromise{ID: id23, EpochVector: qs3ev2}
	qs3msg8  = &EpochPromise{ID: id23, EpochVector: qs3ev2}
	qs3msg9  = &EpochPromise{ID: id30, EpochVector: qs3ev2}
	qs3msg10 = &EpochPromise{ID: id23, EpochVector: qs3ev3}

	qs3msgs1  = []*EpochPromise{qs3msg1}
	qs3msgs2  = []*EpochPromise{qs3msg1, qs3msg1, qs3msg1}
	qs3msgs3  = []*EpochPromise{qs3msg1, qs3msg1, qs3msg2}
	qs3msgs4  = []*EpochPromise{qs3msg1, qs3msg2, qs3msg3}
	qs3msgs5  = []*EpochPromise{qs3msg1, qs3msg2, qs3msg3, qs3msg4}
	qs3msgs6  = []*EpochPromise{qs3msg1, qs3msg2, qs3msg3, qs3msg4, qs3msg5}
	qs3msgs7  = []*EpochPromise{qs3msg3, qs3msg4, qs3msg5}
	qs3msgs8  = []*EpochPromise{qs3msg1, qs3msg2, qs3msg7}
	qs3msgs9  = []*EpochPromise{qs3msg1, qs3msg2, qs3msg10}
	qs3msgs10 = []*EpochPromise{qs3msg1, qs3msg2, qs3msg10, qs3msg9}
)

var extractValidQuorumSize2EpTests = []struct {
	msgs            []*EpochPromise
	quorumSize      uint
	expectedFound   bool
	expectedIndices []int
}{
	{qs2msgs1, 2, false, []int{}},
	{qs2msgs2, 2, true, []int{0, 1}},
	{qs2msgs3, 2, true, []int{0, 1}},
	{qs2msgs4, 2, false, []int{}},
	{qs2msgs5, 2, true, []int{0, 1}},
	{qs2msgs6, 2, true, []int{0, 1}},
	{qs2msgs7, 2, true, []int{0, 2}},
}

func TestExtractValidQuorumSize2Ep(t *testing.T) {
	var actualFound bool
	var actualIndices []int

	for i, evaqt := range extractValidQuorumSize2EpTests {
		actualFound, actualIndices = extractValidQuorumEp(evaqt.msgs, evaqt.quorumSize)
		if actualFound != evaqt.expectedFound {
			t.Errorf("%d. Got '%t', want '%t'", i, actualFound, evaqt.expectedFound)
		}
		if !equal(actualIndices, evaqt.expectedIndices) {
			t.Errorf("%d. Got '%v', want '%v'", i, actualIndices, evaqt.expectedIndices)
		}
	}
}

var extractValidQuorumSize3EpTests = []struct {
	msgs            []*EpochPromise
	quorumSize      uint
	expectedFound   bool
	expectedIndices []int
}{
	{qs3msgs1, 3, false, []int{}},
	{qs3msgs2, 3, false, []int{}},
	{qs3msgs3, 3, false, []int{}},
	{qs3msgs4, 3, true, []int{0, 1, 2}},
	{qs3msgs5, 3, true, []int{0, 1, 2}},
	{qs3msgs6, 3, true, []int{0, 1, 2}},
	{qs3msgs7, 3, true, []int{0, 1, 2}},
	{qs3msgs8, 3, true, []int{0, 1, 2}},
	{qs3msgs9, 3, false, []int{}},
	{qs3msgs10, 3, true, []int{0, 1, 3}},
}

func TestExtractValidQuorumEpSize3(t *testing.T) {
	var actualFound bool
	var actualIndices []int

	for i, evaqt := range extractValidQuorumSize3EpTests {
		actualFound, actualIndices = extractValidQuorumEp(evaqt.msgs, evaqt.quorumSize)
		if actualFound != evaqt.expectedFound {
			t.Errorf("%d. Got '%t', want '%t'", i, actualFound, evaqt.expectedFound)
		}
		if !equal(actualIndices, evaqt.expectedIndices) {
			t.Errorf("%d. Got '%v', want '%v'", i, actualIndices, evaqt.expectedIndices)
		}
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, c := range a {
		if c != b[i] {
			return false
		}
	}
	return true
}
