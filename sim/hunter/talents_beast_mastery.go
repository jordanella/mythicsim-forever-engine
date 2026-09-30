package hunter

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

func (hunter *Hunter) registerBeastMasteryTalents() {
	// Tier 1
	// Deadly Aspects: aspects.go
	hunter.registerEnduranceTraining()

	// Tier 2
	hunter.registerFocusedFire()
	hunter.registerImprovedAspectOfTheMonkey()
	hunter.registerPathfinding()
	hunter.registerImprovedRevivePet()

	// Tier 3
	hunter.registerBestialSwiftness()
	hunter.registerUnleashedFury()

	// Tier 4
	hunter.registerImprovedMendPet()
	hunter.registerFerocity()
	// Summon Hawk: summon_hawk.go

	// Tier 5
	hunter.registerSpiritBond()
	hunter.registerIntimidation()
	hunter.registerBestialDiscipline()

	// Tier 6
	hunter.registerFrenzy()

	// Tier 7
	hunter.registerBestialWrath()
}

func (hunter *Hunter) registerEnduranceTraining() {
	if hunter.Pet == nil || hunter.Talents.EnduranceTraining == 0 {
		return
	}

	// Forever drops the hunter's own health bonus: the spell carries only the pet modifier,
	// +3% a rank.
	hunter.Pet.MultiplyStat(stats.Health, spellData.EnduranceTraining.
		Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_ALL_EFFECTS)).
		MultiplierAt(hunter.Talents.EnduranceTraining))
}

func (hunter *Hunter) registerFocusedFire() {
	if hunter.Pet == nil || hunter.Talents.FocusedFire == 0 {
		return
	}

	// The single dummy effect is the 1% per rank damage bonus, for "you and your pet" per the
	// 1223755 tooltip. Forever drops Focused Fire's Kill Command crit bonus - there is no Kill
	// Command.
	multiplier := spellData.FocusedFire.EffectAt(1).MultiplierAt(hunter.Talents.FocusedFire)
	hunter.PseudoStats.DamageDealtMultiplier *= multiplier
	hunter.Pet.PseudoStats.DamageDealtMultiplier *= multiplier
}

// 19616's mask names the pet passive and Summon Hawk, so the hawks take it too.
func (hunter *Hunter) registerUnleashedFury() {
	if hunter.Talents.UnleashedFury == 0 {
		return
	}

	hunter.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Pct,
		FloatValue: spellData.UnleashedFury.FractionAt(hunter.Talents.UnleashedFury),
		ClassMask:  HunterSpellSummonHawk,
	})

	if hunter.Pet != nil {
		hunter.Pet.PseudoStats.DamageDealtMultiplier *= spellData.UnleashedFury.
			MultiplierAt(hunter.Talents.UnleashedFury)
	}
}

// 19598's mask names the pet passive and Summon Hawk, so the hawks take it too.
func (hunter *Hunter) registerFerocity() {
	if hunter.Talents.Ferocity == 0 {
		return
	}

	crit := spellData.Ferocity.ValueAt(hunter.Talents.Ferocity)
	hunter.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Percent,
		FloatValue: crit,
		ClassMask:  HunterSpellSummonHawk,
	})

	if hunter.Pet != nil {
		hunter.Pet.AddStats(stats.Stats{
			stats.PhysicalCritPercent: crit,
			stats.SpellCritPercent:    crit,
		})
	}
}

func (hunter *Hunter) registerBestialDiscipline() {
	if hunter.Talents.BestialDiscipline == 0 {
		return
	}

	// The pet's focus regen is handled where the focus bar is enabled, in pet.go. This half is the
	// hunter's own mana regen while casting.
	hunter.PseudoStats.SpiritRegenRateCasting += spellData.BestialDiscipline.
		Effect(dbcenums.A_MOD_MANA_REGEN_INTERRUPT, 0).
		FractionAt(hunter.Talents.BestialDiscipline)
}

func (hunter *Hunter) registerFrenzy() {
	if hunter.Pet == nil || hunter.Talents.Frenzy == 0 {
		return
	}

	frenzyRank := spellData.FrenzyTriggered.Highest()
	const speedMultiplier = 1.3

	frenzy := hunter.Pet.RegisterAura(core.Aura{
		Label:    "Frenzy Effect",
		ActionID: core.ActionID{SpellID: frenzyRank.ID},
		Duration: frenzyRank.Duration(),
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, speedMultiplier)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1/speedMultiplier)
		},
	})

	hunter.Pet.MakeProcTriggerAura(core.ProcTrigger{
		Name:       "Frenzy",
		Callback:   core.CallbackOnSpellHitDealt,
		Outcome:    core.OutcomeCrit,
		ProcChance: spellData.Frenzy.FractionAt(hunter.Talents.Frenzy),

		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			frenzy.Activate(sim)
		},
	})
}

// Bosses are immune to the stun, but the pet's next attack still gets the crit bonus, so
// Intimidation is worth pressing on cooldown.
func (hunter *Hunter) registerIntimidation() {
	if hunter.Pet == nil || !hunter.Talents.Intimidation {
		return
	}

	rank := spellData.Intimidation.Rank(1)
	actionID := core.ActionID{SpellID: rank.ID}
	const bonusCrit = 100.0

	petAura := hunter.Pet.RegisterAura(core.Aura{
		Label:    "Intimidation",
		ActionID: actionID,
		Duration: rank.Duration(),
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.PhysicalCritPercent, bonusCrit)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.PhysicalCritPercent, -bonusCrit)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Landed() {
				aura.Deactivate(sim)
			}
		},
	})

	intimidation := hunter.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		ProcMask: core.ProcMaskEmpty,
		Flags:    core.SpellFlagAPL,

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

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			petAura.Activate(sim)
		},
	})

	hunter.AddMajorCooldown(core.MajorCooldown{
		Spell: intimidation,
		Type:  core.CooldownTypeDPS,
	})
}

func (hunter *Hunter) registerBestialWrath() {
	if hunter.Pet == nil || !hunter.Talents.BestialWrath {
		return
	}

	rank := spellData.BestialWrath.Highest()
	actionID := core.ActionID{SpellID: rank.ID}
	const damageMultiplier = 1.5

	hunter.Pet.BestialWrathAura = hunter.Pet.RegisterAura(core.Aura{
		Label:    "Bestial Wrath",
		ActionID: actionID,
		Duration: rank.Duration(),
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			hunter.Pet.PseudoStats.DamageDealtMultiplier *= damageMultiplier
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			hunter.Pet.PseudoStats.DamageDealtMultiplier /= damageMultiplier
		},
	})

	bwSpell := hunter.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: HunterSpellBestialWrath,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL,

		ManaCost: rank.ManaCost(),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				NonEmpty: true,
			},
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			hunter.Pet.BestialWrathAura.Activate(sim)
		},
	})

	hunter.AddMajorCooldown(core.MajorCooldown{
		Spell: bwSpell,
		Type:  core.CooldownTypeDPS,
	})
}

// registerImprovedAspectOfTheMonkey implements Improved Aspect of the Monkey, new in Forever.
//
// TODO: dodge while Aspect of the Monkey is up. A ranged hunter holds Aspect of the Hawk, so this
// never applies in the rotations the sim runs.
func (hunter *Hunter) registerImprovedAspectOfTheMonkey() {
	if hunter.Talents.ImprovedAspectOfTheMonkey == 0 {
		return
	}
}

// registerPathfinding implements Pathfinding, new in Forever.
//
// TODO: movement speed only; no effect on damage.
func (hunter *Hunter) registerPathfinding() {
	if hunter.Talents.Pathfinding == 0 {
		return
	}
}

// registerImprovedRevivePet implements Improved Revive Pet, new in Forever.
//
// TODO: out-of-combat pet revival; nothing the sim models.
func (hunter *Hunter) registerImprovedRevivePet() {
	if hunter.Talents.ImprovedRevivePet == 0 {
		return
	}
}

// registerBestialSwiftness implements Bestial Swiftness, new in Forever.
//
// TODO: pet movement speed only.
func (hunter *Hunter) registerBestialSwiftness() {
	if !hunter.Talents.BestialSwiftness {
		return
	}
}

// registerImprovedMendPet implements Improved Mend Pet, new in Forever.
//
// TODO: Mend Pet is a heal the sim does not cast.
func (hunter *Hunter) registerImprovedMendPet() {
	if hunter.Talents.ImprovedMendPet == 0 {
		return
	}
}

// registerSpiritBond implements Spirit Bond, new in Forever.
//
// TODO: health regen for hunter and pet; no effect on damage.
func (hunter *Hunter) registerSpiritBond() {
	if hunter.Talents.SpiritBond == 0 {
		return
	}
}
