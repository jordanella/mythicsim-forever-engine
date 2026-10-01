package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	pb "google.golang.org/protobuf/proto"
)

// The Crab casts Pinch, the Crocolisk Dismember, the Owl Mine!, the Hyena Tendon Rip and the Spider Web (client
// rows 1264742 / 1264933 / 1265058 / 1265042 / 1265883), the Gorilla Thunderstomp (1264455) and, with 3 enemies up, the Bear
// Swipe (1264502), and the Bat Demoralizing Screech (24579), each landing inside its rank 5 range.
func TestPetStrikes(t *testing.T) {
	for _, c := range []struct {
		pet      proto.HunterOptions_PetType
		id       int32
		min, max float64
		targets  int
	}{
		{proto.HunterOptions_Crab, spellData.PinchTriggered.Highest().ID, 88, 102, 0},
		{proto.HunterOptions_Crocolisk, spellData.DismemberTriggered.Highest().ID, 50, 58, 0},
		{proto.HunterOptions_Owl, spellData.MineTriggered.Highest().ID, 41, 47, 0},
		// The whole bleed, 3 ticks of 20.
		{proto.HunterOptions_Hyena, spellData.TendonRipTriggered.Highest().ID, 60, 60, 0},
		// 4 ticks of 13, Nature.
		{proto.HunterOptions_Spider, spellData.WebTriggered.Highest().ID, 52, 52, 0},
		{proto.HunterOptions_Gorilla, spellData.ThunderstompTriggered.Highest().ID, 122, 142, 0},
		{proto.HunterOptions_Bear, spellData.SwipeTriggered.Highest().ID, 20, 22, 3},
		{proto.HunterOptions_Bat, spellData.DemoralizingScreechTriggered.Highest().ID, 24, 42, 0},
	} {
		player := &proto.Player{
			Name: "bm", Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, TalentsString: BeastMasteryTalents,
			Equipment: WeaponsOnly, DistanceFromTarget: 30,
			Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{
				Ammo: proto.HunterOptions_Doomshot, QuiverBonus: proto.HunterOptions_Speed15, PetType: c.pet,
				PetAttackSpeed: proto.HunterOptions_OneTwo, PetUptime: 1}}}},
			Rotation: core.GetAplRotation("../../ui/specs/hunter/dps/apls", "bm").Rotation,
		}
		raid := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, NumActiveParties: 1}
		// A target with no armor, so a hit shows the row's own damage.
		// The default target is shared, so clone it before changing a stat or every later sim in the
		// package loses the boss's armor too.
		encounter := core.MakeSingleTargetEncounter(0)
		encounter.Targets[0] = pb.Clone(encounter.Targets[0]).(*proto.Target)
		encounter.Targets[0].Stats[proto.Stat_StatArmor] = 0
		for len(encounter.Targets) < c.targets {
			encounter.Targets = append(encounter.Targets, encounter.Targets[0])
		}
		res := core.RunRaidSim(&proto.RaidSimRequest{Raid: raid, Encounter: encounter, SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1}})
		if res.Error != nil {
			t.Fatal(res.Error.Message)
		}

		var hits, targetsHit int32
		var damage float64
		for _, pet := range res.RaidMetrics.Parties[0].Players[0].Pets {
			for _, action := range pet.Actions {
				if action.Id.GetSpellId() != c.id {
					continue
				}
				for _, target := range action.Targets {
					if target.Hits > 0 {
						targetsHit++
					}
					hits += target.Hits
					damage += target.Damage - target.CritDamage
				}
			}
		}
		if hits == 0 {
			t.Fatalf("%v: no hit from spell %d", c.pet, c.id)
		}
		if int(targetsHit) < c.targets {
			t.Fatalf("%v: spell %d hit %d targets, want %d", c.pet, c.id, targetsHit, c.targets)
		}
		avg := damage / float64(hits)
		t.Logf("%v: %d hits, %.1f a hit", c.pet, hits, avg)
		if avg < c.min || avg > c.max*2 {
			t.Fatalf("%v: spell %d averaged %.1f a hit, want the row's %.0f-%.0f before pet multipliers", c.pet, c.id, avg, c.min, c.max)
		}
	}
}
