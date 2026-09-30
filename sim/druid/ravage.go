package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var ravageRank = spellData.Ravage.Highest()

// Client 70009 keeps Classic Era's Ravage: 350% weapon damage on every rank (E_WEAPON_PERCENT_DAMAGE)
// plus the rank's flat 98, which the multiplier scales too (the tooltip's "plus 343"), 60 Energy,
// from Prowl and behind the target. Shred's 155% reads the same way (shred.go).
var ravageWeaponMultiplier = spellData.Ravage.EffectAt(2).FractionAt(ravageRank.RankNumber())

func (druid *Druid) registerRavageSpell() {
	druid.Ravage = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: ravageRank.ID},
		SpellSchool:    ravageRank.SpellSchool(),
		DefenseType:    ravageRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		ClassSpellMask: DruidSpellRavage,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:           ravageRank.RankNumber(),

		EnergyCost: core.EnergyCostOptions{
			Cost:   int32(ravageRank.Cost()),
			Refund: ravageRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: ravageRank.GCD(),
			},
			IgnoreHaste: true,
		},

		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return druid.ProwlAura.IsActive() && !druid.PseudoStats.InFrontOfTarget && !druid.CannotShredTarget
		},

		DamageMultiplier: ravageWeaponMultiplier,
		ThreatMultiplier: 1,
		MaxRange:         core.MaxMeleeRange,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := ravageRank.DamageEffect().Average(core.CharacterLevel) + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},

		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			baseDamage := ravageRank.DamageEffect().Average(core.CharacterLevel) + spell.Unit.AutoAttacks.MH().CalculateAverageWeaponDamage(spell.MeleeAttackPower(target))
			return spell.CalcDamage(sim, target, baseDamage, spell.OutcomeExpectedMeleeWeaponSpecialHitAndCrit)
		},
	})
}
