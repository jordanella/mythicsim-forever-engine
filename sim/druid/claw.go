package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var clawRank = spellData.Claw.Highest()

// Forever raises Claw to 110% weapon damage (Classic 100%), stated as E_WEAPON_PERCENT_DAMAGE on every
// rank; rank 5 deals (weapon + 115) x 110% for 45 Energy, the flat part scaled as Shred's is (client
// 9850, Wowhead env 16 agrees). Unlike Shred it works from the front, and it is the builder the level 20
// beta cats use most (foreverlogs 2686: 165 Claws).
var clawWeaponMultiplier = spellData.Claw.EffectAt(2).FractionAt(clawRank.RankNumber())

func (druid *Druid) registerClawSpell() {
	druid.Claw = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: clawRank.ID},
		SpellSchool:    clawRank.SpellSchool(),
		DefenseType:    clawRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		ClassSpellMask: DruidSpellClaw,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:           clawRank.RankNumber(),

		EnergyCost: core.EnergyCostOptions{
			Cost:   int32(clawRank.Cost()),
			Refund: clawRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: clawRank.GCD(),
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: clawWeaponMultiplier,
		ThreatMultiplier: 1,
		MaxRange:         core.MaxMeleeRange,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := clawRank.DamageEffect().Average(core.CharacterLevel) + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},

		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			baseDamage := clawRank.DamageEffect().Average(core.CharacterLevel) + spell.Unit.AutoAttacks.MH().CalculateAverageWeaponDamage(spell.MeleeAttackPower(target))
			return spell.CalcDamage(sim, target, baseDamage, spell.OutcomeExpectedMeleeWeaponSpecialHitAndCrit)
		},
	})
}
