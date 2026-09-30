package warrior

import (
	"github.com/wowsims/forever/sim/core"
)

func (warrior *Warrior) registerRend() {
	rendRank := spellData.Rend.Highest()

	tick := rendRank.PeriodicEffect()
	// The client row carries no attack power share, but Forever adds one: rank 3 (9 a tick) lands
	// 13-15 for 5 level 20 warriors in beta logs once Defensive Stance's -10% is taken out, 0.018-0.020
	// of attack power a tick over the base, read at each tick (Battle Shout gained mid-bleed counts).
	// Two-handers with Arms points land the same share times Improved Rend's 1.35.
	const apPerTick = 0.02

	warrior.Rend = warrior.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rendRank.ID},
		SpellSchool:    rendRank.SpellSchool(),
		DefenseType:    rendRank.DefenseTypeCore(),
		ClassSpellMask: SpellMaskRend,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,

		RageCost: core.RageCostOptions{
			Cost:   int32(rendRank.Cost()),
			Refund: rendRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rendRank.GCD(),
			},
			IgnoreHaste: true,
		},

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.StanceMatches(BattleStance | DefensiveStance)
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rend",
			},
			NumberOfTicks: int32(rendRank.Duration() / tick.Period()),
			TickLength:    tick.Period(),
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.Spell.CalcAndDealPeriodicDamage(sim, target, tick.Average(core.CharacterLevel)+apPerTick*dot.Spell.MeleeAttackPower(target), rendRank.TickOutcome(dot))
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
