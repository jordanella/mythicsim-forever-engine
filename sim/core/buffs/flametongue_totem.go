package buffs

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Flametongue Totem, as client 1.60.1.70170 states it (patch 70 of docs/mythicsim-patches.md).
//
// Rank 4 (16387) summons the totem, and the totem's party aura (15036, an area A_PROC_TRIGGER_SPELL) states
// 100% on a landed melee auto attack (ProcTypeMask 4) and triggers 16389, "Flametongue Totem Proc". The proc
// has one effect, a dummy of 1363 that does not scale with level: the tooltip reads "Each main hand hit
// causes (1363 / 77 * mult - 1) to (1363 / 25 * mult) additional Fire damage, based on the speed of the
// weapon". `mult` is Improved Weapon Totems (29192, 29193), which Forever's talent trees do not carry, so it
// is 1.
//
// The value is the same shape as Flametongue Weapon's (16344): hundredths of damage per second of weapon
// speed, with the speed held to 1.3 to 4.0 so that the bounds are the tooltip's. The imbue's rank 6 is 2810
// at 60, the totem's rank 4 is 1363, so the base damage is 48% of the imbue's.
//
// Neither dummy deals damage. The client's only fire spells of the family with a hit table are the three
// "Flametongue Attack" spells (10444, 29469, 29470: Magic in SpellCategories, a 0.1 spell power coefficient,
// class mask bit 21), which the imbue's dummies feed, and the totem's proc has no damage row, no
// SpellCategories row and a class mask (bit 34) that no talent names but Improved Weapon Totems' dummy. The
// engine therefore treats the totem's hit as the imbue's: the same Magic fire spell with the same 0.1
// coefficient, and the talents that name Flametongue Attack (Elemental Fury's crit damage, Elemental
// Weapons' damage) apply to it. That is an inference, not a client row: patch 70 says what changes if the
// hit has no coefficient and no talents.
var flametongueTotemParty = spelldata.MustFind(15036)
var flametongueTotemProc = spelldata.MustFind(16389)
var flametongueAttack = spelldata.MustFind(10444)

// FlametongueAttackTraits is what a class's own Flametongue Attack carries that its talents and threat
// modifiers key on. The shaman sets its own (sim/shaman/weapon_imbues.go): the class mask Flametongue Weapon's
// hit has, so Elemental Fury and Elemental Weapons reach the totem's hit as they reach the imbue's, and its
// spell flag. Any other class has none, and its totem hit takes no talent.
type FlametongueAttackTraits struct {
	ClassSpellMask int64
	Flags          core.SpellFlag
}

var flametongueAttackTraits = map[proto.Class]FlametongueAttackTraits{}

// SetFlametongueAttackTraits states what class's Flametongue Attack carries.
func SetFlametongueAttackTraits(class proto.Class, traits FlametongueAttackTraits) {
	flametongueAttackTraits[class] = traits
}

// FlametongueTotemTriggerLabel names the aura that listens for the main-hand hits. A character has one,
// whichever way the totem reaches it, since a second Flametongue Totem does not add a second hit.
const FlametongueTotemTriggerLabel = "Flametongue Totem Trigger"

// The proc's dummy value at the character's level: 1363, flat, since the row has no per-level gain.
func flametongueTotemValue() float64 {
	return flametongueTotemProc.EffectN(1).Average(core.CharacterLevel)
}

// FlametongueTotemBaseDamage is what a hit adds under a main-hand weapon of this swing speed: the value is
// hundredths of damage per second of speed, held to 1.3 to 4.0 as the tooltip's "(X / 77 - 1) to (X / 25)"
// are. 13.63 a second of speed, so 17.72 at the floor and 54.52 at the cap.
func FlametongueTotemBaseDamage(weaponSpeed float64) float64 {
	return min(max(weaponSpeed, 1.3), 4) * flametongueTotemValue() / 100
}

// FlametongueTotemPriority is what a Flametongue Totem bids for the personal benefit.
func FlametongueTotemPriority() float64 {
	return flametongueTotemValue()
}

// DisableFlametongueTotem makes aura switch a character's own Flametongue Totem benefit off while it is
// up. It is a main-hand Flametongue Weapon: "When applied to main hand, disables any benefit you personally
// receive from Flametongue Totem". It bids twice what the totem does, so the totem's effect never becomes
// the active one while the imbue stands, and takes over again when the imbue goes (an item swap). Windfury
// Weapon and Rockbiter Weapon are not in the category, since the client says this only of Flametongue.
func DisableFlametongueTotem(aura *core.Aura) {
	aura.NewExclusiveEffect(FlametongueTotemCategory, false, core.ExclusiveEffect{Priority: 2 * FlametongueTotemPriority()})
}

// FlametongueTotemAttack is the damage a main-hand auto attack adds under the totem, the imbue's spell with
// the totem's base damage: Magic fire, so it rolls the spell hit and crit tables, with the 0.1 spell power
// coefficient of Flametongue Attack. It keeps the totem's own id (16389) so a report lists it apart from the
// imbue. A weapon with no speed (a druid's paws, no weapon at all) adds nothing, as the imbue's does.
func FlametongueTotemAttack(char *core.Character) *core.Spell {
	traits := flametongueAttackTraits[char.Class]
	return char.GetOrRegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: flametongueTotemProc.ID},
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamageProc,
		ClassSpellMask:   traits.ClassSpellMask,
		Flags:            core.SpellFlagPassiveSpell | core.SpellFlagProc | traits.Flags,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: flametongueAttack.EffectN(1).Coeff(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			weapon := char.MainHand()
			if weapon == nil || weapon.SwingSpeed == 0 {
				return
			}
			spell.CalcAndDealDamage(sim, target, FlametongueTotemBaseDamage(weapon.SwingSpeed), spell.OutcomeMagicHitAndCrit)
		},
	})
}

// FlametongueTotemTrigger is the aura that casts the hit off the character's main-hand auto attacks. It is
// not on by itself: the totem aura that holds the personal benefit turns it on (JoinFlametongueTotem), so
// that a main-hand Flametongue Weapon can turn it off. The row's own proc flags say what it hears, a landed
// melee auto attack, narrowed to the main hand as the tooltip says: an off-hand swing does not add the
// damage. The main hand's autos include the extra attacks Windfury grants, which are ordinary swings of it.
func FlametongueTotemTrigger(char *core.Character) *core.Aura {
	if aura := char.GetAura(FlametongueTotemTriggerLabel); aura != nil {
		return aura
	}
	attack := FlametongueTotemAttack(char)
	trigger := spelldata.ProcTrigger(char, flametongueTotemParty, func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
		attack.Cast(sim, result.Target)
	})
	trigger.Name = FlametongueTotemTriggerLabel
	trigger.ActionID = core.ActionID{}
	trigger.ProcMask &= core.ProcMaskMeleeMH
	trigger.Duration = core.NeverExpires
	trigger.TriggerImmediately = true
	return char.MakeProcTriggerAura(trigger)
}

// JoinFlametongueTotem makes aura a Flametongue Totem the character benefits from: while it is up and no
// main-hand Flametongue Weapon disables it, the trigger is on. The party's totem (driveFlametongueTotem)
// and the shaman's own cast share the category and the trigger, so a totem from both sources adds one hit.
func JoinFlametongueTotem(char *core.Character, aura *core.Aura) {
	trigger := FlametongueTotemTrigger(char)
	aura.NewExclusiveEffect(FlametongueTotemCategory, false, core.ExclusiveEffect{
		Priority: FlametongueTotemPriority(),
		OnGain: func(_ *core.ExclusiveEffect, sim *core.Simulation) {
			trigger.Activate(sim)
		},
		OnExpire: func(_ *core.ExclusiveEffect, sim *core.Simulation) {
			trigger.Deactivate(sim)
		},
	})
}

// A Flametongue Totem another shaman keeps down for the party. It is a fire totem, so it sits in no air
// slot and does not interact with the party's Windfury Totem or Grace of Air; the same character's own
// cast Flametongue Totem and the party's one are the same effect and the category keeps one of them.
func driveFlametongueTotem(char *core.Character, _ *proto.PartyBuffs) {
	// The trigger is registered before the permanent aura that switches it on, so that it is reset first.
	FlametongueTotemTrigger(char)
	aura := char.GetOrRegisterAura(core.Aura{
		Label:    "Flametongue Totem",
		ActionID: core.ActionID{SpellID: 16387, Tag: -1},
		Duration: core.NeverExpires,
	})
	JoinFlametongueTotem(char, aura)
	core.MakePermanent(aura)
}
