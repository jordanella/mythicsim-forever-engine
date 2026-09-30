package enhancement

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Windfury Weapon (16361) is +333 attack power for 3 charges and 2 extra attacks: ordinary main-hand
// white swings, so the swing count rises and no special-hit row (25505) is dealt.
func TestWindfuryWeaponGrantsExtraSwings(t *testing.T) {
	run := func(imbue proto.ShamanImbue) *proto.UnitMetrics {
		player := &proto.Player{
			Name: "enh", Class: proto.Class_ClassShaman, Race: proto.Race_RaceOrc, TalentsString: DefaultTalents,
			Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
				{Id: 17182}, // Sulfuras, Hand of Ragnaros
			}},
			Spec: &proto.Player_EnhancementShaman{EnhancementShaman: &proto.EnhancementShaman{Options: &proto.EnhancementShaman_Options{
				ClassOptions: &proto.ShamanOptions{ImbueMh: imbue},
			}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}
		raid := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, NumActiveParties: 1}
		res := core.RunRaidSim(&proto.RaidSimRequest{Raid: raid, Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{Iterations: 20, RandomSeed: 1}})
		if res.Error != nil {
			t.Fatal(res.Error.Message)
		}
		return res.RaidMetrics.Parties[0].Players[0]
	}
	swings := func(m *proto.UnitMetrics) (n int32) {
		for _, a := range m.Actions {
			if a.Id.GetSpellId() == 25505 {
				t.Fatal("Windfury Weapon still deals its own special hits (25505)")
			}
			if a.Id.GetOtherId() == proto.OtherAction_OtherActionAttack && a.Id.Tag == 1 {
				for _, tgt := range a.Targets {
					n += tgt.Casts
				}
			}
		}
		return n
	}

	plain, wf := run(proto.ShamanImbue_NoImbue), run(proto.ShamanImbue_WindfuryWeapon)
	var procs float64
	for _, a := range wf.Auras {
		if a.Id.GetSpellId() == 16361 {
			procs = a.ProcsAvg
		}
	}
	if procs == 0 {
		t.Fatal("Windfury Weapon's attack power buff (16361) never went up")
	}
	t.Logf("swings %d -> %d, %.1f procs a fight", swings(plain), swings(wf), procs)
	// Each proc owes 2 extra swings, and the timer restarts from the proc.
	if got, want := float64(swings(wf)), float64(swings(plain))+2*20*procs; got < 0.95*want || got > 1.05*want {
		t.Fatalf("main-hand swings with Windfury %v, want ~%v (%v without, %.1f procs a fight)", got, want, swings(plain), procs)
	}
}

// A shaman's own Windfury Totem (10614) procs off the party aura 10612 and hands out 10610's
// attack power for extra main-hand swings; no default APL casts it, so nothing else runs this path.
func TestWindfuryTotemSelfProcs(t *testing.T) {
	player := &proto.Player{
		Name: "enh", Class: proto.Class_ClassShaman, Race: proto.Race_RaceOrc, TalentsString: DefaultTalents,
		Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{
			{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			{Id: 17182}, // Sulfuras, Hand of Ragnaros
		}},
		Spec: &proto.Player_EnhancementShaman{EnhancementShaman: &proto.EnhancementShaman{Options: &proto.EnhancementShaman_Options{ClassOptions: &proto.ShamanOptions{}}}},
		Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{{Action: &proto.APLAction{Action: &proto.APLAction_CastSpell{
			CastSpell: &proto.APLActionCastSpell{SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 10614}}},
		}}}}},
	}
	raid := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, NumActiveParties: 1}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: raid, Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{Iterations: 20, RandomSeed: 1}})
	if res.Error != nil {
		t.Fatal(res.Error.Message)
	}
	m := res.RaidMetrics.Parties[0].Players[0]
	var procs float64
	for _, a := range m.Auras {
		if a.Id.GetSpellId() == 10610 {
			procs = a.ProcsAvg
		}
	}
	var extra int32
	for _, a := range m.Actions {
		if a.Id.GetOtherId() == proto.OtherAction_OtherActionAttack && a.Id.Tag == 10610 {
			for _, tgt := range a.Targets {
				extra += tgt.Casts
			}
		}
	}
	if procs == 0 || extra == 0 {
		t.Fatalf("Windfury Totem: %.1f buff procs a fight, %d extra swings; want both above 0", procs, extra)
	}
}
