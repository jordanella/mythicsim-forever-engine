package druid

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

func (druid *Druid) registerFeralCombatTalents() {
	// Tier 1
	druid.applyFerocity()
	druid.applyHeartOfTheWild()

	// Tier 2
	druid.applyFeralSwiftness()
	druid.applyFeralInstincts()
	druid.applyBrutalImpact()
	druid.applyThickHide()

	// Tier 3
	druid.applySavageFury()
	druid.applyFeralCharge()
	druid.applySharpenedClaws()

	// Tier 4
	druid.applyShreddingAttacks()
	// Primal Bite implemented in primal_bite.go
	druid.applyPredatoryStrikes()
	druid.applyBloodFrenzy()

	// Tier 5
	druid.applyPredatoryInstincts()
	// Leader of the Pack implemented in druid.go
	druid.applyKingOfTheJungle()

	// Tier 6
	druid.applyNaturalReaction()
	druid.applyRendAndTear()

	// Tier 7
	druid.applyBerserk()
}

// Forever swaps the armor multiplier for flat base Armor from level and defense skill, and the
// form's own armor multiplier applies on top of it. The client's two ladders are the share of
// defense skill (0.67 a rank) and the armor a level (1 a rank).
func (druid *Druid) applyThickHide() {
	if druid.Talents.ThickHide == 0 {
		return
	}

	perDefense := spellData.ThickHide.EffectAt(1).FractionAt(druid.Talents.ThickHide)
	perLevel := spellData.ThickHide.EffectAt(2).ValueAt(druid.Talents.ThickHide)

	// Defense skill is the rating the gear carries divided by the rating a point costs.
	defenseSkill := druid.EquipStats()[stats.DefenseRating] / core.DefenseRatingPerDefenseLevel
	armor := perLevel*float64(core.CharacterLevel) + perDefense*defenseSkill
	if druid.StartingForm.Matches(Bear) {
		armor *= BaseBearArmorMulti
	}
	druid.AddStat(stats.Armor, armor)
}

// Predatory Instincts: the client states it as extra critical strike damage on every ability the
// druid uses, which for a feral is its melee attacks.
func (druid *Druid) applyPredatoryInstincts() {
	if druid.Talents.PredatoryInstincts == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidSpellsAll,
		School:     core.SpellSchoolPhysical,
		Kind:       core.SpellMod_CritMultiplier_Flat,
		FloatValue: spellData.PredatoryInstincts.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_CRIT_DAMAGE_BONUS)).FractionAt(druid.Talents.PredatoryInstincts),
	})
}

// +2% Intellect a rank in every form. The Cat and Bear form bonuses are applied dynamically in
// RegisterCatFormAura / RegisterBearFormAura.
func (druid *Druid) applyHeartOfTheWild() {
	if druid.Talents.HeartOfTheWild == 0 {
		return
	}

	druid.MultiplyStat(stats.Intellect, spellData.HeartOfTheWild.Effect(dbcenums.A_MOD_TOTAL_STAT_PERCENTAGE, 0).MultiplierAt(druid.Talents.HeartOfTheWild))
}

// +2% Strength a rank in Cat Form; the client files it under a dummy effect.
func heartOfTheWildFormMultiplier(points int32) float64 {
	return spellData.HeartOfTheWild.EffectAt(3).MultiplierAt(points)
}

// +4% Stamina a rank in Bear Form.
func heartOfTheWildBearStaminaMultiplier(points int32) float64 {
	return spellData.HeartOfTheWild.EffectAt(2).MultiplierAt(points)
}

// Sharpened Claws: +3% critical strike chance a rank while in Cat or Bear Form, applied through
// the form's stat bonus.
func sharpenedClawsCritPercent(points int32) float64 {
	return spellData.SharpenedClaws.Effect(dbcenums.A_MOD_CRIT_PCT, 0).ValueAt(points)
}

func (druid *Druid) applySharpenedClaws() {
	// Applied in forms.go through formShiftStats.
}

// Predatory Strikes: attack power off level while in Cat or Bear Form.
func predatoryStrikesAPPerLevel(points int32) float64 {
	return spellData.PredatoryStrikes.EffectAt(1).FractionAt(points)
}

func (druid *Druid) applyPredatoryStrikes() {
	// Applied in forms.go through formShiftStats.
}

func (druid *Druid) applyFeralSwiftness() {
	if druid.Talents.FeralSwiftness == 0 {
		return
	}

	bonus := stats.Stats{
		stats.DodgeRating: core.DodgeRatingPerDodgePercent * spellData.FeralSwiftness.Effect(dbcenums.A_MOD_DODGE_PERCENT, 0).ValueAt(druid.Talents.FeralSwiftness),
	}
	if druid.CatFormAura != nil {
		druid.CatFormAura.AttachStatsBuff(bonus)
	}
	if druid.BearFormAura != nil {
		druid.BearFormAura.AttachStatsBuff(bonus)
	}
}

// Ferocity: one less Energy or Rage a rank on the feral builders.
func (druid *Druid) applyFerocity() {
	if druid.Talents.Ferocity == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask: DruidSpellClaw | DruidSpellRake | DruidSpellPrimalBite | DruidSpellMaul | DruidSpellSwipe,
		Kind:      core.SpellMod_PowerCost_Flat,
		IntValue:  -druid.Talents.Ferocity,
	})
}

// Forever's Savage Fury names Shred, which Classic's did not. Its class masks (16998: 38912 on the
// damage effect, 4096 on the periodic one) hold Claw/Rake/Shred and Maul/Swipe; Primal Bite (word 1, 64)
// is not in them.
func (druid *Druid) applySavageFury() {
	if druid.Talents.SavageFury == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidSpellClaw | DruidSpellRake | DruidSpellShred | DruidSpellMaul | DruidSpellSwipe,
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: spellData.SavageFury.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).FractionAt(druid.Talents.SavageFury),
	})
}

// Shredding Attacks: 6 less Energy a rank on Shred, and one less Rage a rank on Lacerate.
func (druid *Druid) applyShreddingAttacks() {
	if druid.Talents.ShreddingAttacks == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask: DruidSpellShred,
		Kind:      core.SpellMod_PowerCost_Flat,
		IntValue:  int32(spellData.ShreddingAttacks.EffectAt(1).ValueAt(druid.Talents.ShreddingAttacks)),
	})

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask: DruidSpellLacerate,
		Kind:      core.SpellMod_PowerCost_Flat,
		IntValue:  int32(spellData.ShreddingAttacks.EffectAt(2).ValueAt(druid.Talents.ShreddingAttacks)) / 10,
	})
}

// Blood Frenzy (Primal Fury until build 70009) folds the old combo point proc and Primal Fury's
// Rage together: a combo point on a Cat builder crit, and Rage on a Bear crit.
func (druid *Druid) applyBloodFrenzy() {
	if druid.Talents.BloodFrenzy == 0 {
		return
	}

	procChance := spellData.BloodFrenzy.EffectAt(1).FractionAt(druid.Talents.BloodFrenzy)
	triggered := spellData.BloodFrenzyTriggered.Highest()
	actionID := core.ActionID{SpellID: triggered.ID}
	// The client's E_ENERGIZE is in its own units: 50 is 5 Rage.
	rage := triggered.EffectN(1).BaseValue() / 10
	rageMetrics := druid.NewRageMetrics(actionID)
	cpMetrics := druid.NewComboPointMetrics(actionID)

	druid.MakeProcTriggerAura(core.ProcTrigger{
		Name:           "Blood Frenzy (Cat)",
		ActionID:       actionID,
		Callback:       core.CallbackOnSpellHitDealt,
		ClassSpellMask: DruidSpellBuilder,
		Outcome:        core.OutcomeCrit,
		ProcChance:     procChance,
		ExtraCondition: func(_ *core.Simulation, _ *core.Spell, _ *core.SpellResult) bool {
			return druid.InForm(Cat)
		},
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			druid.AddComboPoints(sim, 1, cpMetrics)
		},
	})

	druid.MakeProcTriggerAura(core.ProcTrigger{
		Name:       "Blood Frenzy (Bear)",
		ActionID:   actionID,
		Callback:   core.CallbackOnSpellHitDealt,
		ProcMask:   core.ProcMaskMelee,
		Outcome:    core.OutcomeCrit,
		ProcChance: procChance,
		ExtraCondition: func(_ *core.Simulation, _ *core.Spell, _ *core.SpellResult) bool {
			return druid.InForm(Bear)
		},
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			druid.AddRage(sim, rage, rageMetrics)
		},
	})
}

// Forever repurposes Feral Instinct: Swipe hits for 10% more a rank.
func (druid *Druid) applyFeralInstincts() {
	if druid.Talents.FeralInstinct == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidSpellSwipe,
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: spellData.FeralInstinct.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).FractionAt(druid.Talents.FeralInstinct),
	})
}

// applyBrutalImpact implements Brutal Impact, new in Forever.
//
// TODO: not modelled - the client states stun duration and a Bash cooldown, neither of which the
// sim casts.
func (druid *Druid) applyBrutalImpact() {
	if druid.Talents.BrutalImpact == 0 {
		return
	}
}

// applyFeralCharge implements Feral Charge, new in Forever.
//
// TODO: not modelled - a movement ability with no damage the sim would count.
func (druid *Druid) applyFeralCharge() {
	if !druid.Talents.FeralCharge {
		return
	}
}

// King of the Jungle grants Energy on Tiger's Fury; applied in tigers_fury.go.
func (druid *Druid) applyKingOfTheJungle() {
}

// Natural Reaction, new in Forever: dodge chance, and a chance at Rage on every dodge.
func (druid *Druid) applyNaturalReaction() {
	if druid.Talents.NaturalReaction == 0 {
		return
	}

	druid.AddStat(stats.DodgeRating, core.DodgeRatingPerDodgePercent*spellData.NaturalReaction.Effect(dbcenums.A_MOD_DODGE_PERCENT, 0).ValueAt(druid.Talents.NaturalReaction))

	procChance := spellData.NaturalReaction.EffectAt(2).FractionAt(druid.Talents.NaturalReaction)
	triggered := spellData.NaturalReactionTriggered.Highest()
	rage := triggered.EffectN(1).BaseValue() / 10
	rageMetrics := druid.NewRageMetrics(core.ActionID{SpellID: triggered.ID})

	druid.MakeProcTriggerAura(core.ProcTrigger{
		Name:       "Natural Reaction",
		ActionID:   core.ActionID{SpellID: triggered.ID},
		Callback:   core.CallbackOnSpellHitTaken,
		Outcome:    core.OutcomeDodge,
		ProcChance: procChance,
		ExtraCondition: func(_ *core.Simulation, _ *core.Spell, _ *core.SpellResult) bool {
			return druid.InForm(Bear)
		},
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			druid.AddRage(sim, rage, rageMetrics)
		},
	})
}

// Rend and Tear, new in Forever: +2% a rank to melee abilities against a bleeding target.
func (druid *Druid) applyRendAndTear() {
	if druid.Talents.RendAndTear == 0 {
		return
	}

	multiplier := spellData.RendAndTear.EffectAt(1).MultiplierAt(druid.Talents.RendAndTear)

	for _, target := range druid.Env.Encounter.AllTargetUnits {
		target.AddDynamicDamageTakenModifier(func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult, _ bool) {
			if spell.Unit == &druid.Unit && spell.ProcMask.Matches(core.ProcMaskMeleeSpecial) && druid.IsBleeding(result.Target) {
				result.Damage *= multiplier
			}
		})
	}
}

func (druid *Druid) IsBleeding(target *core.Unit) bool {
	return (druid.Rip != nil && druid.Rip.Dot(target).IsActive()) ||
		(druid.Rake != nil && druid.Rake.Dot(target).IsActive()) ||
		(druid.Lacerate != nil && druid.Lacerate.Dot(target).IsActive())
}

// Berserk, new in Forever (client 417141): 3 minute cooldown (SpellCooldowns), and for 15 seconds
// +100% critical strike chance on the Combo Point builders (effect 0). Its Primal Bite half (no
// cooldown, up to 3 targets) is in primal_bite.go.
func (druid *Druid) applyBerserk() {
	if !druid.Talents.Berserk {
		return
	}

	actionID := core.ActionID{SpellID: 417141}
	// Effect 0's class mask is Claw/Rake, Shred, Ravage and Pounce (233472); Primal Bite is not in it.
	critMod := druid.AddDynamicMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Percent,
		ClassMask:  DruidSpellClaw | DruidSpellShred | DruidSpellRake | DruidSpellRavage,
		FloatValue: 100,
	})

	druid.BerserkAura = druid.RegisterAura(core.Aura{
		Label:    "Berserk",
		ActionID: actionID,
		Duration: time.Second * 15,
		OnGain: func(_ *core.Aura, _ *core.Simulation) {
			critMod.Activate()
		},
		OnExpire: func(_ *core.Aura, _ *core.Simulation) {
			critMod.Deactivate()
		},
	})

	spell := druid.RegisterSpell(Cat|Bear, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: time.Minute * 3,
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			druid.BerserkAura.Activate(sim)
		},
		RelatedSelfBuff: druid.BerserkAura,
	})

	druid.AddMajorCooldown(core.MajorCooldown{
		Spell: spell.Spell,
		Type:  core.CooldownTypeDPS,
	})
}
