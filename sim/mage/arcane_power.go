package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

func (mage *Mage) registerArcanePowerSpell() {
	if !mage.Talents.ArcanePower {
		return
	}

	arcanePowerRank := spellData.ArcanePower.Highest()
	actionID := core.ActionID{SpellID: arcanePowerRank.ID}
	damage := arcanePowerRank.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).Average(core.CharacterLevel) / 100

	mage.ArcanePowerAura = mage.RegisterAura(core.Aura{
		Label:    "Arcane Power",
		ActionID: actionID,
		Duration: arcanePowerRank.Duration(),
	}).AttachSpellMod(core.SpellModConfig{
		ClassMask:  MageSpellsAll &^ MageSpellFrostfireBolt,
		FloatValue: damage,
		Kind:       core.SpellMod_DamageDone_Flat,
	}).AttachSpellMod(core.SpellModConfig{
		// The SPELLMOD_DOT mask leaves out Frostfire Bolt: its hit takes the bonus, its DoT does not.
		ClassMask:  MageSpellFrostfireBolt,
		FloatValue: damage,
		Kind:       core.SpellMod_DirectDamageDone_Flat,
	}).AttachSpellMod(core.SpellModConfig{
		ClassMask:  MageSpellsAll,
		FloatValue: arcanePowerRank.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).Average(core.CharacterLevel) / 100,
		Kind:       core.SpellMod_PowerCost_Pct_Add,
	})

	spell := mage.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		Flags:          core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,
		ClassSpellMask: MageSpellArcanePower,
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: max(arcanePowerRank.Cooldown(), arcanePowerRank.CategoryCooldown()),
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			mage.ArcanePowerAura.Activate(sim)
		},
		RelatedSelfBuff: mage.ArcanePowerAura,
	})

	mage.AddMajorCooldown(core.MajorCooldown{
		Spell: spell,
		Type:  core.CooldownTypeDPS,
	})
}
