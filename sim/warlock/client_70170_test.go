package warlock

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func newClient70170Warlock(t *testing.T, race proto.Race, talents string) (*core.Simulation, *Warlock) {
	t.Helper()
	player := &proto.Player{
		Name: "lock", Class: proto.Class_ClassWarlock, Race: race, TalentsString: talents,
		Equipment: &proto.EquipmentSpec{}, Consumables: &proto.ConsumesSpec{},
		Spec: &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{
			Summon: proto.WarlockOptions_NoSummon, Armor: proto.WarlockOptions_DemonArmor}}}},
		Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 7},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
}

// Hellfire Effect (5857, 11681, 11682) lost Cannot Crit and gained Periodic Can Crit in client
// 1.60.1.70170: every tick rolls a crit on each target it hits. The warlock still burns the base tick.
func TestHellfireTicksCanCrit(t *testing.T) {
	sim, warlock := newClient70170Warlock(t, proto.Race_RaceOrc, "")
	target := warlock.CurrentTarget
	healthBefore := warlock.CurrentHealth()

	warlock.Hellfire.BonusCritPercent = 100
	dot := warlock.Hellfire.AOEDot()
	dot.Apply(sim)
	dot.TickOnce(sim)

	m := warlock.Hellfire.SpellMetrics[target.UnitIndex]
	if m.CritTicks != 1 || m.Ticks != 0 {
		t.Fatalf("Hellfire tick at 100%% crit: %d crit ticks and %d normal ticks, want one crit tick", m.CritTicks, m.Ticks)
	}
	if burned := healthBefore - warlock.CurrentHealth(); burned <= 0 || burned > 300 {
		t.Errorf("Hellfire burned %.0f health in one tick, want the rank's base tick (about 206)", burned)
	}
}
