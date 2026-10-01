package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Every rank is registered: the fire rotation drops to rank 1 when mana runs short.
func (mage *Mage) registerScorchSpell() {
	mage.registerImprovedScorch()
	spellData.Scorch.Each(func(_ int32, rank *spelldata.Spell) { mage.registerScorchRank(rank) })
}

func (mage *Mage) registerScorchRank(scorchRank *spelldata.Spell) {
	procChance := spellData.ImprovedScorch.FractionAt(mage.Talents.ImprovedScorch)

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: scorchRank.ID},
		SpellSchool:    scorchRank.SpellSchool(),
		DefenseType:    scorchRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: MageSpellScorch,
		Rank:           scorchRank.RankNumber(),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(scorchRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      scorchRank.GCD(),
				CastTime: scorchRank.CastTime(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: scorchRank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, scorchRank.DamageEffect().Roll(sim, core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			if result.Landed() && mage.ImprovedScorchAura != nil && sim.Proc(procChance, "Improved Scorch") {
				mage.ImprovedScorchAura.Activate(sim)
				mage.ImprovedScorchAura.AddStack(sim)
			}
		},
	})
}

// In Forever the Fire Vulnerability Improved Scorch stacks only raises the fire damage of the mage
// who applied it (beta client 1.60.1), where Classic and TBC made it a raid debuff.
func (mage *Mage) registerImprovedScorch() {
	if mage.Talents.ImprovedScorch == 0 {
		return
	}

	vulnerabilityRank := spellData.ImprovedScorchTriggered.Highest()
	damagePerStack := vulnerabilityRank.EffectN(1).Average(core.CharacterLevel) / 100

	damageMod := mage.AddDynamicMod(core.SpellModConfig{
		ClassMask: MageSpellsAll,
		School:    core.SpellSchoolFire,
		Kind:      core.SpellMod_DamageDone_Pct,
	})

	mage.ImprovedScorchAura = mage.RegisterAura(core.Aura{
		Label:     "Fire Vulnerability",
		ActionID:  core.ActionID{SpellID: vulnerabilityRank.ID},
		Duration:  vulnerabilityRank.Duration(),
		MaxStacks: 5,
		OnGain: func(_ *core.Aura, _ *core.Simulation) {
			damageMod.Activate()
		},
		OnExpire: func(_ *core.Aura, _ *core.Simulation) {
			damageMod.Deactivate()
		},
		OnStacksChange: func(_ *core.Aura, _ *core.Simulation, _ int32, newStacks int32) {
			damageMod.UpdateFloatValue(damagePerStack * float64(newStacks))
		},
	})
}
