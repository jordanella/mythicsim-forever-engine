package warlock

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Forever dots tick on the caster's current stats. The beta log for a Gnome priest (foreverlogs.gg
// report 2668, encounter 4607) has Shadow Word: Pain ticking 34, 34, 34 and then 38, 38, 38, 39 when
// Eureka! is cast after the dot landed, and the Eureka! window is the only change on the priest. So a
// tick reads the damage multiplier when it lands, not when the dot did.
//
// That log is client 1.60.1.70009's. Client 70170 takes periodic effects off Eureka! (it no longer reaches
// Corruption at all), so this test raises the spell's own damage multiplier, which is the number Eureka!
// raised, while the dot is up.
func TestDotTicksReadTheDamageMultiplierAtTheTick(t *testing.T) {
	player := &proto.Player{
		Name: "lock", Class: proto.Class_ClassWarlock, Race: proto.Race_RaceOrc, TalentsString: AfflictionTalents,
		Equipment: &proto.EquipmentSpec{},
		Spec: &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{
			Summon: proto.WarlockOptions_Succubus, Armor: proto.WarlockOptions_DemonArmor, CurseOptions: proto.WarlockOptions_Elements}}}},
		Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 7},
	}, simsignals.CreateSignals())
	sim.Reset()
	warlock := sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
	target := warlock.CurrentTarget
	corruption := warlock.Corruption

	dot := corruption.Dot(target)
	for i := 0; i < 50 && !dot.IsActive(); i++ {
		corruption.SkipCastAndApplyEffects(sim, target)
	}
	if !dot.IsActive() {
		t.Fatal("Corruption did not land")
	}
	tick := func() float64 {
		return dot.CalcSnapshotDamage(sim, target, corruption.OutcomeExpectedMagicAlwaysHit).Damage
	}

	before := tick()
	corruption.DamageMultiplier *= 1.1
	if got, want := tick()/before, 1.1; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("a tick after the multiplier rose is %.6f of the one before, want %.2f", got, want)
	}
	corruption.DamageMultiplier /= 1.1
	if got := tick() / before; got < 1-1e-9 || got > 1+1e-9 {
		t.Errorf("a tick after the multiplier fell back is %.6f of the first, want 1", got)
	}
}
