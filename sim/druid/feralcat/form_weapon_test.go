package feralcat

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/druid"
)

// Forever's Druid class deep dive (30 September 2026): in Cat, Bear and Dire Bear Form the melee auto
// attack has the equipped weapon's DPS at a 1.0 or 2.5 second swing, and weapon damage abilities use
// the same values. The previous paw was a fixed level 60 weapon that ignored the equipped one.
func TestFormPawCarriesTheEquippedWeaponDPS(t *testing.T) {
	paw := func(t *testing.T, mainHand int32) (*druid.Druid, core.Weapon, core.Weapon) {
		t.Helper()
		items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
		for i := range items {
			items[i] = &proto.ItemSpec{}
		}
		items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: mainHand}
		sim := core.NewSim(&proto.RaidSimRequest{
			SimOptions: &proto.SimOptions{RandomSeed: 1},
			Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
				Name: "Cat", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf, TalentsString: DefaultTalents,
				Equipment: &proto.EquipmentSpec{Items: items}, Buffs: &proto.IndividualBuffs{}, Spec: DefaultSpecOptions,
				Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
			}}}}},
			Encounter: core.MakeSingleTargetEncounter(0),
		}, simsignals.CreateSignals())
		sim.Reset()
		cat := sim.Raid.Parties[0].Players[0].(druid.DruidAgent).GetDruid()
		return cat, cat.GetCatWeapon(), cat.GetBearWeapon()
	}
	dps := func(w core.Weapon) float64 { return (w.BaseDamageMin + w.BaseDamageMax) / 2 / w.SwingSpeed }

	cat, catPaw, bearPaw := paw(t, 7230) // Smite's Mighty Hammer, 3.5 s
	hammer := cat.WeaponFromMainHand()
	if hammer.SwingSpeed != 3.5 {
		t.Fatalf("want the 3.5 s Smite's Mighty Hammer, got a %v s weapon", hammer.SwingSpeed)
	}
	for _, row := range []struct {
		form  string
		paw   core.Weapon
		speed float64
	}{{"Cat", catPaw, 1.0}, {"Bear", bearPaw, 2.5}} {
		if row.paw.SwingSpeed != row.speed || row.paw.NormalizedSwingSpeed != row.speed {
			t.Errorf("%s paw swings every %v s (normalized %v), want %v", row.form, row.paw.SwingSpeed, row.paw.NormalizedSwingSpeed, row.speed)
		}
		if math.Abs(dps(row.paw)-dps(hammer)) > 1e-9 {
			t.Errorf("%s paw %.3f DPS, want the hammer's %.3f", row.form, dps(row.paw), dps(hammer))
		}
		if want := hammer.BaseDamageMin / hammer.BaseDamageMax; math.Abs(row.paw.BaseDamageMin/row.paw.BaseDamageMax-want) > 1e-9 {
			t.Errorf("%s paw spread %.3f, want the hammer's %.3f", row.form, row.paw.BaseDamageMin/row.paw.BaseDamageMax, want)
		}
	}

	// A better weapon is a better paw; an empty hand is the unarmed fist.
	_, fastPaw, _ := paw(t, 15240) // Demon's Claw, 61-115 at 2.3 s
	if dps(fastPaw) <= 0 || math.Abs(dps(fastPaw)-(61+115)/2/2.3) > 0.5 {
		t.Errorf("Demon's Claw paw %.2f DPS, want about %.2f", dps(fastPaw), (61.0+115.0)/2/2.3)
	}
	if _, unarmed, _ := paw(t, 0); dps(unarmed) != dps(core.Weapon{SwingSpeed: 1}) {
		t.Errorf("unarmed paw %.2f DPS, want the unarmed fist", dps(unarmed))
	}
}
