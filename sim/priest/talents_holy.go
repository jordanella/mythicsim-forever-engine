package priest

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

func (priest *Priest) registerHolyTalents() {
	// Tier 1
	priest.applyTwilightFocus()
	priest.applyImprovedRenew()
	priest.applyHolySpecialization()

	// Tier 2
	priest.applySpellWarding()
	priest.applyDivineFury()

	// Tier 3
	priest.applyHolyNova()
	priest.applyBlessedRecovery()
	priest.applyInspiration()

	// Tier 4
	priest.applyHolyReach()
	priest.applyImprovedHealing()
	priest.applySearingLight()
	priest.applyBindingHeal()

	// Tier 5
	priest.applyLitanyOfLight()
	priest.applySpiritOfRedemption()
	priest.applySpiritualGuidance()

	// Tier 6
	priest.applySpiritualHealing()

	// Tier 7
	priest.applyPrayerOfMending()
}

// Twilight Focus is new in Forever: pushback protection (14913, SPELLMOD_NOT_LOSE_CASTING_TIME).
// Of the spells its mask names the sim models Smite, Holy Fire, Mind Blast, Mind Flay, Penance and,
// since build 70009, Starshards.
func (priest *Priest) applyTwilightFocus() {
	if priest.Talents.TwilightFocus == 0 {
		return
	}

	resist := spellData.TwilightFocus.FractionAt(priest.Talents.TwilightFocus)
	priest.AddStaticMod(core.SpellModConfig{
		ClassMask: PriestSpellSmite | PriestSpellHolyFire | PriestSpellMindBlast | PriestSpellMindFlay | PriestSpellPenance | PriestSpellStarshards,
		Kind:      core.SpellMod_Custom,
		ApplyCustom: func(_ *core.SpellMod, spell *core.Spell) {
			spell.PushbackResist += resist
		},
		RemoveCustom: func(_ *core.SpellMod, spell *core.Spell) {
			spell.PushbackResist -= resist
		},
	})
}

// applyImprovedRenew implements Improved Renew, new in Forever.
//
// TODO: To be implemented. Renew is not modelled.
func (priest *Priest) applyImprovedRenew() {
	if priest.Talents.ImprovedRenew == 0 {
		return
	}
}

// Holy Specialization is new in Forever: +1% critical strike per point on Holy spells.
func (priest *Priest) applyHolySpecialization() {
	if priest.Talents.HolySpecialization == 0 {
		return
	}

	priest.AddStaticMod(core.SpellModConfig{
		ClassMask:  PriestSpellsAll,
		School:     core.SpellSchoolHoly,
		FloatValue: spellData.HolySpecialization.ValueAt(priest.Talents.HolySpecialization),
		Kind:       core.SpellMod_BonusCrit_Percent,
	})
}

// applySpellWarding implements Spell Warding, new in Forever.
//
// TODO: To be implemented. It is -2% magic damage taken per point (27900), and nothing in a DPS sim
// takes damage.
func (priest *Priest) applySpellWarding() {
	if priest.Talents.SpellWarding == 0 {
		return
	}
}

func (priest *Priest) applyDivineFury() {
	if priest.Talents.DivineFury == 0 {
		return
	}

	priest.AddStaticMod(core.SpellModConfig{
		ClassMask: PriestSpellSmite | PriestSpellHolyFire,
		TimeValue: time.Millisecond * time.Duration(spellData.DivineFury.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CASTING_TIME)).ValueAt(priest.Talents.DivineFury)),
		Kind:      core.SpellMod_CastTime_Flat,
	})
}

func (priest *Priest) applyHolyNova() {
	if !priest.Talents.HolyNova {
		return
	}

	HolyNovaRankMap.Each(func(_ int32, rank *spelldata.Spell) { priest.registerHolyNovaSpell(rank) })
}

var HolyNovaRankMap = spellData.HolyNova

// Damage to everything in range and a heal on the priest's own party, both at the same coefficient.
func (priest *Priest) registerHolyNovaSpell(rank *spelldata.Spell) {
	heal := spellData.HolyNovaTriggered.Rank(rank.RankNumber())
	partyPlayers := priest.Party.Players

	healSpell := priest.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: heal.ID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 0,
		BonusCoefficient: rank.DamageEffect().Coeff(),

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			for _, player := range partyPlayers {
				spell.CalcAndDealHealing(sim, &player.GetCharacter().Unit, heal.HealEffect().Average(core.CharacterLevel), spell.OutcomeHealingCrit)
			}
		},
	})

	priest.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: PriestSpellHolyNova,
		Rank:           rank.RankNumber(),
		MaxRange:       float64(rank.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: rank.DamageEffect().Coeff(),
		ThreatMultiplier: 0,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			spell.CalcAndDealAoeDamageWithVariance(sim, spell.OutcomeMagicHitAndCrit, func(sim *core.Simulation, _ *core.Spell) float64 {
				return rank.DamageEffect().Roll(sim, core.CharacterLevel)
			})
			healSpell.Cast(sim, &priest.Unit)
		},
	})
}

// applyBlessedRecovery implements Blessed Recovery, new in Forever.
//
// TODO: To be implemented. It heals after a critical strike is taken, and nothing in a DPS sim
// takes one.
func (priest *Priest) applyBlessedRecovery() {
	if priest.Talents.BlessedRecovery == 0 {
		return
	}
}

// applyInspiration implements Inspiration.
//
// TODO: To be implemented. It fires on a healing critical strike, and no healing spell is modelled
// on this engine yet (see healer/healer.go).
func (priest *Priest) applyInspiration() {
	if priest.Talents.Inspiration == 0 {
		return
	}
}

// applyHolyReach implements Holy Reach, new in Forever.
//
// TODO: To be implemented. It is range and radius only (27789), which a single-target sim never reads.
func (priest *Priest) applyHolyReach() {
	if priest.Talents.HolyReach == 0 {
		return
	}
}

// applyImprovedHealing implements Improved Healing, new in Forever. It discounts Lesser Heal, Heal,
// Greater Heal, Penance and Prayer of Mending; Penance is the only one of those the sim casts.
func (priest *Priest) applyImprovedHealing() {
	if priest.Talents.ImprovedHealing == 0 {
		return
	}

	priest.AddStaticMod(core.SpellModConfig{
		ClassMask:  PriestSpellPenance,
		FloatValue: spellData.ImprovedHealing.FractionAt(priest.Talents.ImprovedHealing),
		Kind:       core.SpellMod_PowerCost_Pct_Add,
	})
}

// Searing Light now buffs every Holy spell and gives Holy Fire ticks a chance to refund the next
// Holy Nova. The beta client reads 2/5% Holy damage and a 5/10% chance, and the free Holy Nova
// (Holy Purpose, 1284536) lasts 10 sec.
func (priest *Priest) applySearingLight() {
	if priest.Talents.SearingLight == 0 {
		return
	}

	priest.AddStaticMod(core.SpellModConfig{
		ClassMask:  PriestSpellsAll,
		School:     core.SpellSchoolHoly,
		FloatValue: spellData.SearingLight.EffectAt(1).FractionAt(priest.Talents.SearingLight),
		Kind:       core.SpellMod_DamageDone_Pct,
	})

	freeNova := spellData.SearingLightTriggered.Highest()
	costMod := priest.AddDynamicMod(core.SpellModConfig{
		ClassMask:  PriestSpellHolyNova,
		FloatValue: freeNova.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).Average(core.CharacterLevel) / 100,
		Kind:       core.SpellMod_PowerCost_Pct_Add,
	})

	priest.SearingLightAura = priest.RegisterAura(core.Aura{
		Label:    "Searing Light",
		ActionID: core.ActionID{SpellID: freeNova.ID},
		Duration: freeNova.Duration(),
		OnGain: func(_ *core.Aura, _ *core.Simulation) {
			costMod.Activate()
		},
		OnExpire: func(_ *core.Aura, _ *core.Simulation) {
			costMod.Deactivate()
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.Matches(PriestSpellHolyNova) {
				aura.Deactivate(sim)
			}
		},
	})

	priest.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Searing Light Trigger",
		Callback:           core.CallbackOnPeriodicDamageDealt,
		ClassSpellMask:     PriestSpellHolyFire,
		ProcChance:         spellData.SearingLight.EffectAt(2).FractionAt(priest.Talents.SearingLight),
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			priest.SearingLightAura.Activate(sim)
		},
	})
}

// applyBindingHeal implements Binding Heal, new in Forever.
//
// TODO: To be implemented. It is a healing spell.
func (priest *Priest) applyBindingHeal() {
	if !priest.Talents.BindingHeal {
		return
	}
}

// applyLitanyOfLight implements Litany of Light, new in Forever.
//
// TODO: To be implemented. It is a proc off healing (1317006) with no damage half.
func (priest *Priest) applyLitanyOfLight() {
	if priest.Talents.LitanyOfLight == 0 {
		return
	}
}

// Spirit of Redemption is only the on-death form (27827), which is not modelled. Classic's +5%
// Spirit is gone: Forever's 20711 carries one dummy effect and no stat modifier.
func (priest *Priest) applySpiritOfRedemption() {
	if !priest.Talents.SpiritOfRedemption {
		return
	}
}

// The beta client's curves: damage 1/3/5/6/8% of Spirit, healing 5% per point.
func (priest *Priest) applySpiritualGuidance() {
	if priest.Talents.SpiritualGuidance == 0 {
		return
	}

	points := priest.Talents.SpiritualGuidance
	priest.AddStatDependency(stats.Spirit, stats.SpellDamage,
		spellData.SpiritualGuidance.Effect(dbcenums.A_MOD_SPELL_DAMAGE_OF_STAT_PERCENT, 126).FractionAt(points))
	priest.AddStatDependency(stats.Spirit, stats.HealingPower,
		spellData.SpiritualGuidance.Effect(dbcenums.A_MOD_SPELL_HEALING_OF_STAT_PERCENT, 4).FractionAt(points))
}

// applySpiritualHealing implements Spiritual Healing.
//
// TODO: To be implemented. No healing spell is modelled on this engine yet.
func (priest *Priest) applySpiritualHealing() {
	if priest.Talents.SpiritualHealing == 0 {
		return
	}
}

// applyPrayerOfMending implements Prayer of Mending, new in Forever.
//
// TODO: To be implemented. It is a healing spell.
func (priest *Priest) applyPrayerOfMending() {
	if !priest.Talents.PrayerOfMending {
		return
	}
}
