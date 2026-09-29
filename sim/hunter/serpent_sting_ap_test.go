package hunter

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Serpent Sting adds 3.5% of ranged attack power a tick, read at the tick like the rest of a Forever
// dot: an Aspect of the Hawk cast after the sting landed raises the next ticks by that share of the
// aspect's attack power.
func TestSerpentStingTickReadsAttackPowerAtTheTick(t *testing.T) {
	sting, hawk := strconv.Itoa(int(spellData.SerpentSting.Highest().ID)), strconv.Itoa(int(spellData.AspectOfTheHawk.Highest().ID))
	player := &proto.Player{
		Name: "hunter", Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, TalentsString: MarksmanshipTalents,
		Equipment: WeaponsOnly, DistanceFromTarget: 30,
		Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{
			Ammo: proto.HunterOptions_Doomshot, QuiverBonus: proto.HunterOptions_Speed15, PetType: proto.HunterOptions_Cat,
			PetAttackSpeed: proto.HunterOptions_OneTwo, PetUptime: 1}}}},
		Rotation: core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[
			{"action":{"condition":{"not":{"val":{"dotIsActive":{"spellId":{"spellId":` + sting + `}}}}},"castSpell":{"spellId":{"spellId":` + sting + `}}}},
			{"action":{"condition":{"and":{"vals":[
				{"cmp":{"op":"OpGe","lhs":{"currentTime":{}},"rhs":{"const":{"val":"6s"}}}},
				{"not":{"val":{"auraIsActive":{"auraId":{"spellId":` + hawk + `}}}}}]}},"castSpell":{"spellId":{"spellId":` + hawk + `}}}}]}`),
	}
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 7, DebugFirstIteration: true},
	})
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	tick := regexp.MustCompile(`\{SpellID: ` + sting + `\} \[DEBUG\].*RAP: ([0-9.]+),.*BaseDamage:([0-9.]+),`)
	var rangedAttackPower, base []float64
	for _, line := range strings.Split(result.Logs, "\n") {
		if m := tick.FindStringSubmatch(line); m != nil {
			rap, _ := strconv.ParseFloat(m[1], 64)
			damage, _ := strconv.ParseFloat(m[2], 64)
			rangedAttackPower, base = append(rangedAttackPower, rap), append(base, damage)
		}
	}
	if len(base) < 5 {
		t.Fatalf("only %d Serpent Sting ticks logged", len(base))
	}
	// The first application only: the sting runs 15 seconds in ticks of 3, and the rotation casts it
	// again once it has expired, which would take a new snapshot whatever the model.
	last := 3
	if rangedAttackPower[last] <= rangedAttackPower[0]+1 {
		t.Fatalf("the Aspect of the Hawk never raised ranged attack power: %v", rangedAttackPower)
	}
	// Both numbers are printed to a tenth.
	want := 0.035 * (rangedAttackPower[last] - rangedAttackPower[0])
	if got := base[last] - base[0]; got < want-0.15 || got > want+0.15 {
		t.Errorf("the tick rose by %.2f after %.1f more ranged attack power, want %.2f (3.5%%)", got, rangedAttackPower[last]-rangedAttackPower[0], want)
	}
}
