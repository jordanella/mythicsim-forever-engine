package warlock

import (
	"github.com/wowsims/forever/sim/core"
)

// Conflagrate burns the Immolate on the target. Shadow and Flame's second effect (20% per point)
// is the chance it survives, so at 5/5 Conflagrate stops consuming Immolate altogether.
func (warlock *Warlock) registerConflagrate() {
	rank := spellData.Conflagrate.Highest()
	keepImmolateChance := spellData.ShadowAndFlame.EffectAt(2).FractionAt(warlock.Talents.ShadowAndFlame)

	warlock.Conflagrate = warlock.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: WarlockSpellConflagrate,

		ManaCost: core.ManaCostOptions{FlatCost: int32(rank.Cost())},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warlock.Immolate.Dot(target).IsActive()
		},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         rank.DamageEffect().Coeff(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, rank.DamageEffect().Roll(sim, core.CharacterLevel), spell.OutcomeMagicHitAndCrit)

			dot := warlock.Immolate.Dot(target)
			if dot.IsActive() && !sim.Proc(keepImmolateChance, "Shadow and Flame") {
				dot.Deactivate(sim)
			}
		},
	})
}
