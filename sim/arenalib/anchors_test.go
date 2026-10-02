package arenalib

// Reshaping is the one piece of the search that moves points without measuring anything, so
// nothing downstream would notice it going wrong - the climb would simply start from a build
// of the wrong shape and report it confidently. Every class, every tree, every shape.

import (
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

var allClasses = []proto.Class{
	proto.Class_ClassDruid, proto.Class_ClassHunter, proto.Class_ClassMage,
	proto.Class_ClassPaladin, proto.Class_ClassPriest, proto.Class_ClassRogue,
	proto.Class_ClassShaman, proto.Class_ClassWarlock, proto.Class_ClassWarrior,
}

func TestReshapeHoldsItsShape(t *testing.T) {
	for _, class := range allClasses {
		trees, err := loadTrees(class)
		if err != nil {
			t.Fatalf("%s: %s", class, err)
		}

		// From an empty build, which is the hardest start: every point has to be placed, and
		// placed somewhere the row gates and prerequisites allow.
		empty := make(allocation, len(trees))
		for i := range trees {
			empty[i] = make([]int, len(trees[i].Talents))
		}

		anchors, err := anchorsFor(class)
		if err != nil {
			t.Fatalf("%s: %s", class, err)
		}
		for _, shape := range anchors {
			shaped, err := reshape(trees, empty, shape)
			if err != nil {
				t.Errorf("%s, %s: %s", class, shape.label, err)
				continue
			}
			for _, h := range shape.holds {
				if got := sum(shaped[h.tree]); got != h.points {
					t.Errorf("%s, %s: %s has %d points, not %d", class, shape.label, trees[h.tree].Name, got, h.points)
				}
			}
			if got := shaped.total(); got != talentBudget {
				t.Errorf("%s, %s: build spends %d points, not %d", class, shape.label, got, talentBudget)
			}
			if !shaped.valid(trees) {
				t.Errorf("%s, %s: produced a build the game would not allow (%s)", class, shape.label, shaped)
			}
		}
	}
}

// Six single-tree anchors and twelve splits, named the way the page prints them.
func TestShapes(t *testing.T) {
	anchors, err := anchorsFor(proto.Class_ClassWarrior)
	if err != nil {
		t.Fatal(err)
	}
	if len(anchors) != 18 {
		t.Fatalf("%d shapes, want 6 anchors and 12 splits", len(anchors))
	}
	labels := map[string]anchor{}
	for _, shape := range anchors {
		if _, dup := labels[shape.label]; dup {
			t.Errorf("two shapes are called %q", shape.label)
		}
		labels[shape.label] = shape
	}
	for _, want := range []string{"31 Arms", "21 Protection", "20 Arms / 31 Fury", "31 Arms / 20 Fury", "21 Fury / 30 Protection"} {
		if _, ok := labels[want]; !ok {
			t.Errorf("no shape called %q", want)
		}
	}
	// A split holds all three trees, the third at nothing.
	split := labels["20 Arms / 31 Fury"]
	if len(split.holds) != 3 || split.holds[2] != (hold{2, 0}) {
		t.Errorf("20 Arms / 31 Fury holds %v", split.holds)
	}
}

// A split climb may only trade inside a tree: moving a point between trees breaks the shape.
func TestSplitOnlyAllowsItsShape(t *testing.T) {
	trees, err := loadTrees(proto.Class_ClassWarrior)
	if err != nil {
		t.Fatal(err)
	}
	// Arms 39/12/0 reshaped to 20/31/0.
	shape := anchor{holds: []hold{{0, 20}, {1, 31}, {2, 0}}, label: "20 Arms / 31 Fury"}
	shaped, err := reshape(trees, parseTalents(trees, "32305213132515201-5502"), shape)
	if err != nil {
		t.Fatal(err)
	}
	if !shape.allows(shaped) {
		t.Fatalf("reshaped to %s, which is not 20/31/0", shaped)
	}
	moved := shaped.clone()
	for j := len(moved[0]) - 1; j >= 0; j-- {
		if moved[0][j] > 0 {
			moved[0][j]--
			break
		}
	}
	moved[2][0]++
	if shape.allows(moved) {
		t.Errorf("a point moved from Arms to Protection still counts as 20/31/0: %s", moved)
	}
}

// Reshaping a build that is already the right shape must leave it alone rather than churn it
// into a different one, or an anchored climb would start somewhere arbitrary every time.
func TestReshapeLeavesAGoodShapeAlone(t *testing.T) {
	trees, err := loadTrees(proto.Class_ClassWarrior)
	if err != nil {
		t.Fatal(err)
	}

	// Fury 17/34/0: 34 points in the Fury tree, 51 in total.
	build := parseTalents(trees, "30305213-55051105050010501")
	for _, shape := range []anchor{
		{holds: []hold{{1, 34}}, label: "34 Fury"},
		{holds: []hold{{0, 17}, {1, 34}, {2, 0}}, label: "17 Arms / 34 Fury"},
	} {
		shaped, err := reshape(trees, build, shape)
		if err != nil {
			t.Fatal(err)
		}
		if shaped.String() != build.String() {
			t.Errorf("%s: reshaped a build that already held its shape: %s -> %s", shape.label, build, shaped)
		}
	}
}
