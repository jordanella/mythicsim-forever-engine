package feralcat

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/druid"
)

// Client 1.60.1.70170: "Faerie Fire no longer resets your swing timer when used." The sim never reset it
// (Faerie Fire is an instant on the global cooldown and nothing in the cast path touches the swing), so
// this pins that for Cat Form: using it moves neither the next swing.
func TestFaerieFireDoesNotMoveTheNextSwingInCatForm(t *testing.T) {
	// The Cat's paw swings every second.
	for _, wait := range []time.Duration{100 * time.Millisecond, 400 * time.Millisecond, 800 * time.Millisecond} {
		sim := core.NewSim(&proto.RaidSimRequest{
			SimOptions: &proto.SimOptions{RandomSeed: 1},
			Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
				Name: "Cat", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf, TalentsString: DefaultTalents,
				Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{}, Spec: DefaultSpecOptions,
				Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
			}}}}},
			Encounter: core.MakeSingleTargetEncounter(0),
		}, simsignals.CreateSignals())
		sim.Reset()

		cat := sim.Raid.Parties[0].Players[0].(druid.DruidAgent).GetDruid()
		if !cat.InForm(druid.Cat) {
			t.Fatal("the Cat druid does not start in Cat Form")
		}

		// A swing has just landed, so the next one is a full weapon speed away.
		cat.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime)
		sim.CurrentTime += wait
		cat.GCD.Set(sim.CurrentTime)

		mhBefore := cat.AutoAttacks.MainhandSwingAt()
		if mhBefore <= sim.CurrentTime {
			t.Fatalf("wait %v: the next swing at %v is not ahead of %v", wait, mhBefore, sim.CurrentTime)
		}
		if !cat.FaerieFire.Cast(sim, cat.CurrentTarget) {
			t.Fatalf("wait %v: Faerie Fire failed to cast", wait)
		}
		if got := cat.AutoAttacks.MainhandSwingAt(); got != mhBefore {
			t.Errorf("wait %v: Faerie Fire moved the next main hand swing from %v to %v", wait, mhBefore, got)
		}
	}
}
