package rogue

import (
	"time"

	"github.com/wowsims/forever/sim/core"
)

var kidneyShotRank = spellData.KidneyShot.Highest()

// Kidney Shot (client 8643): 25 Energy, 20 sec cooldown, stuns for 1 sec plus 1 sec a combo point.
// Its third effect is damage taken from the rogue, all schools; it is 0 until Improved Kidney Shot
// (14174, new in Forever) raises it to 5/10%. The stun also stops the target dodging and parrying.
// No default rotation casts it: it spends the points Eviscerate would.
func (rogue *Rogue) registerKidneyShot() {
	actionID := core.ActionID{SpellID: kidneyShotRank.ID}
	damageTaken := 1 + spellData.ImprovedKidneyShot.ValueAt(rogue.Talents.ImprovedKidneyShot)/100

	// Set on each cast; the stun reads it on gain. A non-zero start keeps registration happy.
	stunDuration := kidneyShotRank.Duration()
	stunAuras := rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		aura := target.RegisterVariableStunAura("Kidney Shot - "+rogue.Label, actionID, func() time.Duration {
			return stunDuration
		})
		if damageTaken == 1 {
			return aura
		}
		return aura.ApplyOnGain(func(aura *core.Aura, _ *core.Simulation) {
			rogue.AttackTables[aura.Unit.UnitIndex].DamageTakenMultiplier *= damageTaken
		}).ApplyOnExpire(func(aura *core.Aura, _ *core.Simulation) {
			rogue.AttackTables[aura.Unit.UnitIndex].DamageTakenMultiplier /= damageTaken
		})
	})

	rogue.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    kidneyShotRank.SpellSchool(),
		DefenseType:    kidneyShotRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | SpellFlagFinisher | core.SpellFlagAPL,
		MetricSplits:   6,
		ClassSpellMask: RogueSpellKidneyShot,
		MaxRange:       core.MaxMeleeRange,

		EnergyCost: core.EnergyCostOptions{
			Cost:          int32(kidneyShotRank.Cost()),
			Refund:        kidneyShotRank.MissRefund(),
			RefundMetrics: rogue.EnergyRefundMetrics,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: kidneyShotRank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: max(kidneyShotRank.Cooldown(), kidneyShotRank.CategoryCooldown()),
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(rogue.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if result.Landed() {
				stunDuration = kidneyShotRank.Duration() + time.Second*time.Duration(rogue.ComboPoints())
				core.ApplyStun(sim, stunAuras.Get(target))
				rogue.ApplyFinisher(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},

		RelatedAuraArrays: stunAuras.ToMap(),
	})
}
