package retribution

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// The default rotation twists Seal of Command into Seal of Righteousness. A build without the
// Seal of Command talent (the arena's 31 Retribution shape) has nothing to twist from, so it
// keeps Seal of Righteousness up instead of going the whole fight with no seal.
func TestNoSealOfCommandStillSeals(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Race:          proto.Race_RaceHuman,
		Class:         proto.Class_ClassPaladin,
		Equipment:     WeaponOnly,
		Consumables:   DefaultConsumables,
		TalentsString: "553232--5022503020133032",
		Rotation:      core.GetAplRotation("../../../ui/specs/paladin/retribution/apls", "default").Rotation,
	}, DefaultOptions)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 101, Iterations: 1},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	})
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	casts := 0
	for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() == 20920 {
			t.Fatalf("cast Seal of Command without the talent")
		}
		if action.Id.GetSpellId() == 20293 {
			for _, target := range action.Targets {
				casts += int(target.Casts)
			}
		}
	}
	if casts == 0 {
		t.Errorf("no Seal of Righteousness cast in a fight without Seal of Command")
	}
}
