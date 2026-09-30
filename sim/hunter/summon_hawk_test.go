package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Summon Hawk keeps two hawks out at once (1293527's third effect), each assaulting on its own, and
// their hits can crit so Ferocity reaches them. The dive bomb always hits (client always-hit attribute).
func TestSummonHawkTwoHawks(t *testing.T) {
	player := &proto.Player{
		Name: "bm", Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, TalentsString: BeastMasteryTalents,
		Equipment: WeaponsOnly, DistanceFromTarget: 30,
		Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{
			Ammo: proto.HunterOptions_Doomshot, QuiverBonus: proto.HunterOptions_Speed15, PetType: proto.HunterOptions_Cat,
			PetAttackSpeed: proto.HunterOptions_OneTwo, PetUptime: 1}}}},
		Rotation: core.GetAplRotation("../../ui/specs/hunter/dps/apls", "bm").Rotation,
	}
	raid := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, NumActiveParties: 1}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: raid, Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{Iterations: 10, RandomSeed: 1}})
	if res.Error != nil {
		t.Fatal(res.Error.Message)
	}

	ticks := map[int32]int32{}
	var critTicks, diveBombMisses int32
	for _, action := range res.RaidMetrics.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() != spellData.SummonHawk.Highest().ID {
			continue
		}
		for _, target := range action.Targets {
			if action.Id.GetTag() == 0 {
				diveBombMisses += target.Misses + target.Dodges + target.Parries + target.Blocks
			}
			ticks[action.Id.GetTag()] += target.Ticks + target.CritTicks
			critTicks += target.CritTicks
		}
	}
	if ticks[1] == 0 || ticks[2] == 0 || ticks[3] != 0 {
		t.Fatalf("hawk ticks by hawk: %v, want two hawks both assaulting", ticks)
	}
	if diveBombMisses != 0 {
		t.Fatalf("%d dive bombs missed, were dodged, parried or blocked; the client says they always hit", diveBombMisses)
	}
	if critTicks == 0 {
		t.Fatal("no hawk hit crit")
	}
}
