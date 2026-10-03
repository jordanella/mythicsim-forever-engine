package rogue

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

var sliceAndDiceRank = spellData.SliceAndDice.ByID(6774)

func (rogue *Rogue) registerSliceAndDice() {
	actionID := core.ActionID{SpellID: sliceAndDiceRank.ID}

	// The client states the attack speed bonus as a percentage on the rank's own effect (30).
	rogue.SliceAndDiceBonusFlat = sliceAndDiceRank.Effect(dbcenums.A_MOD_MELEE_HASTE_3, 0).Average(core.CharacterLevel) / 100

	var sliceAndDiceMod float64
	rogue.SliceAndDiceAura = rogue.RegisterAura(core.Aura{
		Label:    "Slice and Dice",
		ActionID: actionID,
		// This will be overridden on cast, but set a non-zero default so it doesn't crash when used in APL prepull
		Duration: rogue.sliceAndDiceDurations[5],
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			sliceAndDiceMod = 1 + rogue.SliceAndDiceBonusFlat
			rogue.MultiplyMeleeSpeed(sim, sliceAndDiceMod)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, 1/sliceAndDiceMod)
		},
	})

	// The cast sets the aura's duration from its combo points. Restore the default every
	// iteration, or an aura activated without the cast (an APL Activate Aura) inherits whatever
	// the previous iteration last cast.
	defaultSliceAndDiceDuration := rogue.SliceAndDiceAura.Duration
	rogue.RegisterResetEffect(func(_ *core.Simulation) {
		rogue.SliceAndDiceAura.Duration = defaultSliceAndDiceDuration
	})

	rogue.SliceAndDice = rogue.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		Flags:          SpellFlagFinisher | core.SpellFlagAPL,
		MetricSplits:   6,
		ClassSpellMask: RogueSpellSliceAndDice,

		EnergyCost: core.EnergyCostOptions{
			Cost: int32(sliceAndDiceRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: sliceAndDiceRank.GCD(),
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(rogue.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			comboPoints := rogue.ComboPoints()
			rogue.ApplyFinisher(sim, spell)
			spell.RelatedSelfBuff.Deactivate(sim)
			spell.RelatedSelfBuff.Duration = rogue.getSliceDuration(comboPoints)
			spell.RelatedSelfBuff.Activate(sim)
		},

		RelatedSelfBuff: rogue.SliceAndDiceAura,
	})
}

func (rogue *Rogue) getSliceDuration(comboPoints int32) time.Duration {
	duration := rogue.sliceAndDiceDurations[comboPoints]
	return time.Duration(float64(duration+rogue.SliceAndDiceBonusDuration) * spellData.ImprovedSliceAndDice.MultiplierAt(rogue.Talents.ImprovedSliceAndDice))
}
