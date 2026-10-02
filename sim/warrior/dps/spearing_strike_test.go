package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	googleproto "google.golang.org/protobuf/proto"
)

// The Arms build of the reference set: Spearing Strike 1/1.
const spearingStrikeTalents = "30305213132515201-4505"

func spearingStrikeWarrior(t *testing.T, stance proto.WarriorStance, gear *proto.EquipmentSpec) (*core.Simulation, *DpsWarrior) {
	t.Helper()
	spec := &proto.Player_DpsWarrior{DpsWarrior: googleproto.Clone(DefaultOptions.DpsWarrior).(*proto.DpsWarrior)}
	spec.DpsWarrior.Options.ClassOptions.DefaultStance = stance
	player := &proto.Player{
		Name:          "Spearing Strike",
		Race:          proto.Race_RaceHuman,
		Class:         proto.Class_ClassWarrior,
		Equipment:     gear,
		TalentsString: spearingStrikeTalents,
		Spec:          spec,
		Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(*DpsWarrior)
}

// Client 1.60.1.70170: "Spearing Strike no longer requires a 2handed weapon. Spearing Strike requires
// Battle Stance." The row carries a Battle Stance mask (0x10000) and every melee weapon type.
func TestSpearingStrikeRequiresBattleStance(t *testing.T) {
	for _, tc := range []struct {
		name   string
		stance proto.WarriorStance
		gear   *proto.EquipmentSpec
		want   bool
	}{
		{"battle stance, two-hander", proto.WarriorStance_WarriorStanceBattle, TwoHandGear.GearSet, true},
		{"battle stance, one-hander", proto.WarriorStance_WarriorStanceBattle, DualWieldGear.GearSet, true},
		{"berserker stance, two-hander", proto.WarriorStance_WarriorStanceBerserker, TwoHandGear.GearSet, false},
		{"defensive stance, two-hander", proto.WarriorStance_WarriorStanceDefensive, TwoHandGear.GearSet, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, war := spearingStrikeWarrior(t, tc.stance, tc.gear)
			spearingStrike := war.GetSpell(core.ActionID{SpellID: 1310222})
			if spearingStrike == nil {
				t.Fatal("Spearing Strike is not registered")
			}
			war.AddRage(sim, 50, war.RageRefundMetrics)

			if got := spearingStrike.CanCast(sim, war.CurrentTarget); got != tc.want {
				t.Fatalf("CanCast = %v, want %v", got, tc.want)
			}
		})
	}
}
