package shaman

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

func (shaman *Shaman) registerRestorationTalents() {
	// Tier 1
	shaman.applyImprovedHealingWave()
	shaman.applyTotemicFocus()

	// Tier 2
	shaman.applyMindfulness()
	shaman.applyNaturalGrace()
	shaman.applyTidalFocus()
	shaman.applyImprovedReincarnation()

	// Tier 3
	shaman.applyAncestralHealing()
	shaman.applyHealingFocus()
	shaman.applyWaterShield()

	// Tier 4
	shaman.applyTidalMastery()
	shaman.applyRestorativeTotems()
	shaman.applyManaTideTotem()

	// Tier 5
	shaman.applyHealingWay()
	shaman.applyNaturesSwiftness()

	// Tier 6
	shaman.applyPurification()

	// Tier 7
	shaman.applyRiptide()
}

func (shaman *Shaman) applyNaturesSwiftness() {
	if !shaman.Talents.NaturesSwiftness {
		return
	}

	nsAura := shaman.RegisterAura(core.Aura{
		ActionID: core.ActionID{SpellID: 16188},
		Label:    "Nature's Swiftness",
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if !spell.Matches(SpellMaskChainLightning | SpellMaskLightningBolt) {
				return
			}
			aura.Deactivate(sim)
		},
	}).AttachSpellMod(core.SpellModConfig{
		Kind:       core.SpellMod_CastTime_Pct,
		FloatValue: -100,
		ClassMask:  SpellMaskChainLightning | SpellMaskLightningBolt,
	})

	shaman.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 16188},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMagic,
		Flags:       core.SpellFlagAPL | core.SpellFlagNoOnCastComplete | SpellFlagInstant,
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    shaman.NewTimer(),
				Duration: time.Second * 180,
			},
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			nsAura.Activate(sim)
		},
	})
}

func (shaman *Shaman) applyRestorativeTotems() {
	if shaman.Talents.RestorativeTotems == 0 {
		return
	}
	// In totems.go
}

func (shaman *Shaman) applyTidalMastery() {
	if shaman.Talents.TidalMastery == 0 {
		return
	}

	// Client 16194's mask is the heals, Lightning Shield and Rolling Thunder - no Lightning Bolt or Chain Lightning.
	shaman.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Percent,
		FloatValue: spellData.TidalMastery.Effect(dbcenums.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CRITICAL_CHANCE)).ValueAt(shaman.Talents.TidalMastery),
		ClassMask:  SpellMaskLightningShield,
	})
}

func (shaman *Shaman) applyTotemicFocus() {
	if shaman.Talents.TotemicFocus == 0 {
		return
	}

	shaman.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_PowerCost_Pct_Add,
		FloatValue: spellData.TotemicFocus.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).FractionAt(shaman.Talents.TotemicFocus),
		ClassMask:  SpellMaskTotem,
	})
}

// applyImprovedHealingWave is Forever's healing talent granting a shorter Healing Wave cast. The DPS specs never
// spend a point here and the sim scores no healing, so it changes no number.
func (shaman *Shaman) applyImprovedHealingWave() {
	if shaman.Talents.ImprovedHealingWave == 0 {
		return
	}
}

// applyMindfulness implements Mindfulness, new in Forever: mana regeneration continues while casting.
func (shaman *Shaman) applyMindfulness() {
	if shaman.Talents.Mindfulness == 0 {
		return
	}

	shaman.PseudoStats.SpiritRegenRateCasting +=
		spellData.Mindfulness.Effect(dbcenums.A_MOD_MANA_REGEN_INTERRUPT, 0).FractionAt(shaman.Talents.Mindfulness)
}

// applyTidalFocus implements Tidal Focus, new in Forever: cheaper heals plus flat melee and spell hit.
// Only the hit changes a damage number here.
func (shaman *Shaman) applyTidalFocus() {
	if shaman.Talents.TidalFocus == 0 {
		return
	}

	points := shaman.Talents.TidalFocus
	shaman.AddStat(stats.MeleeHitRating, core.PhysicalHitRatingPerHitPercent*
		spellData.TidalFocus.Effect(dbcenums.A_MOD_HIT_CHANCE, 0).ValueAt(points))
	shaman.AddStat(stats.SpellHitRating, core.SpellHitRatingPerHitPercent*
		spellData.TidalFocus.Effect(dbcenums.A_MOD_SPELL_HIT_CHANCE, 0).ValueAt(points))
}

// applyNaturalGrace implements Natural Grace, new in Forever: less threat from the shaman's spells.
func (shaman *Shaman) applyNaturalGrace() {
	if shaman.Talents.NaturalGrace == 0 {
		return
	}

	shaman.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_ThreatMultiplier_Pct,
		FloatValue: spellData.NaturalGrace.Effect(dbcenums.A_MOD_THREAT, 126).FractionAt(shaman.Talents.NaturalGrace),
		SpellFlag:  SpellFlagShamanSpell,
	})
}

// applyImprovedReincarnation implements Improved Reincarnation, new in Forever. Only the maximum
// health half is modelled: nothing in the sim dies and comes back.
func (shaman *Shaman) applyImprovedReincarnation() {
	if shaman.Talents.ImprovedReincarnation == 0 {
		return
	}

	shaman.MultiplyStat(stats.Health,
		spellData.ImprovedReincarnation.EffectAt(2).MultiplierAt(shaman.Talents.ImprovedReincarnation))
}

// applyAncestralHealing is Forever's healing talent granting armour on the target of a critical heal. The DPS specs never
// spend a point here and the sim scores no healing, so it changes no number.
func (shaman *Shaman) applyAncestralHealing() {
	if shaman.Talents.AncestralHealing == 0 {
		return
	}
}

// applyHealingFocus is Forever's healing talent granting pushback resistance on heals. The DPS specs never
// spend a point here and the sim scores no healing, so it changes no number.
func (shaman *Shaman) applyHealingFocus() {
	if shaman.Talents.HealingFocus == 0 {
		return
	}
}

// applyWaterShield: the talent teaches Water Shield (409941 -> 408510), which registerWaterShieldSpell
// in shields.go registers when the talent is taken.
func (shaman *Shaman) applyWaterShield() {
	if !shaman.Talents.WaterShield {
		return
	}
}

// applyManaTideTotem: the totem is the ManaTideTotems party buff, which AddPartyBuffs adds.
func (shaman *Shaman) applyManaTideTotem() {
	if !shaman.Talents.ManaTideTotem {
		return
	}
}

// applyHealingWay is Forever's healing talent granting a stacking Healing Wave bonus. The DPS specs never
// spend a point here and the sim scores no healing, so it changes no number.
func (shaman *Shaman) applyHealingWay() {
	if shaman.Talents.HealingWay == 0 {
		return
	}
}

// applyPurification is Forever's healing talent granting stronger heals. The DPS specs never
// spend a point here and the sim scores no healing, so it changes no number.
func (shaman *Shaman) applyPurification() {
	if shaman.Talents.Purification == 0 {
		return
	}
}

// applyRiptide is Forever's healing talent granting a heal over time. The DPS specs never
// spend a point here and the sim scores no healing, so it changes no number.
func (shaman *Shaman) applyRiptide() {
	if !shaman.Talents.Riptide {
		return
	}
}
