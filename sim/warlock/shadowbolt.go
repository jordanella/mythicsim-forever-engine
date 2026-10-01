package warlock

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Every rank is registered so a rotation can drop to a cheaper one when mana runs short; the
// talents and procs reach them all through WarlockSpellShadowBolt. warlock.ShadowBolt stays the
// highest rank.
func (warlock *Warlock) registerShadowBolt() {
	highest := spellData.ShadowBolt.Highest()
	spellData.ShadowBolt.Each(func(_ int32, rank *spelldata.Spell) {
		spell := warlock.registerShadowBoltRank(rank)
		if rank == highest {
			warlock.ShadowBolt = spell
		}
	})
}

func (warlock *Warlock) registerShadowBoltRank(rank *spelldata.Spell) *core.Spell {
	return warlock.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: WarlockSpellShadowBolt,
		Rank:           rank.RankNumber(),
		MissileSpeed:   float64(rank.Speed),

		ManaCost: core.ManaCostOptions{FlatCost: int32(rank.Cost())},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      rank.GCD(),
				CastTime: rank.CastTime(),
			},
		},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         rank.DamageEffect().Coeff(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, rank.DamageEffect().Roll(sim, core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
