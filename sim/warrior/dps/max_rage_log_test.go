package dps

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Rage log lines name the bar's real maximum, as energy's do. Boundless Rage and the Gnome racial
// raise it above 100, and anything reading resources from the log (MythicSim's timeline) otherwise
// shows a 100 cap the warrior does not have.
func TestRageLogNamesTheRealMaximum(t *testing.T) {
	for _, tc := range []struct {
		race    proto.Race
		talents string
		max     float64
	}{
		{proto.Race_RaceHuman, ArmsTalents, 100},
		{proto.Race_RaceHuman, "32305213132515201-55020003", 130},        // Boundless Rage 3/3
		{proto.Race_RaceGnome, "32305213132515201-55020003", 130 * 1.05}, // and Expansive Mind
	} {
		player := core.WithSpec(&proto.Player{
			Race:          tc.race,
			Class:         proto.Class_ClassWarrior,
			Equipment:     weaponsOnly(15240, 15238),
			Consumables:   &proto.ConsumesSpec{},
			TalentsString: tc.talents,
			Rotation:      core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "dps_reck").Rotation,
		}, DefaultOptions)
		result := core.RunRaidSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 7, DebugFirstIteration: true},
		})
		if result.Error != nil {
			t.Fatal(result.Error.Message)
		}
		suffix := fmt.Sprintf(") of %0.0f total.", tc.max)
		lines := 0
		for _, line := range strings.Split(result.Logs, "\n") {
			if !strings.Contains(line, " rage from ") {
				continue
			}
			lines++
			if !strings.HasSuffix(line, suffix) {
				t.Fatalf("%v %s: %q does not end in %q", tc.race, tc.talents, line, suffix)
			}
		}
		if lines == 0 {
			t.Fatalf("%v %s: no rage log lines", tc.race, tc.talents)
		}
	}
}
