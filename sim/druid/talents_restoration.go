package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

func (druid *Druid) registerRestorationTalents() {
	// Tier 1
	druid.applyNaturesFocus()
	druid.applyFuror()

	// Tier 2
	druid.applyNaturalist()
	druid.applySubtlety()
	druid.applyNaturalShapeshifter()

	// Tier 3
	druid.applyReflection()
	druid.applyGiftOfNature()
	druid.applyGiftOfTheEarthmother()

	// Tier 4
	druid.applyTranquilSpirit()
	druid.applyImprovedRejuvenation()
	druid.applySwiftmend()

	// Tier 5
	druid.applyNaturesSwiftness()
	druid.applyLivingSpirit()
	druid.applyImprovedTranquility()

	// Tier 6
	druid.applyImprovedRegrowth()

	// Tier 7
	druid.applyWildGrowth()
}

func (druid *Druid) applyNaturalShapeshifter() {
	if druid.Talents.NaturalShapeshifter == 0 {
		return
	}

	// Client 16833: the mask covers Cat, Bear and Moonkin Form, and since client 70170 Shifting Power,
	// whose tooltip says its cost is reduced by effects that reduce the cost of Shapeshifting.
	druid.AddStaticMod(core.SpellModConfig{
		ClassMask:  DruidSpellCatForm | DruidSpellBearForm | DruidSpellMoonkinForm | DruidSpellShiftingPower,
		Kind:       core.SpellMod_PowerCost_Pct_Add,
		FloatValue: spellData.NaturalShapeshifter.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).FractionAt(druid.Talents.NaturalShapeshifter),
	})
}

func (druid *Druid) applyNaturalist() {
	if druid.Talents.Naturalist == 0 {
		return
	}

	// Forever states the damage bonus against every school (mask 127), not physical only.
	druid.PseudoStats.DamageDealtMultiplier *= spellData.Naturalist.Effect(dbcenums.A_MOD_DAMAGE_PERCENT_DONE, 127).MultiplierAt(druid.Talents.Naturalist)
}

func (druid *Druid) applySubtlety() {
	if druid.Talents.Subtlety == 0 {
		return
	}

	// Reduces the threat of the Arcane and Nature spells (school mask 72) by 10% a rank: all of
	// them, Faerie Fire's flat threat and Thorns included.
	threatReduction := spellData.Subtlety.Effect(dbcenums.A_MOD_THREAT, 72).FractionAt(druid.Talents.Subtlety)
	druid.AddStaticMod(core.SpellModConfig{
		School:     core.SpellSchoolArcane | core.SpellSchoolNature,
		Kind:       core.SpellMod_ThreatMultiplier_Pct,
		FloatValue: threatReduction,
	})
	druid.AddStaticMod(core.SpellModConfig{
		School:     core.SpellSchoolArcane | core.SpellSchoolNature,
		Kind:       core.SpellMod_FlatThreatBonus_Pct,
		FloatValue: threatReduction,
	})
}

func (druid *Druid) applyLivingSpirit() {
	if druid.Talents.LivingSpirit == 0 {
		return
	}

	druid.MultiplyStat(stats.Spirit, spellData.LivingSpirit.Effect(dbcenums.A_MOD_TOTAL_STAT_PERCENTAGE, 0).MultiplierAt(druid.Talents.LivingSpirit))
}

// applyNaturesFocus implements Nature's Focus, new in Forever.
//
// TODO: not modelled - the client states spell pushback, which the sim does not simulate.
func (druid *Druid) applyNaturesFocus() {
	if druid.Talents.NaturesFocus == 0 {
		return
	}
}

// Furor: a 20% chance a rank at 10 Rage when shifting into Bear Form, and Forever's Energy
// carry-over on a Cat powershift. Both halves are read in forms.go; the chance is kept here.
func (druid *Druid) applyFuror() {
	if druid.Talents.Furor == 0 {
		return
	}

	// Both dummy effects carry the same ladder, one per form, so either answers the chance.
	druid.FurorProcChance = spellData.Furor.EffectAt(1).FractionAt(druid.Talents.Furor)

	// A permanent aura so an APL can check for Furor before powershifting (auraIsKnown 17056).
	core.MakePermanent(druid.RegisterAura(core.Aura{
		Label:    "Furor",
		ActionID: core.ActionID{SpellID: spellData.Furor.Highest().ID},
	}))
}

// Reflection, new in Forever: a share of Spirit regeneration continues while casting.
func (druid *Druid) applyReflection() {
	if druid.Talents.Reflection == 0 {
		return
	}

	druid.PseudoStats.SpiritRegenRateCasting += spellData.Reflection.Effect(dbcenums.A_MOD_MANA_REGEN_INTERRUPT, 0).FractionAt(druid.Talents.Reflection)
}

// applyGiftOfNature implements Gift of Nature, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applyGiftOfNature() {
	if druid.Talents.GiftOfNature == 0 {
		return
	}
}

// applyGiftOfTheEarthmother implements Gift of the Earthmother, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applyGiftOfTheEarthmother() {
	if !druid.Talents.GiftOfTheEarthmother {
		return
	}
}

// applyTranquilSpirit implements Tranquil Spirit, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applyTranquilSpirit() {
	if druid.Talents.TranquilSpirit == 0 {
		return
	}
}

// applyImprovedRejuvenation implements Improved Rejuvenation, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applyImprovedRejuvenation() {
	if druid.Talents.ImprovedRejuvenation == 0 {
		return
	}
}

// applySwiftmend implements Swiftmend, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applySwiftmend() {
	if !druid.Talents.Swiftmend {
		return
	}
}

// applyNaturesSwiftness implements Nature's Swiftness, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applyNaturesSwiftness() {
	if !druid.Talents.NaturesSwiftness {
		return
	}
}

// applyImprovedTranquility implements Improved Tranquility, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applyImprovedTranquility() {
	if druid.Talents.ImprovedTranquility == 0 {
		return
	}
}

// applyImprovedRegrowth implements Improved Regrowth, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applyImprovedRegrowth() {
	if druid.Talents.ImprovedRegrowth == 0 {
		return
	}
}

// applyWildGrowth implements Wild Growth, new in Forever.
//
// TODO: not modelled - the sim has no healing rotation for this spec.
func (druid *Druid) applyWildGrowth() {
	if !druid.Talents.WildGrowth {
		return
	}
}
