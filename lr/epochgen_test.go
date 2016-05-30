package lr

import (
	"github.com/relab/goxos/grp"
	"testing"
)

var cantorTests = []struct {
	k1, k2   uint64
	expected uint64
}{
	{0, 0, 0},
	{1, 0, 1},
	{2, 0, 3},
	{0, 1, 2},
	{1, 42, 988},
	{77, 103, 16393},
}

func TestCantorPairing(t *testing.T) {
	var actual uint64
	for i, ct := range cantorTests {
		actual = cantorPairing(ct.k1, ct.k2)
		if actual != ct.expected {
			t.Errorf("%d. %d != %d", i, actual, ct.expected)
		}
	}
}

func TestEpochGenerator(t *testing.T) {
	// Just a small sanity check for duplicates
	for i := 0; i < 100; i++ {
		id0 := grp.NewIDFromInt(0, uint64(i))
		id1 := grp.NewIDFromInt(1, uint64(i))
		id2 := grp.NewIDFromInt(2, uint64(i))

		epochGen0 := epochGenerator(id0)
		epochGen1 := epochGenerator(id1)
		epochGen2 := epochGenerator(id2)

		e0 := make([]grp.Epoch, 500)
		e1 := make([]grp.Epoch, 500)
		e2 := make([]grp.Epoch, 500)

		for j := 0; j < 500; j++ {
			e0[j] = epochGen0(grp.Epoch(j))
			e1[j] = epochGen1(grp.Epoch(j))
			e2[j] = epochGen2(grp.Epoch(j))
		}

		for _, ep0 := range e0 {
			for _, ep1 := range e1 {
				if ep0 == ep1 {
					t.Errorf("Duplicate! %v == %v", ep0, ep1)
				}
			}
			for _, ep2 := range e2 {
				if ep0 == ep2 {
					t.Errorf("Duplicate! %v ==  %v", ep0, ep2)
				}
			}
		}
		for _, ep1 := range e1 {
			for _, ep2 := range e2 {
				if ep1 == ep2 {
					t.Errorf("Duplicate! %v == %v", ep1, ep2)
				}
			}
		}
	}
}
