package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

var prowlRank = spellData.Prowl.Highest()

func (druid *Druid) registerProwlSpell() {
	actionID := core.ActionID{SpellID: prowlRank.ID}
	movementSpeedMultiplier := 1 + prowlRank.Effect(dbcenums.A_MOD_DECREASE_SPEED, 0).BaseValue()/100

	icd := core.Cooldown{
		Timer:    druid.NewTimer(),
		Duration: max(prowlRank.Cooldown(), prowlRank.CategoryCooldown()),
	}

	druid.ProwlAura = druid.RegisterAura(core.Aura{
		Label:    "Prowl",
		ActionID: actionID,
		Duration: core.NeverExpires,

		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyMovementSpeed(sim, movementSpeedMultiplier)

			// No white swing goes out from Prowl; the opener starts it. The sim's swing timer runs from the
			// pull anyway and would win a tie with the opener, so the rotation gets to act before each swing
			// the way a queued Heroic Strike does.
			druid.AutoAttacks.SetReplaceMHSwing(func(_ *core.Simulation, swing *core.Spell) *core.Spell {
				return swing
			})
		},

		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			aura.Deactivate(sim)
		},

		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			icd.Use(sim)
			aura.Unit.MultiplyMovementSpeed(sim, 1.0/movementSpeedMultiplier)
			druid.AutoAttacks.SetReplaceMHSwing(nil)
		},
	})

	druid.CatFormAura.ApplyOnExpire(func(_ *core.Aura, sim *core.Simulation) {
		if druid.ProwlAura.IsActive() {
			druid.ProwlAura.Deactivate(sim)
		}
	})

	druid.Prowl = druid.RegisterSpell(Any, core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: prowlRank.SpellSchool(),
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagAPL,

		Cast: core.CastConfig{
			CD: icd,
		},

		ExtraCastCondition: func(sim *core.Simulation, _ *core.Unit) bool {
			return sim.CurrentTime < 0
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			if !druid.InForm(Cat) {
				druid.CatFormAura.Activate(sim)
			}

			druid.ProwlAura.Activate(sim)
		},
	})
}
