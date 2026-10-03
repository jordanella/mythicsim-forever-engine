package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Bane of Agony ramps every four ticks by a step taken when it lands, and Amplify Curse raises that
// step for the one application it is spent on. With a single step shared by every target, landing
// a plain Agony on a second target rewrote the first target's amplified step.
func TestAgonyRampsByItsOwnTargetsStep(t *testing.T) {
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Targets = append(encounter.Targets, core.NewDefaultTarget())
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Warlock", Class: proto.Class_ClassWarlock, Race: proto.Race_RaceOrc, TalentsString: AfflictionTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec:     &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: encounter,
	}, simsignals.CreateSignals())
	sim.Reset()

	warlock := sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
	if warlock.AmplifyCurseAura == nil {
		t.Fatal("the Affliction build no longer takes Amplify Curse; the test needs a build that does")
	}
	targets := sim.Encounter.AllTargetUnits
	amplified, plain := warlock.CurseOfAgony.Dot(targets[0]), warlock.CurseOfAgony.Dot(targets[1])

	warlock.AmplifyCurseAura.Activate(sim)
	amplified.Apply(sim)
	plain.Apply(sim)
	amplifiedStart, plainStart := amplified.SnapshotBaseDamage, plain.SnapshotBaseDamage

	for amplified.TickCount() < 4 || plain.TickCount() < 4 {
		if finished := sim.Step(); finished {
			t.Fatal("the iteration ended before both Agonies ticked four times")
		}
	}

	amplifiedStep := amplified.SnapshotBaseDamage - amplifiedStart
	plainStep := plain.SnapshotBaseDamage - plainStart
	want := 1 + spellData.AmplifyCurse.EffectAt(1).FractionAt(1)
	if plainStep <= 0 || math.Abs(amplifiedStep/plainStep-want) > 1e-9 {
		t.Errorf("the amplified Agony ramped by %.3f and the plain one by %.3f, a ratio of %.4f; want Amplify Curse's %.4f",
			amplifiedStep, plainStep, amplifiedStep/plainStep, want)
	}
}
