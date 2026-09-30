package druid

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

var enrageRank = spellData.Enrage.Highest()

// Enrage: 2 Rage a second for 10 sec, and Forever adds an instant 10 Rage up front (the client's
// E_ENERGIZE effect, stated as 100 in its own units). Base armor is cut by 27% while it lasts,
// which the client does not carry.
func (druid *Druid) registerEnrageSpell() {
	actionID := core.ActionID{SpellID: enrageRank.ID}
	rageMetrics := druid.NewRageMetrics(actionID)

	const armorMultiplier = 1 - 0.27

	// Energize is the periodic half; the instant 10 is effect 1 (E_ENERGIZE 100).
	instantRage := enrageRank.Effect(dbcenums.A_NONE, 1).Tenths()
	ragePerTick := enrageRank.Effect(dbcenums.A_PERIODIC_ENERGIZE, 1).BaseValue() / 10
	numTicks := int(enrageRank.Duration() / time.Second)

	druid.EnrageAura = druid.RegisterAura(core.Aura{
		Label:    "Enrage",
		ActionID: actionID,
		Duration: enrageRank.Duration(),
		OnGain: func(_ *core.Aura, sim *core.Simulation) {
			druid.ApplyDynamicEquipScaling(sim, stats.Armor, armorMultiplier)
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			druid.RemoveDynamicEquipScaling(sim, stats.Armor, armorMultiplier)
		},
	})

	druid.Enrage = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: DruidSpellEnrage,
		Flags:          core.SpellFlagAPL,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: max(enrageRank.Cooldown(), enrageRank.CategoryCooldown()),
			},
			IgnoreHaste: true,
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			druid.AddRage(sim, instantRage+druid.IntensityEnrageRageBonus+druid.WolfsheadEnrageRage, rageMetrics)
			druid.EnrageAura.Activate(sim)

			core.StartPeriodicAction(sim, core.PeriodicActionOptions{
				Period:   time.Second,
				NumTicks: numTicks,
				Priority: core.ActionPriorityRegen,
				OnAction: func(sim *core.Simulation) {
					if druid.EnrageAura.IsActive() {
						druid.AddRage(sim, ragePerTick, rageMetrics)
					}
				},
			})
		},

		RelatedSelfBuff: druid.EnrageAura,
	})

	druid.AddMajorCooldown(core.MajorCooldown{
		Spell: druid.Enrage.Spell,
		Type:  core.CooldownTypeDPS,
	})
}
