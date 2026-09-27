package mage

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

func (mage *Mage) registerFireTalents() {
	// Tier 1
	mage.registerWakeOfFire()
	mage.registerIncineration()
	mage.registerImprovedFireball()

	// Tier 2
	mage.registerIgnite()
	mage.registerFlameThrowing()
	mage.registerImpact()

	// Tier 3
	mage.registerBurningSoul()
	mage.registerImprovedFlamestrike()
	// Pyroblast: pyroblast.go

	// Tier 4
	// Improved Scorch: scorch.go
	mage.registerImprovedFireWard()
	mage.registerHotStreak()
	mage.registerMasterOfElements()

	// Tier 5
	mage.registerCriticalMass()
	// Blast Wave: blast_wave.go

	// Tier 6
	mage.registerFirePower()

	// Tier 7
	// Combustion: combustion.go
}

// The Fire Blast cooldown half only. The other half, 1312934 (+50% Fire Blast crit for 30 sec), is
// granted by killing a non-trivial target, which the sim's targets never do.
func (mage *Mage) registerWakeOfFire() {
	if mage.Talents.WakeOfFire == 0 {
		return
	}

	mage.AddStaticMod(core.SpellModConfig{
		ClassMask: MageSpellFireBlast,
		TimeValue: time.Millisecond * time.Duration(spellData.WakeOfFire.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_COOLDOWN)).ValueAt(mage.Talents.WakeOfFire)),
		Kind:      core.SpellMod_Cooldown_Flat,
	})
}

// Arcane Blast and Ice Lance as well as TBC's Fire Blast and Scorch.
func (mage *Mage) registerIncineration() {
	if mage.Talents.Incineration == 0 {
		return
	}

	mage.AddStaticMod(core.SpellModConfig{
		ClassMask:  MageSpellArcaneBlast | MageSpellFireBlast | MageSpellIceLance | MageSpellScorch,
		FloatValue: spellData.Incineration.ValueAt(mage.Talents.Incineration),
		Kind:       core.SpellMod_BonusCrit_Percent,
	})
}

func (mage *Mage) registerImprovedFireball() {
	if mage.Talents.ImprovedFireball == 0 {
		return
	}

	mage.AddStaticMod(core.SpellModConfig{
		ClassMask: MageSpellFireball | MageSpellFrostfireBolt,
		TimeValue: time.Millisecond * time.Duration(spellData.ImprovedFireball.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CASTING_TIME)).ValueAt(mage.Talents.ImprovedFireball)),
		Kind:      core.SpellMod_CastTime_Flat,
	})
}

// Ignite pays out a share of the fire critical strike that lit it over its ticks. A crit landing while
// the dot still runs rolls what it still owes into the new one, so a crit is never paid twice or
// dropped. The share is taken from a hit that has already been through every multiplier, so the ticks
// skip them rather than pay them again, and they never crit.
func (mage *Mage) registerIgnite() {
	if mage.Talents.Ignite == 0 {
		return
	}

	igniteRank := spellData.IgniteTriggered.Highest()
	share := spellData.Ignite.FractionAt(mage.Talents.Ignite)
	tickLength := time.Second * 2
	numTicks := int32(igniteRank.Duration() / tickLength)

	mage.Ignite = mage.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: igniteRank.ID},
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClassSpellMask:   MageSpellIgnite,
		Flags:            core.SpellFlagIgnoreModifiers | core.SpellFlagNoSpellMods | core.SpellFlagNoOnCastComplete | core.SpellFlagIgnoreResists | core.SpellFlagProc,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Ignite",
			},
			NumberOfTicks: numTicks,
			TickLength:    tickLength,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.Spell.CalcAndDealPeriodicDamage(sim, target, dot.SnapshotBaseDamage, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Dot(target).Apply(sim)
		},
	})

	mage.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Ignite Talent",
		CanProcFromProcs:   true, // 11119 carries the bit.
		Callback:           core.CallbackOnSpellHitDealt,
		ProcMask:           core.ProcMaskSpellDamage,
		Outcome:            core.OutcomeCrit,
		TriggerImmediately: true,
		ExtraCondition: func(_ *core.Simulation, spell *core.Spell, _ *core.SpellResult) bool {
			return spell.SpellSchool.Matches(core.SpellSchoolFire) && spell != mage.Ignite
		},
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			dot := mage.Ignite.Dot(result.Target)
			owed := 0.0
			if dot.IsActive() {
				owed = dot.SnapshotBaseDamage * float64(dot.RemainingTicks())
			}

			mage.Ignite.Cast(sim, result.Target)
			dot.SnapshotBaseDamage = (owed + result.Damage*share) / float64(numTicks)
		},
	})
}

// registerFlameThrowing implements Flame Throwing, new in Forever.
//
// TODO: To be implemented. Needs the Forever tooltip and a spellData ladder before
// the effect can be modelled; there is no TBC equivalent to port.
func (mage *Mage) registerFlameThrowing() {
	if mage.Talents.FlameThrowing == 0 {
		return
	}
}

// registerImpact implements Impact, new in Forever.
//
// TODO: To be implemented. Needs the Forever tooltip and a spellData ladder before
// the effect can be modelled; there is no TBC equivalent to port.
func (mage *Mage) registerImpact() {
	if mage.Talents.Impact == 0 {
		return
	}
}

// The threat half only; the pushback protection has nothing to act on in the sim.
func (mage *Mage) registerBurningSoul() {
	if mage.Talents.BurningSoul == 0 {
		return
	}

	mage.AddStaticMod(core.SpellModConfig{
		ClassMask:  MageSpellsAll,
		School:     core.SpellSchoolFire,
		FloatValue: spellData.BurningSoul.Effect(dbcenums.A_MOD_THREAT, 4).FractionAt(mage.Talents.BurningSoul),
		Kind:       core.SpellMod_ThreatMultiplier_Pct,
	})
}

func (mage *Mage) registerImprovedFlamestrike() {
	if mage.Talents.ImprovedFlamestrike == 0 {
		return
	}

	mage.AddStaticMod(core.SpellModConfig{
		ClassMask:  MageSpellFlamestrike,
		FloatValue: spellData.ImprovedFlamestrike.ValueAt(mage.Talents.ImprovedFlamestrike),
		Kind:       core.SpellMod_BonusCrit_Percent,
	})
}

// registerImprovedFireWard implements Improved Fire Ward, new in Forever.
//
// TODO: To be implemented. Needs the Forever tooltip and a spellData ladder before
// the effect can be modelled; there is no TBC equivalent to port.
func (mage *Mage) registerImprovedFireWard() {
	if mage.Talents.ImprovedFireWard == 0 {
		return
	}
}

// Fireball, Frostfire Bolt, Fire Blast and Scorch crits each take 25% off Pyroblast's cast time, stacking 3 times,
// so the stacks are worth holding rather than spending. The buff is 400625: its duration (20 sec since
// build 70009), stack cap and per-stack cast time cut are read from the row.
func (mage *Mage) registerHotStreak() {
	if !mage.Talents.HotStreak {
		return
	}

	buff := spellData.HotStreakTriggered.Highest()
	perStack := buff.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_CASTING_TIME)).Percent()

	castTimeMod := mage.AddDynamicMod(core.SpellModConfig{
		ClassMask: MageSpellPyroblast,
		Kind:      core.SpellMod_CastTime_Pct,
	})

	mage.HotStreakAura = mage.RegisterAura(core.Aura{
		Label:     "Hot Streak",
		ActionID:  core.ActionID{SpellID: buff.ID},
		Duration:  buff.Duration(),
		MaxStacks: int32(buff.MaxStack),
		OnGain: func(_ *core.Aura, _ *core.Simulation) {
			castTimeMod.Activate()
		},
		OnExpire: func(_ *core.Aura, _ *core.Simulation) {
			castTimeMod.Deactivate()
		},
		OnStacksChange: func(_ *core.Aura, _ *core.Simulation, _ int32, newStacks int32) {
			castTimeMod.UpdateFloatValue(perStack * float64(newStacks))
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// 400625 carries one charge: the next Pyroblast spends every stack.
			if spell.Matches(MageSpellPyroblast) {
				aura.Deactivate(sim)
			}
		},
	})

	mage.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Hot Streak Trigger",
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     MageSpellFireball | MageSpellFrostfireBolt | MageSpellFireBlast | MageSpellScorch,
		Outcome:            core.OutcomeCrit,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			mage.HotStreakAura.Activate(sim)
			mage.HotStreakAura.AddStack(sim)
		},
	})
}

func (mage *Mage) registerMasterOfElements() {
	if mage.Talents.MasterOfElements == 0 {
		return
	}

	refundCoeff := spellData.MasterOfElements.FractionAt(mage.Talents.MasterOfElements)
	manaMetrics := mage.NewManaMetrics(core.ActionID{SpellID: spellData.MasterOfElements.Highest().ID})

	mage.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Master of Elements",
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     MageSpellsAll,
		Outcome:            core.OutcomeCrit,
		TriggerImmediately: true,
		ExtraCondition: func(_ *core.Simulation, spell *core.Spell, _ *core.SpellResult) bool {
			return spell.SpellSchool.Matches(core.SpellSchoolFire|core.SpellSchoolFrost) && spell.Cost != nil && spell.CurCast.Cost > 0
		},
		Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
			mage.AddMana(sim, float64(spell.Cost.BaseCost)*refundCoeff, manaMetrics)
		},
	})
}

func (mage *Mage) registerCriticalMass() {
	if mage.Talents.CriticalMass == 0 {
		return
	}

	mage.AddStaticMod(core.SpellModConfig{
		ClassMask:  MageSpellsAll,
		School:     core.SpellSchoolFire,
		FloatValue: spellData.CriticalMass.ValueAt(mage.Talents.CriticalMass),
		Kind:       core.SpellMod_BonusCrit_Percent,
	})
}

// Ignite is left out by its NoSpellMods flag.
func (mage *Mage) registerFirePower() {
	if mage.Talents.FirePower == 0 {
		return
	}

	mage.AddStaticMod(core.SpellModConfig{
		ClassMask:  MageSpellsAll,
		School:     core.SpellSchoolFire,
		FloatValue: spellData.FirePower.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).FractionAt(mage.Talents.FirePower),
		Kind:       core.SpellMod_DamageDone_Flat,
	})
}
