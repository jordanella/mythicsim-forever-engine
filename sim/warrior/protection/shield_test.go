package protection

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	googleProto "google.golang.org/protobuf/proto"
)

// Dual Wield Specialization on top of the default protection talents.
const shieldAndDualWieldTalents = "05-00000005-255533121331021351"

const (
	mainHandSlot = 14
	offHandSlot  = 15
	dualWieldOH  = 13015 // Serathil, a one-hand axe
	crusader     = 1900
)

// A protection warrior with a main-hand Crusader, Windfury Totem in the party, and either the
// preraid shield or an off-hand weapon.
func shieldTestSim(t *testing.T, talents string, offHand int32) (*core.Simulation, *core.Character, *core.Unit) {
	t.Helper()

	gear := googleProto.Clone(core.GetGearSet("../../../ui/specs/warrior/protection/gear_sets", "preraid").GearSet).(*proto.EquipmentSpec)
	gear.Items[mainHandSlot].Enchant = crusader
	if offHand != 0 {
		gear.Items[offHandSlot] = &proto.ItemSpec{Id: offHand}
	}

	player := &proto.Player{
		Name:          "Shield",
		Race:          proto.Race_RaceOrc,
		Class:         proto.Class_ClassWarrior,
		Equipment:     gear,
		TalentsString: talents,
		Spec:          DefaultOptions,
		Rotation:      core.GetAplRotation("../../../ui/specs/warrior/protection/apls", "protection").Rotation,
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{WindfuryTotem: true}, &proto.RaidBuffs{},
			&proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()

	return sim, sim.Raid.Parties[0].Players[0].GetCharacter(), sim.Encounter.AllTargetUnits[0]
}

func shieldSpells(t *testing.T, war *core.Character) map[string]*core.Spell {
	t.Helper()
	spells := map[string]*core.Spell{
		"Shield Bash": war.GetSpell(core.ActionID{SpellID: 1672}),
		"Shield Slam": war.GetSpell(core.ActionID{SpellID: 23925}),
	}
	for name, spell := range spells {
		if spell == nil {
			t.Fatalf("the warrior has no %s", name)
		}
	}
	return spells
}

// Dual Wield Specialization's off-hand damage is the off-hand weapon's: a shield's hits don't take
// it, and a real off-hand weapon's auto does. (The 70170 hotfixes moved its off-hand hit to Furious
// Precision.)
func TestDualWieldSpecializationLeavesTheShieldAlone(t *testing.T) {
	_, bare, _ := shieldTestSim(t, DefaultProtectionTalents, 0)
	_, specced, _ := shieldTestSim(t, shieldAndDualWieldTalents, 0)
	if specced.AutoAttacks.IsDualWielding {
		t.Fatal("the shield warrior dual-wields")
	}

	bareSpells := shieldSpells(t, bare)
	for name, spell := range shieldSpells(t, specced) {
		if spell.BonusHitPercent != bareSpells[name].BonusHitPercent || spell.DamageMultiplier != bareSpells[name].DamageMultiplier {
			t.Errorf("%s with the talent: hit %v, damage %v; want the untalented %v and %v", name,
				spell.BonusHitPercent, spell.DamageMultiplier, bareSpells[name].BonusHitPercent, bareSpells[name].DamageMultiplier)
		}
	}

	_, bareWeapon, _ := shieldTestSim(t, DefaultProtectionTalents, dualWieldOH)
	_, speccedWeapon, _ := shieldTestSim(t, shieldAndDualWieldTalents, dualWieldOH)
	if !speccedWeapon.AutoAttacks.IsDualWielding {
		t.Fatal("the warrior with an off-hand axe does not dual-wield")
	}
	bareOH, speccedOH := bareWeapon.AutoAttacks.OHAuto(), speccedWeapon.AutoAttacks.OHAuto()
	if speccedOH.DamageMultiplier <= bareOH.DamageMultiplier {
		t.Errorf("the off-hand axe's auto with the talent: damage %v; want above the untalented %v",
			speccedOH.DamageMultiplier, bareOH.DamageMultiplier)
	}
}

// A shield's hit is an off-hand hit: it never procs the main-hand weapon's enchant, which a
// main-hand special does. (Windfury Totem is a party aura since build 70009, not a main-hand
// enchant, so a shield's hit procs it like any other melee special.)
func TestShieldHitsNeverProcMainHandWeaponEffects(t *testing.T) {
	sim, war, _ := shieldTestSim(t, DefaultProtectionTalents, 0)
	mainHandSpecial := &core.Spell{ProcMask: core.ProcMaskMeleeMHSpecial}

	crusaderTrigger := war.GetAura("Enchant Weapon - Crusader")
	if crusaderTrigger == nil || crusaderTrigger.Dpm == nil {
		t.Fatal("the main hand's Crusader registered no proc manager")
	}
	if crusaderTrigger.Dpm.Chance(mainHandSpecial.ProcMask, sim) == 0 {
		t.Fatal("a main-hand special never rolls Crusader, so the check below proves nothing")
	}

	for name, spell := range shieldSpells(t, war) {
		if got := crusaderTrigger.Dpm.Chance(spell.ProcMask, sim); got != 0 {
			t.Errorf("%s rolls the main hand's Crusader at %v, want never", name, got)
		}
	}
}

// A procs-per-minute rate rolled from a shield's hit is measured against the main hand's swing
// speed, since no off-hand weapon is behind it.
func TestShieldHitsRollProcsPerMinuteAtTheMainHandSpeed(t *testing.T) {
	sim, war, _ := shieldTestSim(t, DefaultProtectionTalents, 0)
	dpm := war.NewStaticLegacyPPMManager(1, core.ProcMaskMelee)
	want := war.AutoAttacks.MH().SwingSpeed / 60

	for name, spell := range shieldSpells(t, war) {
		if got := dpm.Chance(spell.ProcMask, sim); math.Abs(got-want) > 1e-12 {
			t.Errorf("%s rolls a 1 PPM rate at %v, want the main hand's %v", name, got, want)
		}
	}
}
