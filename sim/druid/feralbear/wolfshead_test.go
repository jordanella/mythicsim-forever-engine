package feralbear

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
func TestWolfsheadEnrageBonusDoesNotApplyToShifts(t *testing.T) {
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
		d.ClearForm(sim)
		if !d.BearForm.Cast(sim, d.CurrentTarget) {
			t.Fatal("Bear Form failed")
		}
		if d.CurrentRage() != 0 {
			t.Fatalf("helm %d: shifting generated %v rage", helm, d.CurrentRage())
		}
		d.GCD.Set(sim.CurrentTime)
		if !d.Enrage.Cast(sim, d.CurrentTarget) {
			t.Fatal("Enrage failed")
		}
		want := 10.0
		if helm != 0 {
			want = 15
		}
		if d.CurrentRage() != want {
			t.Fatalf("helm %d: Enrage rage %v, want %v", helm, d.CurrentRage(), want)
		}
	}
}
