package feralbear

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Forever's Windfury Totem is a party aura (8515 procs 8516), not Classic's
// weapon enchant, so a bear procs it. The extra attack swings the paw: its
// average hit tracks the bear's own melee auto, not the equipped weapon.
func TestWindfuryTotemProcsInBearForm(t *testing.T) {
	run := func(windfury bool) (extra, autos, extraDamage, autoDamage float64) {
		items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
		for i := range items {
			items[i] = &proto.ItemSpec{}
		}
		items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: 7230} // Smite's Mighty Hammer, 3.5 s
		player := &proto.Player{
			Name: "Bear", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf,
			TalentsString: DefaultTalents, Equipment: &proto.EquipmentSpec{Items: items},
			Consumables: DefaultConsumables, Spec: DefaultSpecOptions,
			Rotation: core.GetAplRotation("../../../ui/specs/druid/feralbear/apls", "default").Rotation,
		}
		res := core.RunRaidSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{WindfuryTotem: windfury}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{Iterations: 50, RandomSeed: 1},
		})
		if res.Error != nil {
			t.Fatal(res.Error.Message)
		}
		for _, action := range res.RaidMetrics.Parties[0].Players[0].Actions {
			if action.Id.GetOtherId() != proto.OtherAction_OtherActionAttack {
				continue
			}
			for _, target := range action.Targets {
				landed := float64(target.Hits + target.Crits + target.Glances)
				switch action.Id.GetTag() {
				case 25584:
					extra += landed
					extraDamage += target.Damage
				case 1:
					autos += landed
					autoDamage += target.Damage
				}
			}
		}
		return
	}

	if extra, _, _, _ := run(false); extra != 0 {
		t.Fatalf("%v Windfury attacks without the totem", extra)
	}
	extra, autos, extraDamage, autoDamage := run(true)
	if extra == 0 || autos == 0 {
		t.Fatalf("with the totem: %v Windfury attacks, %v autos; want both", extra, autos)
	}
	// 20% a landed melee hit on a 2.5 s paw plus specials: several a minute.
	if perAuto := extra / autos; perAuto < 0.05 || perAuto > 0.4 {
		t.Errorf("%.3f Windfury attacks per auto, want about 0.1 to 0.3", perAuto)
	}
	// The extra attack carries the proc's attack power, so it hits somewhat harder
	// than an auto; a swing of the 3.5 s hammer would be about three times harder.
	if ratio := (extraDamage / extra) / (autoDamage / autos); ratio < 0.9 || ratio > 1.6 {
		t.Errorf("Windfury hit / auto hit = %.2f; the extra attack is not swinging the paw", ratio)
	}
}
