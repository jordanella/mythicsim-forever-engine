package shaman

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

func (shaman *Shaman) registerEnhancementTalents() {
	// Tier 1
	shaman.applyEarthsGrasp()
	shaman.applyThunderingStrikes()
	shaman.applyAncestralKnowledge()

	// Tier 2
	shaman.applyGuardianTotems()
	shaman.applyMentalDexterity()
	shaman.applyImprovedGhostWolf()
	shaman.applyImprovedLightningShield()

	// Tier 3
	shaman.applyElementalWeapons()
	shaman.applyShamanisticFocus()
	shaman.applyAnticipation()

	// Tier 4
	shaman.applyToughness()
	shaman.applyFlurry()
	shaman.applyStormstrike()

	// Tier 5
	shaman.applySpiritWeapons()
	shaman.applyMentalQuickness()
	shaman.applyImprovedStormstrike()

	// Tier 6
	shaman.applyMaelstromWeapon()

	// Tier 7
	shaman.applyRageOfTheFarseer()
}

func (shaman *Shaman) applyAncestralKnowledge() {
	if shaman.Talents.AncestralKnowledge == 0 {
		return
	}

	shaman.MultiplyStat(stats.Intellect, spellData.AncestralKnowledge.MultiplierAt(shaman.Talents.AncestralKnowledge))
}

func (shaman *Shaman) applyElementalWeapons() {
	if shaman.Talents.ElementalWeapons == 0 {
		return
	}

	points := shaman.Talents.ElementalWeapons
	shaman.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: spellData.ElementalWeapons.EffectAt(1).FractionAt(points),
		ClassMask:  SpellMaskRockbiterWeapon,
	})
	shaman.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: spellData.ElementalWeapons.EffectAt(2).FractionAt(points),
		ClassMask:  SpellMaskFlametongueWeapon | SpellMaskFrostbrandWeapon,
	})
}

func (shaman *Shaman) applyFlurry() {
	if shaman.Talents.Flurry == 0 {
		return
	}

	flurryICD := &core.Cooldown{
		Timer:    shaman.NewTimer(),
		Duration: 500 * time.Millisecond,
	}

	flurryBuff := spellData.FlurryTriggered.Highest()
	attackSpeed := spellData.Flurry.MultiplierAt(shaman.Talents.Flurry)

	flurryAura := shaman.RegisterAura(core.Aura{
		ActionID:  core.ActionID{SpellID: flurryBuff.ID},
		Label:     "Flurry",
		Duration:  flurryBuff.Duration(),
		MaxStacks: int32(flurryBuff.ProcCharges),
	}).AttachMultiplyMeleeSpeed(attackSpeed)

	shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:             "Flurry Trigger",
		Callback:         core.CallbackOnSpellHitDealt,
		ProcMask:         core.ProcMaskMelee,
		CanProcFromProcs: true, // 16256, 16281-16284 carry the bit.
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Outcome.Matches(core.OutcomeCrit) {
				flurryAura.Activate(sim)
				flurryAura.SetStacks(sim, flurryAura.MaxStacks)
				return
			}

			// Remove a stack.
			if flurryAura.IsActive() && spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) && flurryICD.IsReady(sim) {
				flurryICD.Use(sim)
				flurryAura.RemoveStack(sim)
			}
		},
	})
}

func (shaman *Shaman) applyImprovedLightningShield() {
	if shaman.Talents.ImprovedLightningShield == 0 {
		return
	}

	shaman.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: spellData.ImprovedLightningShield.FractionAt(shaman.Talents.ImprovedLightningShield),
		ClassMask:  SpellMaskLightningShield,
	})
}

func (shaman *Shaman) applyMentalQuickness() {
	if shaman.Talents.MentalQuickness == 0 {
		return
	}

	// TBC's Mental Quickness turned attack power into spell damage and cut instant costs. Forever's
	// (30812) does neither: it states Intellect to spell damage and Intellect to spell healing.
	shaman.AddStatDependency(stats.Intellect, stats.SpellDamage,
		spellData.MentalQuickness.Effect(dbcenums.A_MOD_SPELL_DAMAGE_OF_STAT_PERCENT, 126).FractionAt(shaman.Talents.MentalQuickness))
}

func (shaman *Shaman) applyShamanisticFocus() {
	if !shaman.Talents.ShamanisticFocus {
		return
	}
	// 1223030: Shock and Lightning Shield cost 45% less, always.
	shaman.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_PowerCost_Pct_Add,
		FloatValue: -0.45,
		ClassMask:  SpellMaskShock | SpellMaskLightningShield,
	})
}

func (shaman *Shaman) applySpiritWeapons() {
	if !shaman.Talents.SpiritWeapons {
		return
	}

	// Client 16268: parry and -30% threat; its Rockbiter half (eff 1) has nothing to act on, since the sim leaves out Rockbiter's threat.
	shaman.PseudoStats.CanParry = true
	shaman.PseudoStats.ThreatMultiplier *= spellData.SpiritWeapons.Effect(dbcenums.A_MOD_THREAT, 127).MultiplierAt(1)
}

func (shaman *Shaman) applyStormstrike() {
	if !shaman.Talents.Stormstrike {
		return
	}
	shaman.registerStormstrikeSpell()
}

func (shaman *Shaman) applyThunderingStrikes() {
	if shaman.Talents.ThunderingStrikes == 0 {
		return
	}

	shaman.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Percent,
		FloatValue: spellData.ThunderingStrikes.ValueAt(shaman.Talents.ThunderingStrikes),
	})
}

// applyEarthsGrasp implements Earth's Grasp, new in Forever.
//
// TODO: To be implemented. Needs the Forever tooltip and a spellData ladder before
// the effect can be modelled; there is no TBC equivalent to port.
func (shaman *Shaman) applyEarthsGrasp() {
	if shaman.Talents.EarthsGrasp == 0 {
		return
	}
}

// applyGuardianTotems implements Guardian Totems, new in Forever.
//
// TODO: To be implemented. Needs the Forever tooltip and a spellData ladder before
// the effect can be modelled; there is no TBC equivalent to port.
func (shaman *Shaman) applyGuardianTotems() {
	if shaman.Talents.GuardianTotems == 0 {
		return
	}
}

// applyMentalDexterity implements Mental Dexterity, new in Forever: Intellect into attack power.
func (shaman *Shaman) applyMentalDexterity() {
	if shaman.Talents.MentalDexterity == 0 {
		return
	}

	shaman.AddStatDependency(stats.Intellect, stats.AttackPower,
		spellData.MentalDexterity.EffectAt(1).FractionAt(shaman.Talents.MentalDexterity))
}

// applyImprovedGhostWolf implements Improved Ghost Wolf, new in Forever.
//
// TODO: To be implemented. Needs the Forever tooltip and a spellData ladder before
// the effect can be modelled; there is no TBC equivalent to port.
func (shaman *Shaman) applyImprovedGhostWolf() {
	if shaman.Talents.ImprovedGhostWolf == 0 {
		return
	}
}

// applyAnticipation implements Anticipation, new in Forever: flat dodge.
func (shaman *Shaman) applyAnticipation() {
	if shaman.Talents.Anticipation == 0 {
		return
	}

	shaman.AddStat(stats.DodgeRating, core.DodgeRatingPerDodgePercent*
		spellData.Anticipation.Effect(dbcenums.A_MOD_DODGE_PERCENT, 0).ValueAt(shaman.Talents.Anticipation))
}

// applyToughness implements Toughness, new in Forever: more Stamina.
func (shaman *Shaman) applyToughness() {
	if shaman.Talents.Toughness == 0 {
		return
	}

	shaman.MultiplyStat(stats.Stamina,
		spellData.Toughness.Effect(dbcenums.A_MOD_TOTAL_STAT_PERCENTAGE, 0).MultiplierAt(shaman.Talents.Toughness))
}

// applyImprovedStormstrike implements Improved Stormstrike, new in Forever: a chance on Stormstrike
// to regain mana while casting, and a chance for a dodge or parry taken to reset its cooldown. The
// buff (1238931) is the same at both ranks; only the chances scale.
func (shaman *Shaman) applyImprovedStormstrike() {
	if !shaman.Talents.Stormstrike || shaman.Talents.ImprovedStormstrike == 0 {
		return
	}

	chance := spellData.ImprovedStormstrike.EffectAt(1).FractionAt(shaman.Talents.ImprovedStormstrike)
	buff := spellData.ImprovedStormstrikeTriggered.Highest()
	regenRate := buff.Effect(dbcenums.A_MOD_MANA_REGEN_INTERRUPT, 0).Percent()

	focusAura := shaman.RegisterAura(core.Aura{
		Label:    "Improved Stormstrike",
		ActionID: core.ActionID{SpellID: buff.ID},
		Duration: buff.Duration(),
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			shaman.PseudoStats.SpiritRegenRateCasting += regenRate
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			shaman.PseudoStats.SpiritRegenRateCasting -= regenRate
		},
	})

	shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:           "Improved Stormstrike Trigger",
		Callback:       core.CallbackOnCastComplete,
		ClassSpellMask: SpellMaskStormstrikeCast,
		ProcChance:     chance,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			focusAura.Activate(sim)
		},
	})

	shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:       "Improved Stormstrike Reset",
		Callback:   core.CallbackOnSpellHitTaken,
		Outcome:    core.OutcomeDodge | core.OutcomeParry,
		ProcChance: chance,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			shaman.Stormstrike.CD.Reset()
		},
	})
}

// applyMaelstromWeapon implements Maelstrom Weapon, new in Forever: melee hits stack a buff that
// makes the next Lightning Bolt faster and cheaper, and the cast consumes it.
//
// The client states the per-point, per-stack value (408498) and the buff (408505, 30 sec, -20% cast
// and cost at five stacks) but no proc rate at all: no procs-per-minute entry and no proc chance
// below 100. 2 PPM per point is ours, from master; it puts 5/5 at a full stack roughly every 30 sec.
// 408498 is Season of Discovery's rune spell, and the Era client's row carries the same unexplained
// eff1 = 50: there it is the tooltip's "chance increased by 50% with Windfury Weapon, and another 50%
// with a two-hander" on SoD's 10 PPM (wowsims/sod). Forever's tooltip drops that sentence, so 10 PPM
// at 5/5 matches SoD's base; whether the Windfury and two-hander bonuses survive is unconfirmed.
func (shaman *Shaman) applyMaelstromWeapon() {
	if shaman.Talents.MaelstromWeapon == 0 {
		return
	}

	buff := spellData.MaelstromWeaponTriggered.Highest()
	maxStacks := int32(5)
	perStack := spellData.MaelstromWeapon.EffectAt(1).FractionAt(shaman.Talents.MaelstromWeapon)

	ppmm := shaman.NewLegacyPPMManager(2*float64(shaman.Talents.MaelstromWeapon), core.ProcMaskMelee)

	castMod := shaman.AddDynamicMod(core.SpellModConfig{
		Kind:      core.SpellMod_CastTime_Pct,
		ClassMask: SpellMaskLightningBolt,
	})
	costMod := shaman.AddDynamicMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Pct_Add,
		ClassMask: SpellMaskLightningBolt,
	})

	aura := shaman.RegisterAura(core.Aura{
		Label:     "Maelstrom Weapon",
		ActionID:  core.ActionID{SpellID: buff.ID},
		Duration:  buff.Duration(),
		MaxStacks: maxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks, newStacks int32) {
			castMod.UpdateFloatValue(perStack * float64(newStacks))
			costMod.UpdateFloatValue(perStack * float64(newStacks))
			castMod.Activate()
			costMod.Activate()
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			castMod.Deactivate()
			costMod.Deactivate()
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.Matches(SpellMaskLightningBolt) {
				aura.Deactivate(sim)
			}
		},
	})

	shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:     "Maelstrom Weapon Trigger",
		Callback: core.CallbackOnSpellHitDealt,
		ProcMask: core.ProcMaskMelee,
		Outcome:  core.OutcomeLanded,
		DPM:      ppmm,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			aura.Activate(sim)
			aura.AddStack(sim)
		},
	})

}

// applyRageOfTheFarseer implements Rage of the Farseer, new in Forever: a melee haste cooldown
// (425336). Build 70009 dropped its cast-speed effect.
func (shaman *Shaman) applyRageOfTheFarseer() {
	if !shaman.Talents.RageOfTheFarseer {
		return
	}

	rank := spellData.RageOfTheFarseer.Highest()
	multiplier := 1 + rank.Effect(dbcenums.A_MOD_MELEE_RANGED_HASTE_2, 0).Percent()

	buffAura := shaman.RegisterAura(core.Aura{
		Label:    "Rage of the Farseer",
		ActionID: core.ActionID{SpellID: rank.ID},
		Duration: rank.Duration(),
	}).AttachMultiplyMeleeSpeed(multiplier)

	spell := shaman.RegisterSpell(core.SpellConfig{
		ActionID: core.ActionID{SpellID: rank.ID},
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    shaman.NewTimer(),
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			buffAura.Activate(sim)
		},
		RelatedSelfBuff: buffAura,
	})

	shaman.AddMajorCooldown(core.MajorCooldown{Spell: spell, Type: core.CooldownTypeDPS})
}
