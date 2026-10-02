package warrior

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

func (warrior *Warrior) registerDemoralizingShout() {
	// TODO: Ingame research needed if this adds flat threat
	demoralizingShoutRank := spellData.DemoralizingShout.Highest()

	warrior.DemoralizingShoutAuras = warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		// Forever has no Improved Demoralizing Shout and Booming Voice widens the radius and cuts the
		// cost only, so the aura is the client's attack power reduction for 45 seconds.
		return buffs.DemoralizingShoutAura(target, true, 0)
	})

	warrior.DemoralizingShout = warrior.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: demoralizingShoutRank.ID},
		SpellSchool:    demoralizingShoutRank.SpellSchool(),
		DefenseType:    demoralizingShoutRank.DefenseTypeCore(),
		ClassSpellMask: SpellMaskDemoralizingShout,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL,

		RageCost: core.RageCostOptions{
			Cost: int32(demoralizingShoutRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: demoralizingShoutRank.GCD(),
			},
			IgnoreHaste: true,
		},

		// Not in the client table; our Classic value until measured in game.
		ThreatMultiplier: 0.4,
		FlatThreatBonus:  0.4 * 2 * 54,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.ActiveTargetUnits {
				result := spell.CalcAndDealOutcome(sim, aoeTarget, spell.OutcomeMagicHit)
				if result.Landed() {
					warrior.DemoralizingShoutAuras.Get(aoeTarget).Activate(sim)
				}
			}
		},

		RelatedAuraArrays: warrior.DemoralizingShoutAuras.ToMap(),
	})
}
