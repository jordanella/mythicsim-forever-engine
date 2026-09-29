package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

// Forever's Windfury Totem is a party aura that procs extra attacks, not Classic's weapon enchant,
// so a main-hand Elemental Sharpening Stone keeps its 2% crit beside it.
func TestWindfuryTotemKeepsTheMainHandStone(t *testing.T) {
	crit := func(windfury bool, stone int32) float64 {
		player := core.WithSpec(&proto.Player{
			Race:          proto.Race_RaceHuman,
			Class:         proto.Class_ClassWarrior,
			Equipment:     weaponsOnly(15240, 15238),
			Consumables:   &proto.ConsumesSpec{MhImbueId: stone},
			TalentsString: DpsTalents,
			Rotation:      core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "dps_reck").Rotation,
		}, DefaultOptions)
		sim := core.NewSim(&proto.RaidSimRequest{
			SimOptions: &proto.SimOptions{RandomSeed: 1},
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{WindfuryTotem: windfury}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
		}, simsignals.CreateSignals())
		sim.Reset()
		return sim.Raid.Parties[0].Players[0].GetCharacter().GetStat(stats.PhysicalCritPercent)
	}
	for _, windfury := range []bool{false, true} {
		if got := crit(windfury, 18262) - crit(windfury, 0); got < 1.999 || got > 2.001 {
			t.Errorf("Windfury Totem %v: the main-hand stone adds %v%% crit, want 2", windfury, got)
		}
	}
}
