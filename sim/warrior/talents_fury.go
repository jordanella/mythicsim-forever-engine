package warrior

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

func (warrior *Warrior) registerFuryTalents() {
	// Tier 1
	// Booming Voice (12321) widens the shout radius only, which the sim does not model.
	warrior.registerCruelty()

	// Tier 2
	warrior.registerIronWill()
	warrior.registerUnbridledWrath()

	// Tier 3
	warrior.registerImprovedCleave()
	warrior.registerPiercingHowl()
	warrior.registerBloodCraze()
	// Boundless Rage: warrior.go, when it enables the rage bar

	// Tier 4
	warrior.registerDualWieldSpecialization()
	warrior.registerRagingBlows()
	warrior.registerEnrage()
	warrior.registerImprovedExecute()

	// Tier 5
	warrior.registerPrecision()
	warrior.registerDeathWish()
	warrior.registerImprovedIntercept()

	// Tier 6
	// Improved Berserker Rage: berserker_rage.go
	warrior.registerFlurry()

	// Tier 7
	warrior.registerBloodthirst()
}

func (warrior *Warrior) registerCruelty() {
	if warrior.Talents.Cruelty == 0 {
		return
	}

	warrior.AddStat(stats.PhysicalCritPercent, spellData.Cruelty.ValueAt(warrior.Talents.Cruelty))
}

func (warrior *Warrior) registerUnbridledWrath() {
	if warrior.Talents.UnbridledWrath == 0 {
		return
	}

	unbridledWrathRank := spellData.UnbridledWrathTriggered.Highest()
	unbridledWrathRage := unbridledWrathRank.EnergizeEffect().Tenths()

	rageMetrics := warrior.NewRageMetrics(core.ActionID{SpellID: unbridledWrathRank.ID})

	warrior.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Unbridled Wrath",
		ProcMask:           core.ProcMaskMeleeWhiteHit,
		ProcChance:         spellData.UnbridledWrath.FractionAt(warrior.Talents.UnbridledWrath),
		RequireDamageDealt: true,
		Outcome:            core.OutcomeLanded,
		Callback:           core.CallbackOnSpellHitDealt,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// The tooltip of 12322 doubles the rage for a two-handed weapon.
			twoHanded := warrior.GetMainHandType() == proto.HandType_HandTypeTwoHand
			warrior.AddRage(sim, unbridledWrathRage*core.TernaryFloat64(twoHanded, 2, 1), rageMetrics)
		},
	})
}

func (warrior *Warrior) registerDualWieldSpecialization() {
	if warrior.Talents.DualWieldSpecialization == 0 {
		return
	}

	warrior.AddStaticMod(core.SpellModConfig{
		ProcMask:   core.ProcMaskMeleeOH,
		Kind:       core.SpellMod_DamageDone_Pct,
		FloatValue: spellData.DualWieldSpecialization.Effect(dbcenums.A_MOD_OFFHAND_DAMAGE_PCT, 0).FractionAt(warrior.Talents.DualWieldSpecialization),
	})

	warrior.AddStaticMod(core.SpellModConfig{
		ProcMask:   core.ProcMaskMeleeOH,
		Kind:       core.SpellMod_BonusHit_Percent,
		FloatValue: spellData.DualWieldSpecialization.Effect(dbcenums.A_MOD_HIT_CHANCE, 0).ValueAt(warrior.Talents.DualWieldSpecialization),
	})
}

func (warrior *Warrior) registerIronWill() {
	if warrior.Talents.IronWill == 0 {
		return
	}
	warrior.PseudoStats.FearDurationMultiplier = spellData.IronWill.Effect(dbcenums.A_MECHANIC_DURATION_MOD, 1).MultiplierAt(warrior.Talents.IronWill)
	warrior.PseudoStats.StunDurationMultiplier = spellData.IronWill.Effect(dbcenums.A_MECHANIC_DURATION_MOD, 12).MultiplierAt(warrior.Talents.IronWill)
}

func (warrior *Warrior) registerImprovedExecute() {
	if warrior.Talents.ImprovedExecute == 0 {
		return
	}

	warrior.AddStaticMod(core.SpellModConfig{
		ClassMask: SpellMaskExecute,
		Kind:      core.SpellMod_PowerCost_Flat,
		IntValue:  int32(spellData.ImprovedExecute.TenthsAt(warrior.Talents.ImprovedExecute)),
	})
}

func (warrior *Warrior) registerEnrage() {
	if warrior.Talents.Enrage == 0 {
		return
	}

	enrageBuff := spellData.EnrageTriggered.Highest()

	warrior.EnrageAura = warrior.GetOrRegisterAura(core.Aura{
		Label:    "Enrage",
		ActionID: core.ActionID{SpellID: enrageBuff.ID},
		Duration: enrageBuff.Duration(),
	}).AttachSpellMod(core.SpellModConfig{
		School:     core.SpellSchoolPhysical,
		Kind:       core.SpellMod_DamageDone_Pct,
		FloatValue: spellData.Enrage.FractionAt(warrior.Talents.Enrage),
	})

	warrior.EnrageAura.NewExclusiveEffect("Enrage", true, core.ExclusiveEffect{Priority: spellData.Enrage.ValueAt(warrior.Talents.Enrage)})

	warrior.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Enrage - Trigger",
		Callback:           core.CallbackOnSpellHitTaken,
		Outcome:            core.OutcomeLanded,
		RequireDamageDealt: true,
		ProcChance:         float64(spellData.Enrage.Rank(warrior.Talents.Enrage).ProcChance) / 100,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			warrior.EnrageAura.Activate(sim)
		},
	})
}

func (warrior *Warrior) registerFlurry() {
	if warrior.Talents.Flurry == 0 {
		return
	}

	flurryBuff := spellData.FlurryTriggered.Highest()

	// TODO: Ingame test needed: the talent ladder gives 5% per point (25% at rank 5) while the
	// applied buff 12966 carries a flat 30%. The ladder is kept: the shaman's Flurry buff 16257 still
	// carries Era's rank-1 10% under the same 5..25% ladder, so a triggered buff's own number is not
	// what the server applies. One point of Flurry settles it: the buff reads 5% or 30%.
	flurryAura := warrior.RegisterAura(core.Aura{
		Label:     "Flurry",
		ActionID:  core.ActionID{SpellID: flurryBuff.ID},
		Duration:  flurryBuff.Duration(),
		MaxStacks: int32(flurryBuff.ProcCharges),
	}).AttachMultiplyMeleeSpeed(spellData.Flurry.MultiplierAt(warrior.Talents.Flurry))

	warrior.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Flurry - Trigger",
		ActionID:           core.ActionID{SpellID: 12319},
		ProcMask:           core.ProcMaskMelee,
		TriggerImmediately: true,
		Callback:           core.CallbackOnSpellHitDealt,
		Outcome:            core.OutcomeLanded,

		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.Matches(SpellMaskWhirlwindOh) {
				return
			}

			if result.Outcome.Matches(core.OutcomeCrit) {
				flurryAura.Activate(sim)
				flurryAura.SetStacks(sim, int32(flurryBuff.ProcCharges))
				return
			}

			if flurryAura.IsActive() && spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) {
				flurryAura.RemoveStack(sim)
			}
		},
	})
}

func (warrior *Warrior) registerPrecision() {
	if warrior.Talents.Precision == 0 {
		return
	}

	warrior.AddStat(stats.PhysicalHitPercent, spellData.Precision.Effect(dbcenums.A_MOD_HIT_CHANCE, 0).ValueAt(warrior.Talents.Precision))
	warrior.AddStat(stats.SpellHitPercent, spellData.Precision.Effect(dbcenums.A_MOD_SPELL_HIT_CHANCE, 0).ValueAt(warrior.Talents.Precision))
}

func (warrior *Warrior) registerBloodthirst() {
	if !warrior.Talents.Bloodthirst {
		return
	}

	bloodthirstRank := spellData.Bloodthirst.Highest()
	apShare := bloodthirstRank.Effects[1].Percent()

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: bloodthirstRank.ID},
		Rank:           bloodthirstRank.RankNumber(),
		SpellSchool:    bloodthirstRank.SpellSchool(),
		DefenseType:    bloodthirstRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		ClassSpellMask: SpellMaskBloodthirst,
		MaxRange:       float64(bloodthirstRank.MaxRange),

		RageCost: core.RageCostOptions{
			Cost:   int32(bloodthirstRank.Cost()),
			Refund: bloodthirstRank.MissRefund(),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: bloodthirstRank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: cooldownOf(bloodthirstRank),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := spell.MeleeAttackPower(target)*apShare + bloodthirstRank.DamageEffect().Average(core.CharacterLevel)
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}

// TODO: In-game test required if there's any threat interaction
func (warrior *Warrior) registerPiercingHowl() {
	if !warrior.Talents.PiercingHowl {
		return
	}

	piercingHowlRank := spellData.PiercingHowl.Highest()

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: piercingHowlRank.ID},
		SpellSchool:    core.SpellSchoolPhysical,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: SpellMaskNone,

		RageCost: core.RageCostOptions{
			Cost: int32(piercingHowlRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: piercingHowlRank.GCD(),
			},
			IgnoreHaste: true,
		},
	})
}

func (warrior *Warrior) registerBloodCraze() {
	if warrior.Talents.BloodCraze == 0 {
		return
	}

	bloodCrazeHot := spellData.BloodCrazeTriggered.Highest()
	healthFraction := spellData.BloodCraze.EffectAt(1).FractionAt(warrior.Talents.BloodCraze)
	hitThreshold := spellData.BloodCraze.EffectAt(2).FractionAt(warrior.Talents.BloodCraze)
	tick := bloodCrazeHot.PeriodicEffect()

	bloodCraze := warrior.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: bloodCrazeHot.ID},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskSpellHealing,
		// A heal of max health: the Physical damage-done mods (Two-Handed Weapon Specialization,
		// Enrage, Bastion) do not raise it in the client.
		Flags: core.SpellFlagPassiveSpell | core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete | core.SpellFlagNoSpellMods,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Hot: core.DotConfig{
			Aura:          core.Aura{Label: "Blood Craze"},
			SelfOnly:      true,
			NumberOfTicks: int32(bloodCrazeHot.Duration() / tick.Period()),
			TickLength:    tick.Period(),
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				healPerTick := warrior.MaxHealth() * healthFraction / float64(dot.ExpectedTickCount())
				dot.Spell.CalcAndDealPeriodicHealing(sim, target, healPerTick, dot.OutcomeTick)
			},
		},
	})

	warrior.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Blood Craze - Damage Taken",
		Callback:           core.CallbackOnSpellHitTaken,
		Outcome:            core.OutcomeLanded,
		RequireDamageDealt: true,
		ExtraCondition: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) bool {
			return result.Outcome.Matches(core.OutcomeCrit) || result.Damage > warrior.MaxHealth()*hitThreshold
		},
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			bloodCraze.SelfHot().Apply(sim)
		},
	})

	warrior.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Blood Craze - Bloodthirst",
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     SpellMaskBloodthirst,
		Outcome:            core.OutcomeLanded,
		RequireDamageDealt: true,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			bloodCraze.SelfHot().Apply(sim)
		},
	})
}

func (warrior *Warrior) registerRagingBlows() {
	if !warrior.Talents.RagingBlows {
		return
	}

	warrior.AddStaticMod(core.SpellModConfig{
		ClassMask: SpellMaskCleave,
		Kind:      core.SpellMod_PowerCost_Flat,
		IntValue:  int32(spellData.RagingBlows.EffectAt(2).TenthsAt(1)),
	})
}

func (warrior *Warrior) registerDeathWish() {
	if !warrior.Talents.DeathWish {
		return
	}

	deathWishRank := spellData.DeathWish.Highest()

	actionID := core.ActionID{SpellID: deathWishRank.ID}

	deathWishAura := warrior.RegisterAura(core.Aura{
		Label:    "Death Wish",
		ActionID: actionID,
		Duration: deathWishRank.Duration(),
	}).
		AttachMultiplicativePseudoStatBuff(
			&warrior.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical],
			1+deathWishRank.Effect(dbcenums.A_MOD_DAMAGE_PERCENT_DONE, 1).Percent(),
		).
		AttachMultiplicativePseudoStatBuff(
			&warrior.PseudoStats.DamageTakenMultiplier,
			1+deathWishRank.Effect(dbcenums.A_MOD_DAMAGE_PERCENT_TAKEN, 127).Percent(),
		).
		AttachFearImmunity()

	deathWishSpell := warrior.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: SpellMaskDeathWish,
		Flags:          core.SpellFlagCastWhileIncapacitated,

		RageCost: core.RageCostOptions{
			Cost: int32(deathWishRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: cooldownOf(deathWishRank),
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			deathWishAura.Activate(sim)
			warrior.WaitUntil(sim, sim.CurrentTime+core.GCDDefault)
		},

		RelatedSelfBuff: deathWishAura,
	})

	warrior.AddMajorCooldown(core.MajorCooldown{
		Spell: deathWishSpell,
		Type:  core.CooldownTypeDPS,
	})
}

func (warrior *Warrior) registerImprovedIntercept() {
	if warrior.Talents.ImprovedIntercept == 0 {
		return
	}

	warrior.AddStaticMod(core.SpellModConfig{
		ClassMask: SpellMaskIntercept,
		Kind:      core.SpellMod_Cooldown_Flat,
		TimeValue: time.Duration(spellData.ImprovedIntercept.ValueAt(warrior.Talents.ImprovedIntercept)) * time.Millisecond,
	})
}

func (warrior *Warrior) registerImprovedCleave() {
	if warrior.Talents.ImprovedCleave == 0 {
		return
	}
	warrior.AddStaticMod(core.SpellModConfig{
		ClassMask: SpellMaskCleave,
		Kind:      core.SpellMod_PowerCost_Flat,
		IntValue:  int32(spellData.ImprovedCleave.TenthsAt(warrior.Talents.ImprovedCleave)),
	})
}
