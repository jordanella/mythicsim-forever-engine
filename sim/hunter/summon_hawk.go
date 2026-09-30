package hunter

import (
	"strconv"
	"time"

	"github.com/wowsims/forever/sim/core"
)

// Summon Hawk shares its cooldown with Arcane Shot, and Ferocity and Unleashed Fury buff hawks the
// same way they buff pets (talents_beast_mastery.go).
//
// The beta client (1293241, 1293525-1293527) gives the dive bomb, 32/47/85/108 plus 5% of ranged
// attack power, the mana cost, a 6 sec cooldown and the 18 sec hawk (1293248), and caps the hawks
// out at once at its third effect, 2. The hawk that stays is a guardian whose swings the client does
// not describe, so each hawk's assault is modelled as the rank's dive bomb base damage every 3 sec,
// a melee hit that can crit. Every dive bomb rank carries the client's always-hit attribute, so it
// never misses, and is never dodged or parried, and always leaves a hawk. A cast past the cap
// replaces the hawk closest to leaving.
func (hunter *Hunter) registerSummonHawkSpell(timer *core.Timer) {
	if !hunter.Talents.SummonHawk {
		return
	}

	rank := spellData.SummonHawk.Highest()
	baseDamage := rank.DamageEffect().Average(core.CharacterLevel)
	hawkDuration := spellData.SummonHawkTriggered.ByID(1293248).Duration()
	const swingInterval = time.Second * 3

	hawks := make([]*core.Spell, int(rank.EffectN(3).BasePoints))
	for i := range hawks {
		hawks[i] = hunter.RegisterSpell(core.SpellConfig{
			ActionID:       core.ActionID{SpellID: rank.ID, Tag: int32(i + 1)},
			SpellSchool:    rank.SpellSchool(),
			DefenseType:    core.DefenseTypeMelee,
			ClassSpellMask: HunterSpellSummonHawk,
			ProcMask:       core.ProcMaskEmpty,
			Flags:          core.SpellFlagMeleeMetrics,

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			Dot: core.DotConfig{
				Aura: core.Aura{
					Label: "Summon Hawk " + strconv.Itoa(i+1) + hunter.Label,
				},
				NumberOfTicks: int32(hawkDuration / swingInterval),
				TickLength:    swingInterval,

				OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.Snapshot(target, baseDamage)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.Spell.OutcomeTickPhysicalCrit)
				},
			},
		})
	}

	hunter.SummonHawk = hunter.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    core.DefenseTypeMelee,
		ClassSpellMask: HunterSpellSummonHawk,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		MaxRange:       float64(rank.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    timer,
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := baseDamage + 0.05*spell.RangedAttackPower(target)
			outcome := spell.OutcomeMeleeSpecialHitAndCrit
			if rank.AlwaysHits() {
				outcome = spell.OutcomeMeleeSpecialCritOnly
			}
			result := spell.CalcAndDealDamage(sim, target, damage, outcome)
			if !result.Landed() {
				return
			}

			// A free slot, or else the hawk with the least time left.
			hawk := hawks[0].Dot(target)
			for _, h := range hawks[1:] {
				if dot := h.Dot(target); hawk.IsActive() && (!dot.IsActive() || dot.RemainingDuration(sim) < hawk.RemainingDuration(sim)) {
					hawk = dot
				}
			}
			hawk.Apply(sim)
		},
	})
}
