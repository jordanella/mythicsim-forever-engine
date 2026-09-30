package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var ferociousBiteRank = spellData.FerociousBite.Highest()

// The client states the damage a point of excess Energy adds on the rank's second effect (270 at
// rank 5, in its own units) and the damage per combo point on the damage effect's points per
// resource (147 at rank 5). A bite rolls the damage effect's range on top: the Forever tooltip's
// 199-259 at one point is 82 +-30 plus 147. The attack power share is not in the client: 3% per
// combo point, the sim's Classic value.
var ferociousBiteDamagePerEnergy = spellData.FerociousBite.EffectAt(2).FractionAt(ferociousBiteRank.RankNumber())
var ferociousBiteDamagePerComboPoint = float64(ferociousBiteRank.DamageEffect().PointsPerResource)

const ferociousBiteAPPerComboPoint = 0.03

func (druid *Druid) registerFerociousBiteSpell() {
	druid.FerociousBite = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: ferociousBiteRank.ID},
		SpellSchool:    ferociousBiteRank.SpellSchool(),
		DefenseType:    ferociousBiteRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		ClassSpellMask: DruidSpellFerociousBite,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:           ferociousBiteRank.RankNumber(),

		EnergyCost: core.EnergyCostOptions{
			Cost:   int32(ferociousBiteRank.Cost()),
			Refund: ferociousBiteRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: ferociousBiteRank.GCD(),
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return druid.ComboPoints() > 0
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		MaxRange:         core.MaxMeleeRange,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			cp := float64(druid.ComboPoints())
			excessEnergy := druid.CurrentEnergy()
			baseDamage := ferociousBiteDamage(ferociousBiteRank.DamageEffect().Roll(sim, core.CharacterLevel), cp, spell.MeleeAttackPower(target)) + ferociousBiteDamagePerEnergy*excessEnergy

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				druid.SpendEnergy(sim, excessEnergy, spell.EnergyMetrics())
				druid.SpendComboPoints(sim, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},

		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			baseDamage := ferociousBiteDamage(ferociousBiteRank.DamageEffect().Average(core.CharacterLevel), float64(druid.ComboPoints()), spell.MeleeAttackPower(target))
			return spell.CalcDamage(sim, target, baseDamage, spell.OutcomeExpectedMeleeWeaponSpecialHitAndCrit)
		},
	})
}

func ferociousBiteDamage(base float64, comboPoints float64, attackPower float64) float64 {
	return base +
		ferociousBiteDamagePerComboPoint*comboPoints +
		ferociousBiteAPPerComboPoint*comboPoints*attackPower
}

func (druid *Druid) CurrentFerociousBiteCost() float64 {
	return druid.FerociousBite.Cost.GetCurrentCost()
}
