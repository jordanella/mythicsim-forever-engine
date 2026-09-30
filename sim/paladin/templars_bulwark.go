package paladin

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

// Templar's Bulwark (talent)
// https://www.wowhead.com/forever/spell=1311015
//
// When activated, this ability grants you an absorb shield equal to 100% of your maximum health
// for 8 sec. Applies Forbearance for 1 min. Cannot be cast while Forbearance is active.
func (paladin *Paladin) registerTemplarsBulwark() {
	rank := spellData.TemplarsBulwark.Highest()
	actionID := core.ActionID{SpellID: rank.ID}
	healthShare := rank.Effect(dbcenums.A_SCHOOL_ABSORB, 127).Percent()

	shield := paladin.NewDamageAbsorptionAura(core.AbsorptionAuraConfig{
		Aura: core.Aura{
			Label:    "Templar's Bulwark" + paladin.Label,
			ActionID: actionID,
			Duration: rank.Duration(),
		},
		ShieldStrengthCalculator: func(unit *core.Unit) float64 {
			return unit.MaxHealth() * healthShare
		},
	})

	spell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ClassSpellMask: SpellMaskTemplarsBulwark,

		ManaCost: rank.ManaCost(),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				NonEmpty: true,
			},
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: cooldown(rank),
			},
		},

		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return !paladin.Forbearance.IsActive()
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			shield.Activate(sim)
			paladin.Forbearance.Activate(sim)
		},

		RelatedSelfBuff: shield.Aura,
	})

	paladin.AddMajorCooldown(core.MajorCooldown{
		Spell: spell,
		Type:  core.CooldownTypeSurvival,
	})
}
