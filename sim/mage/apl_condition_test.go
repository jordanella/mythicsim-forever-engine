package mage

import (
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// A condition that could not be built used to read as no condition, so its action fired every time
// it was ready: an Enhancement preset's "Lightning Bolt at 5 Maelstrom stacks" cast Lightning Bolt
// on cooldown for a Shaman without Maelstrom Weapon. These pin what a condition reads as now.
func TestAPLConditionsThatCannotBeBuilt(t *testing.T) {
	const fireball, frostbolt = 25306, 25304
	spellID := func(id int32) *proto.ActionID { return &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: id}} }
	cast := func(id int32) *proto.APLAction_CastSpell {
		return &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: spellID(id)}}
	}
	auraIsActive := func(id int32) *proto.APLValue {
		return &proto.APLValue{Value: &proto.APLValue_AuraIsActive{AuraIsActive: &proto.APLValueAuraIsActive{AuraId: spellID(id)}}}
	}
	cmp := func(op proto.APLValueCompare_ComparisonOperator, lhs *proto.APLValue, rhs string) *proto.APLValue {
		return &proto.APLValue{Value: &proto.APLValue_Cmp{Cmp: &proto.APLValueCompare{
			Op: op, Lhs: lhs, Rhs: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: rhs}}},
		}}}
	}
	// No handler in this sim: Solar Energy is a Cataclysm Balance resource.
	unsupported := &proto.APLValue{Value: &proto.APLValue_CurrentSolarEnergy{CurrentSolarEnergy: &proto.APLValueCurrentSolarEnergy{}}}
	manaAbove := cmp(proto.APLValueCompare_OpGt, &proto.APLValue{Value: &proto.APLValue_CurrentManaPercent{CurrentManaPercent: &proto.APLValueCurrentManaPercent{}}}, "0%")
	// An aura and a spell no mage has.
	const missing = 1

	player := func(condition *proto.APLValue) *proto.Player {
		return &proto.Player{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome, TalentsString: FrostTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec: &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{
				{Action: &proto.APLAction{Condition: condition, Action: cast(fireball)}},
				{Action: &proto.APLAction{Action: cast(frostbolt)}},
			}},
		}
	}
	raid := func(condition *proto.APLValue) *proto.Raid {
		return core.SinglePlayerRaidProto(player(condition), &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	}
	fireballs := func(t *testing.T, condition *proto.APLValue) float64 {
		t.Helper()
		const iterations = 10
		result := core.RunRaidSim(&proto.RaidSimRequest{
			Raid:       raid(condition),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 1},
		})
		if result.Error != nil {
			t.Fatalf("sim failed: %s", result.Error.Message)
		}
		casts := 0.0
		for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
			if action.Id.GetSpellId() == fireball {
				for _, target := range action.Targets {
					casts += float64(target.Casts)
				}
			}
		}
		return casts / iterations
	}
	warnings := func(t *testing.T, condition *proto.APLValue) string {
		t.Helper()
		result := core.ComputeStats(&proto.ComputeStatsRequest{Raid: raid(condition), Encounter: core.MakeSingleTargetEncounter(0)})
		var lines []string
		for _, validation := range result.RaidStats.Parties[0].Players[0].RotationStats.PriorityList[0].Validations {
			lines = append(lines, validation.Validation)
		}
		return strings.Join(lines, "\n")
	}

	tests := []struct {
		name      string
		condition *proto.APLValue
		casts     bool
		warning   string
	}{
		{"an aura the character does not have is never active", auraIsActive(missing), false, "No aura found"},
		{"... and has no stacks", cmp(proto.APLValueCompare_OpGe,
			&proto.APLValue{Value: &proto.APLValue_AuraNumStacks{AuraNumStacks: &proto.APLValueAuraNumStacks{AuraId: spellID(missing)}}}, "1"), false, "No aura found"},
		{"a spell the character does not know is never ready", cmp(proto.APLValueCompare_OpGt,
			&proto.APLValue{Value: &proto.APLValue_SpellTimeToReady{SpellTimeToReady: &proto.APLValueSpellTimeToReady{SpellId: spellID(missing)}}}, "1.5s"), true, "does not know spell"},
		{"a kind this sim has no handler for disables the action", unsupported, false, "currentSolarEnergy is not supported by this sim"},
		{"... even as one clause of an And", &proto.APLValue{Value: &proto.APLValue_And{And: &proto.APLValueAnd{Vals: []*proto.APLValue{manaAbove, unsupported}}}}, false, "never runs"},
		{"... or of an Or", &proto.APLValue{Value: &proto.APLValue_Or{Or: &proto.APLValueOr{Vals: []*proto.APLValue{manaAbove, unsupported}}}}, false, "never runs"},
		{"an Or with a clause that holds still holds", &proto.APLValue{Value: &proto.APLValue_Or{Or: &proto.APLValueOr{Vals: []*proto.APLValue{
			{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: "true"}}}, unsupported}}}}, true, "not supported"},
		{"an empty condition is no condition", &proto.APLValue{}, true, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if casts := fireballs(t, test.condition); (casts > 0) != test.casts {
				t.Errorf("%.1f Fireballs a fight; want casts: %v", casts, test.casts)
			}
			if got := warnings(t, test.condition); !strings.Contains(got, test.warning) {
				t.Errorf("warnings %q do not mention %q", got, test.warning)
			}
		})
	}
}
