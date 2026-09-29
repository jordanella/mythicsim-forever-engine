package warlock

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Forever dots tick on the caster's current stats. The beta log for a Gnome priest (foreverlogs.gg
// report 2668, encounter 4607) has Shadow Word: Pain ticking 34, 34, 34 and then 38, 38, 38, 39 when
// Eureka! is cast after the dot landed, and the Eureka! window is the only change on the priest. So a
// tick reads the damage multiplier when it lands, not when the dot did.
func TestDotTicksReadTheDamageMultiplierAtTheTick(t *testing.T) {
	player := &proto.Player{
		Name: "lock", Class: proto.Class_ClassWarlock, Race: proto.Race_RaceGnome, TalentsString: AfflictionTalents,
		Equipment: &proto.EquipmentSpec{},
		Spec: &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{
			Summon: proto.WarlockOptions_Succubus, Armor: proto.WarlockOptions_DemonArmor, CurseOptions: proto.WarlockOptions_Elements}}}},
		// Corruption once, Eureka! at 8 seconds (three charges, spent by the Shadow Bolts after it), then Shadow Bolt.
		Rotation: core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[
			{"action":{"condition":{"not":{"val":{"dotIsActive":{"spellId":{"spellId":25311}}}}},"castSpell":{"spellId":{"spellId":25311}}}},
			{"action":{"condition":{"cmp":{"op":"OpGe","lhs":{"currentTime":{}},"rhs":{"const":{"val":"8s"}}}},"castSpell":{"spellId":{"spellId":1259821}}}},
			{"action":{"castSpell":{"spellId":{"spellId":25307}}}}]}`),
	}
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 7, DebugFirstIteration: true},
	})
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	tick := regexp.MustCompile(`\{SpellID: 25311\} \[DEBUG\].*BaseDamage:([0-9.]+), AfterAttackerMods:([0-9.]+)`)
	var multipliers []float64
	for _, line := range strings.Split(result.Logs, "\n") {
		if m := tick.FindStringSubmatch(line); m != nil {
			base, _ := strconv.ParseFloat(m[1], 64)
			after, _ := strconv.ParseFloat(m[2], 64)
			multipliers = append(multipliers, after/base)
		}
	}
	if len(multipliers) < 6 {
		t.Fatalf("only %d Corruption ticks logged", len(multipliers))
	}
	raised := 0
	for _, multiplier := range multipliers {
		if multiplier > multipliers[0]*1.05 {
			raised++
		}
	}
	if raised == 0 {
		t.Errorf("Corruption ticks never rose above %.4f when Eureka! was cast: %v", multipliers[0], multipliers)
	}
	if last := multipliers[len(multipliers)-1]; last > multipliers[0]*1.001 {
		t.Errorf("the last Corruption tick still carries Eureka! (%.4f against %.4f)", last, multipliers[0])
	}
}
