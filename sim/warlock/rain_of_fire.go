package warlock

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

// Forever's Rain of Fire is an area trigger, like its Blizzard: an 8 second channel whose periodic
// dummy casts a Fire damage spell (1282385 at rank 4) on every enemy in the area every 2 seconds.
// The damage spell is a direct hit, so it rolls crit unless the client marks it as unable to.
func (warlock *Warlock) registerRainOfFire() {
	rank := spellData.RainOfFire.Highest()
	tickSpell := spellData.RainOfFireTriggered.Rank(rank.RankNumber())
	tick := tickSpell.DamageEffect()
	tickLength := rank.Effect(dbcenums.A_PERIODIC_DUMMY, 0).Period()
	actionID := core.ActionID{SpellID: rank.ID}

	tickCast := warlock.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: tickSpell.ID},
		SpellSchool:    tickSpell.SpellSchool(),
		DefenseType:    tickSpell.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagNoOnCastComplete,
		ClassSpellMask: WarlockSpellRainOfFire,

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		BonusCoefficient:         tick.Coeff(),
		ThreatMultiplier:         1,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			outcome := spell.OutcomeMagicHitAndCrit
			if tickSpell.CannotCrit() {
				outcome = spell.OutcomeMagicHit
			}
			spell.CalcAndDealAoeDamage(sim, tick.Average(core.CharacterLevel), outcome)
		},
	})

	warlock.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagChanneled | core.SpellFlagAPL,
		ClassSpellMask: WarlockSpellRainOfFire,

		ManaCost: core.ManaCostOptions{FlatCost: int32(rank.Cost())},
		Cast:     core.CastConfig{DefaultCast: core.Cast{GCD: rank.GCD()}},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label:    "Rain of Fire",
				ActionID: actionID,
			},
			NumberOfTicks: int32(rank.Duration() / tickLength),
			TickLength:    tickLength,
			OnTick: func(sim *core.Simulation, target *core.Unit, _ *core.Dot) {
				tickCast.Cast(sim, target)
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			spell.AOEDot().Apply(sim)
		},
	})
}
