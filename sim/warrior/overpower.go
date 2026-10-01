package warrior

import (
	"github.com/wowsims/forever/sim/core"
)

func (warrior *Warrior) registerOverpower() {
	overpowerRank := spellData.Overpower.ByID(11585)
	overpowerBaseDamage := overpowerRank.DamageEffect().Average(core.CharacterLevel)
	// The window a dodge opens: the aura Offensive State (DND) fires on the hit.
	overpowerWindow := spellData.OffensiveStateTriggered.Highest()

	actionID := core.ActionID{SpellID: overpowerRank.ID}
	overpowerCD := cooldownOf(overpowerRank)

	// A dodge triggers 1282733, which energizes 1 of power type 4 (the combo point pool); that point is
	// Overpower's second cost. The window stands in for it: one cast per dodge, stacked charges not modelled.
	warrior.OverpowerAura = warrior.RegisterAura(core.Aura{
		ActionID: core.ActionID{SpellID: overpowerWindow.ID},
		Label:    "Overpower Aura",
		Duration: overpowerWindow.Duration(),
	})

	warrior.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Overpower - Trigger",
		TriggerImmediately: true,
		Outcome:            core.OutcomeDodge,
		Callback:           core.CallbackOnSpellHitDealt,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			warrior.OverpowerAura.Activate(sim)
		},
	})

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		ClassSpellMask: SpellMaskOverpower,
		MaxRange:       core.MaxMeleeRange,

		RageCost: core.RageCostOptions{
			Cost:   int32(overpowerRank.Cost()),
			Refund: overpowerRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: overpowerRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: overpowerCD,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		// Not in the client table; our Classic value until measured in game.
		ThreatMultiplier: 0.75,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.StanceMatches(BattleStance) && warrior.OverpowerAura.IsActive()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := overpowerBaseDamage + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialNoBlockDodgeParry)
			warrior.OverpowerAura.Deactivate(sim)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
