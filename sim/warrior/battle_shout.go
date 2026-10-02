package warrior

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

// How close to running out the warrior's own shout has to be before recasting it is worth a global.
const ShoutExpirationThreshold = time.Second * 3

func (warrior *Warrior) registerBattleShout() {
	battleShoutRank := spellData.BattleShout.Highest()

	// A warrior that shouts builds a copy of its own; one that shouts nothing gets the
	// isPlayer=false constructor, whose aura is the party's external copy.
	castsOwnShout := warrior.UseBattleShout

	// Three pieces of Battlegear of Wrath add a flat 30 to the shout this warrior makes, both to
	// what its copy applies and to what it bids for the category.
	battleShoutBase := buffs.BattleShoutValue(0)
	battleShoutValue := battleShoutBase
	shoutsWithTheSet := castsOwnShout && warrior.HasBsT2
	if shoutsWithTheSet {
		battleShoutValue += buffs.BattleShoutT2Bonus
	}

	auras := warrior.NewAllyAuraArray(func(unit *core.Unit) *core.Aura {
		// The party's Battle Shout registers the external copy before this runs, and that copy
		// keeps the build phase it was registered with.
		partyShout := !castsOwnShout && unit.GetAuraByID(core.ActionID{SpellID: battleShoutRank.ID}.WithTag(-1)) != nil

		// Booming Voice widens the radius and cuts the cost (registerBoomingVoice), so the aura takes no talent points.
		aura := buffs.BattleShoutAura(unit, castsOwnShout, 0)
		if shoutsWithTheSet {
			core.AddGeneratedFlatBonus(aura, stats.AttackPower, battleShoutBase, buffs.BattleShoutT2Bonus)
		}
		if !partyShout {
			aura.BuildPhase = core.Ternary(castsOwnShout, core.CharacterBuildPhaseBuffs, core.CharacterBuildPhaseNone)
		}
		return aura
	})
	battleShoutCategory := warrior.GetExclusiveEffectCategory(buffs.BattleShoutCategory)

	warrior.BattleShout = warrior.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: battleShoutRank.ID},
		ClassSpellMask: SpellMaskBattleShout,
		SpellSchool:    battleShoutRank.SpellSchool(),
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ProcMask:       core.ProcMaskEmpty,

		RageCost: core.RageCostOptions{
			Cost: int32(battleShoutRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: battleShoutRank.GCD(),
			},
			IgnoreHaste: true,
		},

		ThreatMultiplier: 1,
		// TODO: Manual review needed -- spell 25289 carries no threat effect; none is modelled until
		// measured in game.
		FlatThreatBonus: battleShoutRank.FindEffect(dbcenums.E_THREAT, 0, 0).Average(core.CharacterLevel),

		// Battle Shout is a single-aura exclusive category. Nothing there and the cast puts the buff
		// up; this warrior's own copy there is only worth refreshing as it runs out; anything else
		// (the party buff) has to be outbid first, or casting would only burn rage every GCD.
		ExtraCastCondition: func(sim *core.Simulation, _ *core.Unit) bool {
			active := battleShoutCategory.GetActiveEffect()
			if active == nil {
				return true
			}
			if active.Aura == auras.Get(&warrior.Unit) {
				return active.Aura.RemainingDuration(sim) <= ShoutExpirationThreshold
			}
			return battleShoutValue >= active.Priority
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
			auras.ActivateAllPlayers(sim)
		},

		RelatedAuraArrays: auras.ToMap(),
	})
}
