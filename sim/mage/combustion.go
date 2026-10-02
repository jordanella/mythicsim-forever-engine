package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

// Each fire hit adds a stack of crit until the row's charges of fire crits are spent: 3 since client
// 1.60.1.70170 ("Combustion has 3 charges again"), 4 in 70124.
func (mage *Mage) registerCombustionSpell() {
	if !mage.Talents.Combustion {
		return
	}

	combustionRank := spellData.Combustion.Highest()
	critPerStack := spellData.CombustionTriggered.Highest().Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CRITICAL_CHANCE)).Average(core.CharacterLevel)
	maxCrits := int32(combustionRank.ProcCharges)

	actionID := core.ActionID{SpellID: combustionRank.ID}
	cd := core.Cooldown{
		Timer:    mage.NewTimer(),
		Duration: max(combustionRank.Cooldown(), combustionRank.CategoryCooldown()),
	}

	critMod := mage.AddDynamicMod(core.SpellModConfig{
		ClassMask: MageSpellsAll,
		School:    core.SpellSchoolFire,
		Kind:      core.SpellMod_BonusCrit_Percent,
	})

	numCrits := int32(0)
	combustAura := mage.RegisterAura(core.Aura{
		Label:     "Combustion",
		ActionID:  actionID,
		Duration:  core.NeverExpires,
		MaxStacks: int32(spellData.CombustionTriggered.Highest().MaxStack), // 10 on 28682
		OnGain: func(_ *core.Aura, _ *core.Simulation) {
			numCrits = 0
			critMod.Activate()
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			critMod.Deactivate()
			cd.Use(sim)
			mage.UpdateMajorCooldowns()
		},
		OnStacksChange: func(_ *core.Aura, _ *core.Simulation, _ int32, newStacks int32) {
			critMod.UpdateFloatValue(critPerStack * float64(newStacks))
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// Ignite's ticks are not a cast of their own and never spend a charge.
			if !result.Landed() || !spell.Matches(MageSpellsAllDamaging) || !spell.SpellSchool.Matches(core.SpellSchoolFire) {
				return
			}

			aura.AddStack(sim)
			if result.DidCrit() {
				numCrits++
				if numCrits >= maxCrits {
					aura.Deactivate(sim)
				}
			}
		},
	})

	combustSpell := mage.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		Flags:          core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,
		ClassSpellMask: MageSpellCombustion,
		Cast: core.CastConfig{
			CD: cd,
		},
		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return !combustAura.IsActive()
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			combustAura.Activate(sim)
			combustAura.AddStack(sim)
		},
		RelatedSelfBuff: combustAura,
	})

	mage.AddMajorCooldown(core.MajorCooldown{
		Spell: combustSpell,
		Type:  core.CooldownTypeDPS,
	})
}
