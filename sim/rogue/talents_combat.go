package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

func (rogue *Rogue) registerCombatTalents() {
	// Tier 1
	rogue.registerImprovedGouge()
	rogue.registerImprovedSinisterStrike()
	rogue.registerLightningReflexes()

	// Tier 2
	// Improved Slice and Dice implemented in slice_and_dice.go
	rogue.registerDeflection()
	rogue.registerPrecision()

	// Tier 3
	rogue.registerEndurance()
	rogue.registerRiposte()

	// Tier 4
	rogue.registerImprovedSprint()
	rogue.registerImprovedKick()
	rogue.registerFlawlessExecution()
	rogue.registerDualWieldSpecialization()

	// Tier 5
	rogue.registerBladeFlurry()
	rogue.registerHackAndSlash()

	// Tier 6
	rogue.registerWeaponExpertise()
	rogue.registerAggression()

	// Tier 7
	rogue.registerAdrenalineRush()
}

// registerImprovedGouge implements Improved Gouge.
//
// TODO: Not modelled. The talent lengthens Gouge, which our sim does not cast.
func (rogue *Rogue) registerImprovedGouge() {}

func (rogue *Rogue) registerImprovedSinisterStrike() {
	if rogue.Talents.ImprovedSinisterStrike == 0 {
		return
	}

	rogue.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Flat,
		ClassMask: RogueSpellSinisterStrike,
		IntValue:  int32(spellData.ImprovedSinisterStrike.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).ValueAt(rogue.Talents.ImprovedSinisterStrike)),
	})
}

func (rogue *Rogue) registerLightningReflexes() {
	if rogue.Talents.LightningReflexes == 0 {
		return
	}

	rogue.AddStat(stats.DodgeRating, spellData.LightningReflexes.ValueAt(rogue.Talents.LightningReflexes)*core.DodgeRatingPerDodgePercent)
}

func (rogue *Rogue) registerDeflection() {
	if rogue.Talents.Deflection == 0 {
		return
	}

	rogue.AddStat(stats.ParryRating, spellData.Deflection.ValueAt(rogue.Talents.Deflection)*core.ParryRatingPerParryPercent)
}

func (rogue *Rogue) registerPrecision() {
	if rogue.Talents.Precision == 0 {
		return
	}

	// Precision covers Poisons in Forever, and those roll against the spell hit table.
	hit := spellData.Precision.EffectAt(1).ValueAt(rogue.Talents.Precision)
	rogue.AddStat(stats.PhysicalHitPercent, hit)
	rogue.AddStat(stats.SpellHitPercent, spellData.Precision.EffectAt(2).ValueAt(rogue.Talents.Precision))
}

// registerEndurance implements Endurance.
//
// TODO: Not modelled. The talent shortens Sprint and Evasion, neither of which our sim casts.
func (rogue *Rogue) registerEndurance() {}

// registerImprovedSprint implements Improved Sprint.
//
// TODO: Not modelled. Movement only.
func (rogue *Rogue) registerImprovedSprint() {}

// registerImprovedKick implements Improved Kick.
//
// TODO: Not modelled. The talent silences on a successful interrupt, which our sim does not do.
func (rogue *Rogue) registerImprovedKick() {}

// Flawless Execution, new in Forever: the client states a flat cost reduction, which our
// Forever sim reads as ten energy off Eviscerate.
func (rogue *Rogue) registerFlawlessExecution() {
	if !rogue.Talents.FlawlessExecution {
		return
	}

	rogue.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Flat,
		ClassMask: RogueSpellEviscerate,
		IntValue:  int32(spellData.FlawlessExecution.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).ValueAt(1)),
	})
}

func (rogue *Rogue) registerDualWieldSpecialization() {
	if rogue.Talents.DualWieldSpecialization == 0 {
		return
	}

	rogue.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Flat,
		ProcMask:   core.ProcMaskMeleeOH,
		FloatValue: spellData.DualWieldSpecialization.FractionAt(rogue.Talents.DualWieldSpecialization),
	})
}

func (rogue *Rogue) registerBladeFlurry() {
	if !rogue.Talents.BladeFlurry {
		return
	}

	bladeFlurryRank := spellData.BladeFlurry.Highest()
	actionID := core.ActionID{SpellID: bladeFlurryRank.ID}
	attackSpeed := 1 + bladeFlurryRank.Effect(dbcenums.A_MOD_MELEE_HASTE_3, 0).Average(core.CharacterLevel)/100

	var curDmg float64
	bfHit := rogue.GetOrRegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 22482},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskEmpty, // No proc mask, so it won't proc itself.
		Flags:       core.SpellFlagIgnoreResists | core.SpellFlagIgnoreModifiers | core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell | core.SpellFlagNoOnCastComplete,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, curDmg, spell.OutcomeAlwaysHit)
		},
	})

	rogue.BladeFlurryAura = rogue.GetOrRegisterAura(core.Aura{
		Label:    "Blade Flurry",
		ActionID: actionID,
		Duration: bladeFlurryRank.Duration(),

		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if sim.ActiveTargetCount() < 2 {
				return
			}

			if result.Damage == 0 || !spell.ProcMask.Matches(core.ProcMaskMelee) {
				return
			}

			curDmg = result.Damage
			bfHit.Cast(sim, rogue.Env.NextActiveTargetUnit(result.Target))
			bfHit.SpellMetrics[result.Target.UnitIndex].Casts--
		},
	}).AttachMultiplyAttackSpeed(attackSpeed)

	rogue.BladeFlurry = rogue.GetOrRegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: RogueSpellBladeFlurry,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: bladeFlurryRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: max(bladeFlurryRank.Cooldown(), bladeFlurryRank.CategoryCooldown()),
			},
			IgnoreHaste: true,
		},
		EnergyCost: core.EnergyCostOptions{
			Cost: int32(bladeFlurryRank.Cost()),
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			rogue.BladeFlurryAura.Activate(sim)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell: rogue.BladeFlurry,
		Type:  core.CooldownTypeDPS,
	})
}

// Hack and Slash folds the four Classic weapon specialization talents into one and picks its
// effect from the weapons equipped: 1% extra attack per rank on axes and swords, 1% crit per
// rank on daggers and fists, 3% of the target's armor ignored per rank on maces.
//
// The extra attack's 200 ms internal cooldown is the client's own (13960's ProcCategoryRecovery).
func (rogue *Rogue) registerHackAndSlash() {
	if rogue.Talents.HackAndSlash == 0 {
		return
	}

	points := rogue.Talents.HackAndSlash

	// Axes and swords: extra attack.
	if mask := rogue.GetProcMaskForTypes(proto.WeaponType_WeaponTypeAxe, proto.WeaponType_WeaponTypeSword); mask != core.ProcMaskUnknown {
		rogue.MakeProcTriggerAura(core.ProcTrigger{
			Name:               "Hack and Slash",
			Callback:           core.CallbackOnSpellHitDealt,
			ProcMask:           mask,
			Outcome:            core.OutcomeLanded,
			ProcChance:         spellData.HackAndSlash.EffectAt(1).ValueAt(points) / 100,
			ICD:                spellData.HackAndSlash.Highest().ICD(),
			TriggerImmediately: true,
			Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
				rogue.AutoAttacks.ExtraMHAttack(sim)
			},
		})
	}
	crit := spellData.HackAndSlash.EffectAt(3).ValueAt(points)
	armorIgnore := spellData.HackAndSlash.EffectAt(2).ValueAt(points) / 100

	// Daggers and fists: crit. The character pane shows the bonus for the main hand, so an
	// off-hand-only qualifier gets it on off-hand hits alone.
	switch rogue.GetProcMaskForTypes(proto.WeaponType_WeaponTypeDagger, proto.WeaponType_WeaponTypeFist) {
	case core.ProcMaskMelee:
		rogue.AddStat(stats.PhysicalCritPercent, crit)
	case core.ProcMaskMeleeMH:
		rogue.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusCrit_Percent,
			ProcMask:   core.ProcMaskMeleeMH,
			FloatValue: crit,
		})
	case core.ProcMaskMeleeOH:
		rogue.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusCrit_Percent,
			ProcMask:   core.ProcMaskMeleeOH,
			FloatValue: crit,
		})
	}

	// Maces: a share of the target's armor ignored.
	if rogue.GetProcMaskForTypes(proto.WeaponType_WeaponTypeMace) != core.ProcMaskUnknown {
		rogue.addArmorIgnore(armorIgnore)
	}
}

// Armor ignored as a share of the target's, which lives on the attack table rather than on a
// stat, so it has to wait until the tables exist.
func (rogue *Rogue) addArmorIgnore(factor float64) {
	rogue.Env.RegisterPostFinalizeEffect(func() {
		for _, at := range rogue.AttackTables {
			at.ArmorIgnoreFactor += factor
		}
	})
}

// Weapon Expertise no longer grants weapon skill; it takes the chance for the rogue's attacks
// to be dodged or parried off the attack table directly.
func (rogue *Rogue) registerWeaponExpertise() {
	if rogue.Talents.WeaponExpertise == 0 {
		return
	}

	rogue.AddStat(stats.ExpertisePercent, spellData.WeaponExpertise.ValueAt(rogue.Talents.WeaponExpertise))
}

func (rogue *Rogue) registerAggression() {
	if rogue.Talents.Aggression == 0 {
		return
	}

	rogue.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Flat,
		ClassMask:  RogueSpellSinisterStrike | RogueSpellBackstab | RogueSpellEviscerate,
		FloatValue: spellData.Aggression.FractionAt(rogue.Talents.Aggression),
	})
}

func (rogue *Rogue) registerAdrenalineRush() {
	if !rogue.Talents.AdrenalineRush {
		return
	}

	adrenalineRushRank := spellData.AdrenalineRush.Highest()
	actionID := core.ActionID{SpellID: adrenalineRushRank.ID}
	regenMultiplier := 1 + adrenalineRushRank.Effect(dbcenums.A_MOD_POWER_REGEN_PERCENT, 3).Average(core.CharacterLevel)/100

	rogue.AdrenalineRushAura = rogue.GetOrRegisterAura(core.Aura{
		Label:    "Adrenaline Rush",
		ActionID: actionID,
		Duration: adrenalineRushRank.Duration(),

		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyEnergyRegenSpeed(sim, regenMultiplier)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyEnergyRegenSpeed(sim, 1/regenMultiplier)
		},
	})

	rogue.AdrenalineRush = rogue.GetOrRegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: RogueSpellAdrenalineRush,

		Cast: core.CastConfig{
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: max(adrenalineRushRank.Cooldown(), adrenalineRushRank.CategoryCooldown()),
			},
			DefaultCast: core.Cast{
				GCD: adrenalineRushRank.GCD(),
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			rogue.AdrenalineRushAura.Activate(sim)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell: rogue.AdrenalineRush,
		Type:  core.CooldownTypeDPS,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			return rogue.CurrentEnergy() <= 45.0
		},
	})
}

// Riposte answers a parried attack. The rogue is not the one being hit in a DPS sim, so it only
// fires when the encounter swings back.
func (rogue *Rogue) registerRiposte() {
	if !rogue.Talents.Riposte {
		return
	}

	riposteRank := spellData.Riposte.Highest()
	actionID := core.ActionID{SpellID: riposteRank.ID}
	weaponDamage := riposteRank.Effect(dbcenums.A_NONE, 0).Average(core.CharacterLevel) / 100

	var riposteReady *core.Aura

	rogue.GetOrRegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    riposteRank.SpellSchool(),
		DefenseType:    riposteRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		ClassSpellMask: RogueSpellRiposte,
		MaxRange:       core.MaxMeleeRange,

		EnergyCost: core.EnergyCostOptions{
			Cost: int32(riposteRank.Cost()),
		},
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: max(riposteRank.Cooldown(), riposteRank.CategoryCooldown()),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return riposteReady.IsActive()
		},

		DamageMultiplier:         weaponDamage,
		DamageMultiplierAdditive: 1,
		ThreatMultiplier:         1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			riposteReady.Deactivate(sim)

			damage := rogue.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
		},
	})

	riposteReady = rogue.RegisterAura(core.Aura{
		Label:    "Riposte Ready",
		ActionID: actionID,
		Duration: riposteRank.Duration(),
	})

	rogue.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Riposte Trigger",
		Callback:           core.CallbackOnSpellHitTaken,
		Outcome:            core.OutcomeParry,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			riposteReady.Activate(sim)
		},
	})
}
