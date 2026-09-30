package paladin

import (
	"github.com/wowsims/forever/sim/core"
)

// Judgement
// https://www.wowhead.com/forever/spell=20271
//
// Unleash the energy of a Seal spell upon an enemy. Does not consume the Seal. Refer to individual
// Seals for Judgement effect.
//
// The spell itself has no defense type and rolls nothing: the seal's own judgement spell is Melee in
// SpellCategories and carries the hit roll along with the effect.
func (paladin *Paladin) registerJudgement() {
	rank := spellData.Judgement.Highest()

	// No SpellFlagNoOnCastComplete: Sanctified Judgement and Swift Judgement listen for the cast
	// through OnCastComplete, and the flag silences both.
	paladin.Judgement = paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL | core.SpellFlagPassiveSpell,
		ClassSpellMask: SpellMaskJudgement,
		MaxRange:       float64(rank.MaxRange),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ManaCost: rank.ManaCost(),
		Cast: core.CastConfig{
			// Off the global cooldown, as the client states.
			DefaultCast: core.Cast{
				NonEmpty: true,
			},
			CD: core.Cooldown{
				Timer:    paladin.sharedTimer(&paladin.judgementTimer),
				Duration: cooldown(rank),
			},
		},

		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return paladin.activeSeal() != nil
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			paladin.activeSeal().judgement.Cast(sim, target)

			// Let the rotation react when the cooldown ends, since nothing on the GCD marks it.
			pa := sim.GetConsumedPendingActionFromPool()
			pa.NextActionAt = sim.CurrentTime + spell.TimeToReady(sim) + core.SpellBatchWindow
			pa.OnAction = func(sim *core.Simulation) {
				paladin.ReactToEvent(sim, false, false)
			}
			sim.AddPendingAction(pa)
		},
	})

	// Every melee strike that lands refreshes the judgement debuffs on its target.
	paladin.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Judgement Refresh" + paladin.Label,
		Callback:           core.CallbackOnSpellHitDealt,
		ProcMask:           core.ProcMaskMelee,
		Outcome:            core.OutcomeLanded,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			paladin.refreshJudgements(sim, result.Target)
		},
	})
}
