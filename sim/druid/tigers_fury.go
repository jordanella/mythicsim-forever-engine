package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

var tigersFuryRank = spellData.TigersFury.Highest()

// Forever pays a share of Physical damage rather than Classic's flat amount, so it scales with the
// cat's weapon and attack power instead of fading as gear improves: the client states 15% on the
// rank's dummy effect, with no Energy cost and a 30 second cooldown.
func (druid *Druid) registerTigersFurySpell() {
	actionID := core.ActionID{SpellID: tigersFuryRank.ID}
	multiplier := 1 + tigersFuryRank.Effect(dbcenums.A_DUMMY, 0).BaseValue()/100

	// King of the Jungle: Tiger's Fury instantly grants 20 Energy a rank.
	energyGain := spellData.KingOfTheJungle.EffectAt(1).ValueAt(druid.Talents.KingOfTheJungle)
	// Tagged so it does not collide with the metrics the cost would register under the same action.
	energyMetrics := druid.NewEnergyMetrics(actionID.WithTag(1))

	druid.TigersFuryAura = druid.RegisterAura(core.Aura{
		Label:    "Tiger's Fury",
		ActionID: actionID,
		Duration: tigersFuryRank.Duration(),
	}).AttachMultiplicativePseudoStatBuff(&druid.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical], multiplier)

	druid.TigersFury = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: DruidSpellTigersFury,
		Flags:          core.SpellFlagAPL,

		EnergyCost: core.EnergyCostOptions{
			Cost: int32(tigersFuryRank.Cost()),
		},
		Cast: core.CastConfig{
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: max(tigersFuryRank.Cooldown(), tigersFuryRank.CategoryCooldown()),
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			if gain := energyGain + druid.WolfsheadTigersFuryEnergy; gain > 0 {
				druid.AddEnergy(sim, gain, energyMetrics)
			}
			druid.TigersFuryAura.Activate(sim)
		},

		RelatedSelfBuff: druid.TigersFuryAura,
	})
}
