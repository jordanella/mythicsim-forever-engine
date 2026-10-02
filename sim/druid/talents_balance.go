package druid

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

func (druid *Druid) registerBalanceTalents() {
	// Tier 1
	druid.applyImprovedWrath()
	druid.applyGenesis()

	// Tier 2
	druid.applyMoonglow()
	druid.applyImprovedMoonfire()
	druid.applyNaturesMajesty()
	druid.applyNaturesReach()

	// Tier 3
	druid.applyImprovedEntanglingRoots()
	druid.applyNaturesSplendor()

	// Tier 4
	druid.applyInsectSwarm()
	druid.applyVengeance()
	druid.applyImprovedStarfire()

	// Tier 5
	druid.applyOvergrowth()
	druid.applyNaturesGrace()
	druid.applyEclipse()

	// Tier 6
	druid.applyMoonfury()

	// Tier 7
	// Moonkin Form implemented in forms.go
}

func (druid *Druid) applyMoonfury() {
	if druid.Talents.Moonfury == 0 {
		return
	}

	// Forever states Moonfury as +2% damage a rank to the Arcane|Nature schools (mask 72) rather
	// than as a spell modifier: a separate multiplier on all Arcane and Nature damage.
	multiplier := spellData.Moonfury.Effect(dbcenums.A_MOD_DAMAGE_PERCENT_DONE, 72).MultiplierAt(druid.Talents.Moonfury)
	druid.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexArcane] *= multiplier
	druid.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexNature] *= multiplier
}

func (druid *Druid) applyMoonglow() {
	if druid.Talents.Moonglow == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		// The client's mask (7340807) is Wrath, Moonfire, Starfire, Insect Swarm, Hurricane and Nature's
		// Grasp ("your damaging spells"): not Moonkin Form, Innervate or the heals.
		ClassMask:  DruidSpellMoonfire | DruidSpellStarfire | DruidSpellWrath | DruidSpellInsectSwarm | DruidSpellHurricane,
		Kind:       core.SpellMod_PowerCost_Pct_Add,
		FloatValue: spellData.Moonglow.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).FractionAt(druid.Talents.Moonglow),
	})
}

// Forever replaces the cast time reduction on the next cast with a short haste buff.
func (druid *Druid) applyNaturesGrace() {
	if !druid.Talents.NaturesGrace {
		return
	}

	triggered := spellData.NaturesGraceTriggered.Highest()
	hasteMultiplier := 1 + triggered.Effect(dbcenums.A_MOD_CASTING_SPEED_NOT_STACK, 0).BaseValue()/100

	// Effect 1 also cuts the global cooldown by 10% ("reducing your global cooldown by $m1%"), on top
	// of the haste. The client's mask (7604215/67108864) is every druid spell but the forms, Innervate,
	// Lifebloom and the cat and bear abilities. Every spell in it has the default 1.5 sec GCD, so the
	// percentage is taken off that.
	gcdReduction := time.Duration(float64(core.GCDDefault) * triggered.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_GLOBAL_COOLDOWN)).Percent())

	aura := druid.RegisterAura(core.Aura{
		Label:    "Nature's Grace",
		ActionID: core.ActionID{SpellID: triggered.ID},
		Duration: triggered.Duration(),
		OnGain: func(_ *core.Aura, sim *core.Simulation) {
			druid.MultiplyCastSpeed(sim, hasteMultiplier)
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			druid.MultiplyCastSpeed(sim, 1/hasteMultiplier)
		},
	}).AttachSpellMod(core.SpellModConfig{
		Kind: core.SpellMod_GlobalCooldown_Flat,
		ClassMask: DruidSpellEntanglingRoots | DruidSpellFaerieFire | DruidSpellHurricane | DruidSpellInsectSwarm |
			DruidSpellMoonfire | DruidSpellStarfire | DruidSpellThorns | DruidSpellWrath | DruidSpellHealingTouch |
			DruidSpellRegrowth | DruidSpellRejuvenation | DruidSpellTranquility | DruidSpellMarkOfTheWild,
		TimeValue: gcdReduction,
	})

	druid.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Nature's Grace Trigger",
		ActionID:           core.ActionID{SpellID: spellData.NaturesGrace.Highest().ID},
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     DruidDamagingSpells,
		Outcome:            core.OutcomeCrit,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			aura.Activate(sim)
		},
	})
}

func (druid *Druid) applyVengeance() {
	if druid.Talents.Vengeance == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidDamagingSpells,
		Kind:       core.SpellMod_CritMultiplier_Flat,
		FloatValue: spellData.Vengeance.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_CRIT_DAMAGE_BONUS)).FractionAt(druid.Talents.Vengeance),
	})
}

// The client states a bonus range, which the sim does not model.
func (druid *Druid) applyNaturesReach() {
	if druid.Talents.NaturesReach == 0 {
		return
	}

	hit := spellData.NaturesReach.Effect(dbcenums.A_MOD_HIT_CHANCE, 0).ValueAt(druid.Talents.NaturesReach)
	druid.AddStat(stats.PhysicalHitPercent, hit)
	druid.AddStat(stats.SpellHitPercent, spellData.NaturesReach.Effect(dbcenums.A_MOD_SPELL_HIT_CHANCE, 0).ValueAt(druid.Talents.NaturesReach))
}

func (druid *Druid) applyInsectSwarm() {
	if !druid.Talents.InsectSwarm {
		return
	}

	druid.registerInsectSwarmSpell()
}

func (druid *Druid) applyImprovedMoonfire() {
	if druid.Talents.ImprovedMoonfire == 0 {
		return
	}

	// 5% a point damage increase to Moonfire and its DoT.
	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidSpellMoonfire,
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: spellData.ImprovedMoonfire.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).FractionAt(druid.Talents.ImprovedMoonfire),
	})

	// 5% a point chance to crit with Moonfire.
	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidSpellMoonfire,
		Kind:       core.SpellMod_BonusCrit_Percent,
		FloatValue: spellData.ImprovedMoonfire.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CRITICAL_CHANCE)).ValueAt(druid.Talents.ImprovedMoonfire),
	})
}

// Improved Wrath, new in Forever: 0.1 sec off the cast and 10% off the cost a rank.
func (druid *Druid) applyImprovedWrath() {
	if druid.Talents.ImprovedWrath == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask: DruidSpellWrath,
		Kind:      core.SpellMod_CastTime_Flat,
		TimeValue: time.Millisecond * time.Duration(spellData.ImprovedWrath.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CASTING_TIME)).ValueAt(druid.Talents.ImprovedWrath)),
	})

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidSpellWrath,
		Kind:       core.SpellMod_PowerCost_Pct_Add,
		FloatValue: spellData.ImprovedWrath.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).FractionAt(druid.Talents.ImprovedWrath),
	})
}

// Genesis, new in Forever: +1% periodic damage and healing a rank.
func (druid *Druid) applyGenesis() {
	if druid.Talents.Genesis == 0 {
		return
	}

	// Client 1223081's mask includes the bleeds (Rake, Rip, Lacerate) and not Lifebloom.
	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidSpellDoT | DruidSpellRake | DruidSpellRip | DruidSpellLacerate | DruidSpellRejuvenation | DruidSpellRegrowth,
		Kind:       core.SpellMod_DotDamageDone_Pct,
		FloatValue: spellData.Genesis.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DOT)).FractionAt(druid.Talents.Genesis),
	})
}

// Nature's Majesty, new in Forever: +2% critical strike chance a rank.
func (druid *Druid) applyNaturesMajesty() {
	if druid.Talents.NaturesMajesty == 0 {
		return
	}

	crit := spellData.NaturesMajesty.Effect(dbcenums.A_MOD_CRIT_PCT, 0).ValueAt(druid.Talents.NaturesMajesty)
	druid.AddStat(stats.SpellCritPercent, crit)
	druid.AddStat(stats.PhysicalCritPercent, crit)
}

// applyImprovedEntanglingRoots implements Improved Entangling Roots, new in Forever.
//
// TODO: not modelled - the client states a duration modifier on a spell the sim does not cast.
func (druid *Druid) applyImprovedEntanglingRoots() {
	if druid.Talents.ImprovedEntanglingRoots == 0 {
		return
	}
}

// Moonfire ticks every 3 sec and Insect Swarm every 2 sec, so the added duration is one extra tick
// on each. The talent has no generated ladder, so the extra tick is the sim's client-read value.
func (druid *Druid) applyNaturesSplendor() {
	if !druid.Talents.NaturesSplendor {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask: DruidSpellDoT,
		Kind:      core.SpellMod_DotNumberOfTicks_Flat,
		IntValue:  1,
	})
}

// Improved Starfire, new in Forever: 0.1 sec off the cast a rank. The stun proc is not modelled.
func (druid *Druid) applyImprovedStarfire() {
	if druid.Talents.ImprovedStarfire == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		ClassMask: DruidSpellStarfire,
		Kind:      core.SpellMod_CastTime_Flat,
		TimeValue: time.Millisecond * time.Duration(spellData.ImprovedStarfire.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CASTING_TIME)).ValueAt(druid.Talents.ImprovedStarfire)),
	})
}

// applyOvergrowth implements Overgrowth, new in Forever.
//
// Not modelled: 17245 lets Entangling Roots hold 1/2 more targets; the sim casts no roots.
func (druid *Druid) applyOvergrowth() {
	if druid.Talents.Overgrowth == 0 {
		return
	}
}

// Eclipse, new in Forever: every Wrath banks charges that each shorten one Starfire cast.
func (druid *Druid) applyEclipse() {
	if druid.Talents.Eclipse == 0 {
		return
	}

	triggered := spellData.EclipseTriggered.Highest()
	// The client's curve is 0.17 / 0.33 / 0.5 sec.
	castTimeReduction := time.Millisecond * time.Duration(spellData.Eclipse.EffectAt(2).ValueAt(druid.Talents.Eclipse))
	const chargesPerWrath = 2

	druid.EclipseAura = druid.RegisterAura(core.Aura{
		Label:     "Eclipse",
		ActionID:  core.ActionID{SpellID: triggered.ID},
		Duration:  triggered.Duration(),
		MaxStacks: 4,
	}).AttachSpellMod(core.SpellModConfig{
		ClassMask: DruidSpellStarfire,
		Kind:      core.SpellMod_CastTime_Flat,
		TimeValue: castTimeReduction,
	})

	core.MakePermanent(druid.RegisterAura(core.Aura{
		Label: "Eclipse Trigger",
		OnCastComplete: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell) {
			switch {
			case spell.Matches(DruidSpellWrath):
				druid.EclipseAura.Activate(sim)
				druid.EclipseAura.AddStacks(sim, chargesPerWrath)
			case spell.Matches(DruidSpellStarfire):
				if druid.EclipseAura.IsActive() {
					druid.EclipseAura.RemoveStack(sim)
				}
			}
		},
	}))
}
