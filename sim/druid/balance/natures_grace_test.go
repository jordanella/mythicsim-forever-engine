package balance

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/druid"
)

// Nature's Grace (16886) hastes casts 10% and, through its second effect, takes 10% off the global
// cooldown as well: an instant Moonfire under it holds the GCD (1.5 s - 10%) / 1.1 = 1.227 s.
func TestNaturesGraceShortensGlobalCooldown(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Owl", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf, TalentsString: DefaultTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec: &proto.Player_BalanceDruid{BalanceDruid: &proto.BalanceDruid{
				Options: &proto.BalanceDruid_Options{ClassOptions: &proto.DruidOptions{}},
			}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	owl := sim.Raid.Parties[0].Players[0].(druid.DruidAgent).GetDruid()
	gcdAfterMoonfire := func() time.Duration {
		for !owl.GCD.IsReady(sim) {
			sim.Step()
		}
		start := sim.CurrentTime
		if !owl.Moonfire.Cast(sim, owl.CurrentTarget) {
			t.Fatal("Moonfire did not cast")
		}
		return owl.GCD.ReadyAt() - start
	}

	if got := gcdAfterMoonfire(); got != core.GCDDefault {
		t.Errorf("Moonfire GCD without Nature's Grace = %v, want %v", got, core.GCDDefault)
	}
	owl.GetAura("Nature's Grace").Activate(sim)
	if got, want := gcdAfterMoonfire(), 1227*time.Millisecond; got != want {
		t.Errorf("Moonfire GCD under Nature's Grace = %v, want %v", got, want)
	}
}
