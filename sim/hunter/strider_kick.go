package hunter

import (
	"github.com/wowsims/forever/sim/core"
)

// Strider Kick from the beta client (1317257): 100% normalized melee weapon damage, 8 sec cooldown,
// 5.81% of base mana.
func (hunter *Hunter) registerStriderKickSpell() {
	if !hunter.Talents.StriderKick {
		return
	}

	rank := spellData.StriderKick.Highest()

	hunter.StriderKick = hunter.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ClassSpellMask: HunterSpellStriderKick,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		MaxRange:       float64(rank.MaxRange),

		ManaCost: rank.ManaCost(),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := hunter.AutoAttacks.MH().CalculateNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
		},
	})
}
