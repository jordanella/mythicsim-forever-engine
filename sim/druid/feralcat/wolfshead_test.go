package feralcat

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/druid"
)

// Forever item 8345 / spell 17768 grants resources from cooldowns, not forms.
// Source: https://www.wowhead.com/forever/item=8345/wolfshead-helm
func TestWolfsheadResourcesComeFromCooldowns(t *testing.T) {
	for _, helm := range []int32{0, 8345} {
		player := &proto.Player{Name: "Wolfshead", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf,
			Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: helm}}},
			Spec:      DefaultSpecOptions, Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL}}
		sim := core.NewSim(&proto.RaidSimRequest{SimOptions: &proto.SimOptions{RandomSeed: 1},
			Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter: core.MakeSingleTargetEncounter(0)}, simsignals.CreateSignals())
		sim.Reset()
		d := sim.Raid.Parties[0].Players[0].(druid.DruidAgent).GetDruid()
		sim.CurrentTime = time.Second
		d.SpendEnergy(sim, d.CurrentEnergy(), d.NewEnergyMetrics(core.ActionID{SpellID: 1}))
		d.ClearForm(sim)
		if !d.CatForm.Cast(sim, d.CurrentTarget) {
			t.Fatal("Cat Form failed")
		}
		if d.CurrentEnergy() != 0 {
			t.Fatalf("helm %d: shifting generated %v energy", helm, d.CurrentEnergy())
		}
		d.GCD.Set(sim.CurrentTime)
		if !d.TigersFury.Cast(sim, d.CurrentTarget) {
			t.Fatal("Tiger's Fury failed")
		}
		want := 0.0
		if helm != 0 {
			want = 20
		}
		if d.CurrentEnergy() != want {
			t.Fatalf("helm %d: Tiger's Fury energy %v, want %v", helm, d.CurrentEnergy(), want)
		}
	}
}
