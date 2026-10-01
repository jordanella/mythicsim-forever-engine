package shaman

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var earthShockRank = spellData.EarthShock.Highest()
var flameShockRank = spellData.FlameShock.Highest()
var frostShockRank = spellData.FrostShock.Highest()

// Shared logic for all shocks.
func (shaman *Shaman) newShockSpellConfig(rank *spelldata.Spell, spellSchool core.SpellSchool, shockTimer *core.Timer) core.SpellConfig {
	return core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.ID},
		SpellSchool: spellSchool,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       SpellFlagShamanSpell | SpellFlagShock | core.SpellFlagAPL | SpellFlagInstant,
		MaxRange:    float64(rank.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    shockTimer,
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: rank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,
	}
}

func (shaman *Shaman) registerEarthShockSpell(shockTimer *core.Timer) {
	config := shaman.newShockSpellConfig(earthShockRank, core.SpellSchoolNature, shockTimer)
	config.ClassSpellMask = SpellMaskEarthShock
	config.Flags |= core.SpellFlagBinary
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := earthShockRank.DamageEffect().Roll(sim, core.CharacterLevel)
		spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
	}

	shaman.EarthShock = shaman.RegisterSpell(config)
}

func (shaman *Shaman) registerFlameShockSpell(shockTimer *core.Timer) {
	tick := flameShockRank.PeriodicEffect()

	config := shaman.newShockSpellConfig(flameShockRank, core.SpellSchoolFire, shockTimer)
	config.ClassSpellMask = SpellMaskFlameShockDirect
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := flameShockRank.DamageEffect().Roll(sim, core.CharacterLevel)
		result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		if result.Landed() {
			spell.RelatedDotSpell.Cast(sim, target)
		}
		spell.DealDamage(sim, result)
	}
	shaman.FlameShock = shaman.RegisterSpell(config)

	shaman.FlameShock.RelatedDotSpell = shaman.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: flameShockRank.ID, Tag: 1},
		SpellSchool:      flameShockRank.SpellSchool(),
		DefenseType:      flameShockRank.DefenseTypeCore(),
		ProcMask:         core.ProcMaskSpellDamage,
		Flags:            config.Flags & ^core.SpellFlagAPL | core.SpellFlagPassiveSpell,
		ClassSpellMask:   SpellMaskFlameShockDot,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Flame Shock",
			},
			NumberOfTicks:    int32(flameShockRank.Duration() / tick.Period()),
			TickLength:       tick.Period(),
			BonusCoefficient: tick.Coeff(),
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.Snapshot(target, tick.Average(core.CharacterLevel))
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, periodicTickOutcome(flameShockRank, dot)) // client: Periodic Can Crit
			},
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Dot(target).Apply(sim)
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			dot := spell.Dot(target)
			if useSnapshot {
				result := dot.CalcSnapshotDamage(sim, target, periodicTickOutcome(flameShockRank, dot))
				result.Damage /= dot.TickPeriod().Seconds()
				return result
			} else {
				result := spell.CalcPeriodicDamage(sim, target, tick.Average(core.CharacterLevel), spell.OutcomeExpectedMagicCrit)
				result.Damage /= dot.CalcTickPeriod().Round(time.Millisecond).Seconds()
				return result
			}
		},
	})
}

func (shaman *Shaman) registerFrostShockSpell(shockTimer *core.Timer) {
	config := shaman.newShockSpellConfig(frostShockRank, core.SpellSchoolFrost, shockTimer)
	config.ClassSpellMask = SpellMaskFrostShock
	config.Flags |= core.SpellFlagBinary
	config.ThreatMultiplier *= 2
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := frostShockRank.DamageEffect().Roll(sim, core.CharacterLevel)
		spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
	}

	shaman.FrostShock = shaman.RegisterSpell(config)
}

func (shaman *Shaman) registerShocks() {
	shockTimer := shaman.NewTimer()
	shaman.registerEarthShockSpell(shockTimer)
	shaman.registerFlameShockSpell(shockTimer)
	shaman.registerFrostShockSpell(shockTimer)
}
