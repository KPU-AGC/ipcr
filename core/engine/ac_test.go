package engine

import (
	"fmt"
	"reflect"
	"testing"
)

func TestScanACEachStreamsHitsAndResetsOnNonACGT(t *testing.T) {
	seeds := []SeedPattern{
		{Pat: []byte("ACG")},
		{Pat: []byte("CG")},
	}
	automaton, _ := buildAC(seeds)

	var got []string
	scanACEach([]byte("TTACGNCGacgACNG"), automaton, func(endPos int, seedIdx int) {
		got = append(got, fmt.Sprintf("%d:%d", endPos, seedIdx))
	})

	want := []string{
		"4:0",  // ACG in TTACG
		"4:1",  // overlapping CG in TTACG
		"7:1",  // CG after N reset
		"10:0", // lowercase acg maps to the same compact transitions
		"10:1", // overlapping lowercase cg
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hits = %#v, want %#v", got, want)
	}
}

func TestBuildACRejectsNonACGTSeeds(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("buildAC accepted a non-ACGT seed")
		}
	}()

	_, _ = buildAC([]SeedPattern{{Pat: []byte("ACN")}})
}
