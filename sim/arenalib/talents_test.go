package arenalib

// The talent search is only worth anything if the builds it reaches are builds the game
// would let somebody spend. A validator that is too permissive produces an illegal "best
// build", which is worse than no answer; one that is too strict quietly fences the search
// off from the peak. Both directions are pinned here.

import (
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

func TestTalentsRoundTrip(t *testing.T) {
	trees, err := loadTrees(proto.Class_ClassWarrior)
	if err != nil {
		t.Fatal(err)
	}

	// The two community warrior builds, straight out of ui/specs/warrior/dps/presets.ts.
	for _, talents := range []string{"30305213-55051105050010501", "32305213132515201-5502"} {
		points := parseTalents(trees, talents)
		if got := points.String(); got != talents {
			t.Errorf("round trip changed the build: %s -> %s", talents, got)
		}
		if total := points.total(); total != 51 {
			t.Errorf("%s spends %d points, not 51", talents, total)
		}
		if !points.valid(trees) {
			t.Errorf("%s is a build people actually run, and the validator rejects it", talents)
		}
	}
}

func TestTalentsValidation(t *testing.T) {
	trees, err := loadTrees(proto.Class_ClassWarrior)
	if err != nil {
		t.Fatal(err)
	}
	base := parseTalents(trees, "30305213-55051105050010501")

	t.Run("over the rank cap", func(t *testing.T) {
		broken := base.clone()
		// Improved Heroic Strike is 3 points; a fourth is not something the game offers.
		broken[0][0] = 4
		if broken.valid(trees) {
			t.Error("accepted a talent past its rank cap")
		}
	})

	t.Run("past the row gate", func(t *testing.T) {
		// Bloodthirst sits on row 6 and needs 30 points in Fury above it. Empty the tree and
		// leave only that point, which no character could ever have spent.
		broken := base.clone()
		for j := range broken[1] {
			broken[1][j] = 0
		}
		broken[1][len(broken[1])-1] = 1
		if broken.valid(trees) {
			t.Error("accepted a deep talent with nothing spent above it")
		}
	})

	t.Run("without its prerequisite", func(t *testing.T) {
		// Flurry has an arrow from Improved Slam. Taking a point out of the prerequisite has
		// to invalidate the talent it feeds, not just itself.
		broken := base.clone()
		found := false
		for i, tree := range trees {
			for j, talent := range tree.Talents {
				if talent.PrereqLocation == nil || broken[i][j] == 0 {
					continue
				}
				for k, other := range tree.Talents {
					if other.Location.RowIdx == talent.PrereqLocation.RowIdx && other.Location.ColIdx == talent.PrereqLocation.ColIdx {
						if broken[i][k] > 0 {
							broken[i][k]--
							found = true
						}
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Skip("this build has no talent standing on a prerequisite")
		}
		if broken.valid(trees) {
			t.Error("accepted a talent whose prerequisite is no longer maxed")
		}
	})
}

// Every class the arena ranks has to have a tree file that parses, or the search fails at
// the far end of a long CI job rather than here.
func TestTalentTreesLoad(t *testing.T) {
	classes := []proto.Class{
		proto.Class_ClassDruid, proto.Class_ClassHunter, proto.Class_ClassMage,
		proto.Class_ClassPaladin, proto.Class_ClassPriest, proto.Class_ClassRogue,
		proto.Class_ClassShaman, proto.Class_ClassWarlock, proto.Class_ClassWarrior,
	}
	for _, class := range classes {
		trees, err := loadTrees(class)
		if err != nil {
			t.Errorf("%s: %s", class, err)
			continue
		}
		if len(trees) != 3 {
			t.Errorf("%s has %d trees, not 3", class, len(trees))
		}
		for _, tree := range trees {
			if len(tree.Talents) == 0 {
				t.Errorf("%s: %s has no talents", class, tree.Name)
			}
		}
	}
}
