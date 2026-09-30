package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// The Demonic Pact 2/31/18 preset: Demonic Sacrifice, Soul Link and Demonic Pact.
const pactTalents = "113-0005003221220311351-0550005"

func sacrificeAura(t *testing.T, talents string, pact proto.WarlockOptions_Summon) *core.Aura {
	player := core.WithSpec(&proto.Player{
		Race:          proto.Race_RaceOrc,
		Class:         proto.Class_ClassWarlock,
		Equipment:     &proto.EquipmentSpec{},
		Consumables:   &proto.ConsumesSpec{},
		TalentsString: talents,
		Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{
		Summon:        proto.WarlockOptions_Succubus,
		PactSacrifice: pact,
	}}}})
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock().GetAura("Demonic Sacrifice")
}

// Demonic Pact keeps a sacrificed Imp's Shadow buff with the Succubus out, but not a sacrificed
// Succubus (resummoning the sacrificed demon cancels it), and nothing without the talent.
func TestDemonicPactKeepsTheSacrifice(t *testing.T) {
	if aura := sacrificeAura(t, pactTalents, proto.WarlockOptions_Imp); aura == nil || aura.ActionID.SpellID != 18789 {
		t.Errorf("Pact + sacrificed Imp: got %v, want Touch of Shadow (18789)", aura)
	}
	if aura := sacrificeAura(t, pactTalents, proto.WarlockOptions_Succubus); aura != nil {
		t.Errorf("Pact + sacrificed Succubus with the Succubus out: got %v, want none", aura)
	}
	if aura := sacrificeAura(t, "113-0005003221220311350-0550005", proto.WarlockOptions_Imp); aura != nil {
		t.Errorf("no Demonic Pact: got %v, want none", aura)
	}
}

// Bane of Havoc copies 15% of the warlock's damage to other targets onto the baned target (client
// 1225228), and nothing of the damage the baned target takes itself.
func TestBaneOfHavocCopiesDamage(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Race:          proto.Race_RaceOrc,
		Class:         proto.Class_ClassWarlock,
		Equipment:     &proto.EquipmentSpec{},
		Consumables:   &proto.ConsumesSpec{},
		TalentsString: "--0000000000001",
		Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{
		Summon: proto.WarlockOptions_NoSummon,
	}}}})
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Targets = append(encounter.Targets, core.NewDefaultTarget())
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  encounter,
	}, simsignals.CreateSignals())
	sim.Reset()

	warlock := sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
	main, havoc := sim.Encounter.AllTargetUnits[0], sim.Encounter.AllTargetUnits[1]
	bane := warlock.GetSpell(core.ActionID{SpellID: 1225228})
	copied := warlock.GetSpell(core.ActionID{SpellID: 1225228, Tag: 1})
	for i := 0; i < 20 && !havoc.HasActiveAura("Bane of Havoc-"+warlock.Label); i++ {
		bane.SkipCastAndApplyEffects(sim, havoc)
	}

	for i := 0; i < 20 && warlock.SearingPain.SpellMetrics[main.UnitIndex].TotalDamage == 0; i++ {
		warlock.SearingPain.SkipCastAndApplyEffects(sim, main)
	}
	dealt := warlock.SearingPain.SpellMetrics[main.UnitIndex].TotalDamage
	got := copied.SpellMetrics[havoc.UnitIndex].TotalDamage
	if dealt == 0 || math.Abs(got-dealt*0.15) > 1e-6 {
		t.Fatalf("Searing Pain dealt %.2f to the main target, bane copied %.2f, want %.2f", dealt, got, dealt*0.15)
	}

	warlock.SearingPain.SkipCastAndApplyEffects(sim, havoc)
	if again := copied.SpellMetrics[havoc.UnitIndex].TotalDamage; again != got {
		t.Errorf("damage to the baned target itself copied: %.2f -> %.2f", got, again)
	}
}

// The warlock's own Imp gives its party Blood Pact (11767); a sacrificed Imp or another demon doesn't.
func TestImpGivesBloodPact(t *testing.T) {
	for _, tc := range []struct {
		summon    proto.WarlockOptions_Summon
		sacrifice bool
		want      bool
	}{{proto.WarlockOptions_Imp, false, true}, {proto.WarlockOptions_Imp, true, false}, {proto.WarlockOptions_Succubus, false, false}} {
		player := core.WithSpec(&proto.Player{
			Race:        proto.Race_RaceOrc,
			Class:       proto.Class_ClassWarlock,
			Equipment:   &proto.EquipmentSpec{},
			Consumables: &proto.ConsumesSpec{},
			Rotation:    &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}, &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{
			Summon:          tc.summon,
			SacrificeSummon: tc.sacrifice,
		}}}})
		sim := core.NewSim(&proto.RaidSimRequest{
			SimOptions: &proto.SimOptions{RandomSeed: 100},
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
		}, simsignals.CreateSignals())
		sim.Reset()
		aura := sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock().GetAuraByID(core.ActionID{SpellID: 11767, Tag: -1})
		if got := aura != nil && aura.IsActive(); got != tc.want {
			t.Errorf("summon %v, sacrificed %v: Blood Pact up = %v, want %v", tc.summon, tc.sacrifice, got, tc.want)
		}
	}
}

// The Imp's Firebolt rolls client 11763's row: 45 +-11.4% at 60, and the level scaling stops at 63.
func TestImpFireboltRollsItsRow(t *testing.T) {
	if got := impFireboltEffect.Average(60); got != 45 {
		t.Errorf("average at 60 = %v, want 45", got)
	}
	if lo, hi := impFireboltEffect.Min(60), impFireboltEffect.Max(60); math.Abs(lo-42.44) > 0.01 || math.Abs(hi-47.56) > 0.01 {
		t.Errorf("roll at 60 = %.2f to %.2f, want 42.44 to 47.56", lo, hi)
	}
	if got := impFireboltEffect.Average(70); got != 47 {
		t.Errorf("average at 70 = %v, want 47 (capped at 63)", got)
	}
}
