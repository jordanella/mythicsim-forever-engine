package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// A Tallstrider keeps Dust Cloud (client 1265904, armor -505) on the target, so the hunter's own
// physical shots land harder than beside a Turtle, the same pet without it.
func TestTallstriderDustCloud(t *testing.T) {
	run := func(pet proto.HunterOptions_PetType) *proto.RaidSimResult {
		player := &proto.Player{
			Name: "bm", Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, TalentsString: BeastMasteryTalents,
			Equipment: WeaponsOnly, DistanceFromTarget: 30,
			Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{
				Ammo: proto.HunterOptions_Doomshot, QuiverBonus: proto.HunterOptions_Speed15, PetType: pet,
				PetAttackSpeed: proto.HunterOptions_OneTwo, PetUptime: 1}}}},
			Rotation: core.GetAplRotation("../../ui/specs/hunter/dps/apls", "bm").Rotation,
		}
		raid := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, NumActiveParties: 1}
		res := core.RunRaidSim(&proto.RaidSimRequest{Raid: raid, Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{Iterations: 20, RandomSeed: 1}})
		if res.Error != nil {
			t.Fatal(res.Error.Message)
		}
		return res
	}

	strider, turtle := run(proto.HunterOptions_Tallstrider), run(proto.HunterOptions_Turtle)

	var uptime float64
	for _, aura := range strider.EncounterMetrics.Targets[0].Auras {
		if aura.Id.GetSpellId() == spellData.DustCloudTriggered.Highest().ID {
			uptime = aura.UptimeSecondsAvg
		}
	}
	if fight := core.LongDuration; uptime < 0.9*float64(fight) {
		t.Errorf("Dust Cloud up %.0f s of %d s, want nearly all of it", uptime, fight)
	}

	s, u := strider.RaidMetrics.Parties[0].Players[0].Dps.Avg, turtle.RaidMetrics.Parties[0].Players[0].Dps.Avg
	t.Logf("hunter DPS beside a Tallstrider %.1f, a Turtle %.1f; Dust Cloud up %.0f s", s, u, uptime)
	if s < u*1.02 {
		t.Errorf("hunter DPS beside a Tallstrider %.1f, want at least 2%% over a Turtle's %.1f", s, u)
	}
}
