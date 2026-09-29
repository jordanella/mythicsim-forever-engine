package shaman

import (
	"fmt"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

const (
	frostbrandEnchantID  int32 = 2
	flametongueEnchantID int32 = 5
	windfuryEnchantID    int32 = 283
	earthlivingEnchantID int32 = 3345
	rockbiterEnchantID   int32 = 3021
)

func (shaman *Shaman) RegisterOnItemSwapWithImbue(effectID int32, procMask *core.ProcMask, aura *core.Aura) {
	shaman.RegisterItemSwapCallback(core.AllWeaponSlots(), func(sim *core.Simulation, slot proto.ItemSlot) {
		mask := core.ProcMaskUnknown
		if shaman.MainHand().TempEnchant == effectID {
			mask |= core.ProcMaskMeleeMH
		}
		if shaman.OffHand().TempEnchant == effectID {
			mask |= core.ProcMaskMeleeOH
		}
		*procMask = mask

		if mask == core.ProcMaskUnknown {
			aura.Deactivate(sim)
		} else {
			aura.Activate(sim)
		}
	})
}

func (shaman *Shaman) setupItemSwapImbue(imbue proto.ShamanImbue, imbueID int32) {
	if shaman.ItemSwap.IsEnabled() {
		if mhSwap := shaman.ItemSwap.GetUnequippedItemBySlot(proto.ItemSlot_ItemSlotMainHand); mhSwap != nil && shaman.SelfBuffs.ImbueMHSwap == imbue {
			mhSwap.TempEnchant = imbueID
			shaman.ItemSwap.AddTempEnchant(imbueID, proto.ItemSlot_ItemSlotMainHand, true)
		}
		if ohSwap := shaman.ItemSwap.GetUnequippedItemBySlot(proto.ItemSlot_ItemSlotOffHand); ohSwap != nil && shaman.SelfBuffs.ImbueOHSwap == imbue {
			ohSwap.TempEnchant = imbueID
			shaman.ItemSwap.AddTempEnchant(imbueID, proto.ItemSlot_ItemSlotOffHand, true)
		}
	}
}

func (shaman *Shaman) newWindfuryImbueSpell(isMH bool) *core.Spell {
	tag := 1
	procMask := core.ProcMaskMeleeMHSpecial
	weaponDamageFunc := shaman.MHWeaponDamage
	if !isMH {
		tag = 2
		procMask = core.ProcMaskMeleeOHSpecial
		weaponDamageFunc = shaman.OHWeaponDamage
	}

	spellConfig := core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 25505, Tag: int32(tag)},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee, // Windfury Attack (25504)
		ProcMask:       procMask,
		ClassSpellMask: SpellMaskWindfuryWeapon,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			apBonus := shaman.WindfuryAPBonus * (1 + spellData.ElementalWeapons.EffectAt(3).FractionAt(shaman.Talents.ElementalWeapons))
			mAP := spell.MeleeAttackPower(target) + apBonus

			baseDamage1 := weaponDamageFunc(sim, mAP)
			baseDamage2 := weaponDamageFunc(sim, mAP)
			result1 := spell.CalcDamage(sim, target, baseDamage1, spell.OutcomeMeleeSpecialHitAndCrit)
			result2 := spell.CalcDamage(sim, target, baseDamage2, spell.OutcomeMeleeSpecialHitAndCrit)
			spell.DealDamage(sim, result1)
			spell.DealDamage(sim, result2)
		},
	}

	return shaman.RegisterSpell(spellConfig)
}

func (shaman *Shaman) makeWFProcTriggerAura(dpm *core.DynamicProcManager, procMask *core.ProcMask, mhSpell *core.Spell, ohSpell *core.Spell) *core.Aura {
	aura := shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Windfury Imbue",
		Callback:           core.CallbackOnSpellHitDealt,
		ProcMask:           *procMask,
		IsWeaponProc:       true,
		Outcome:            core.OutcomeLanded,
		ICD:                time.Millisecond * 1500,
		DPM:                dpm,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.IsMH() {
				mhSpell.Cast(sim, result.Target)
			} else {
				ohSpell.Cast(sim, result.Target)
			}
		},
	})
	return aura
}

func (shaman *Shaman) getWindfuryFixedProcChance(procMask core.ProcMask) float64 {
	return 0.2
}

// TODO: To be implemented. Not verified against Forever.
func (shaman *Shaman) RegisterWindfuryImbue(procMask core.ProcMask) {
	if procMask == core.ProcMaskUnknown && !shaman.ItemSwap.IsEnabled() {
		return
	}

	mask := core.ProcMaskUnknown

	mH := shaman.MainHand()
	if mH != nil && shaman.SelfBuffs.ImbueMH == proto.ShamanImbue_WindfuryWeapon {
		mH.TempEnchant = windfuryEnchantID
		if shaman.ItemSwap.IsEnabled() {
			shaman.ItemSwap.AddTempEnchant(windfuryEnchantID, proto.ItemSlot_ItemSlotMainHand, false)
		}
		mask |= core.ProcMaskMeleeMH
	}
	oH := shaman.OffHand()
	if oH != nil && shaman.SelfBuffs.ImbueOH == proto.ShamanImbue_WindfuryWeapon {
		oH.TempEnchant = windfuryEnchantID
		if shaman.ItemSwap.IsEnabled() {
			shaman.ItemSwap.AddTempEnchant(windfuryEnchantID, proto.ItemSlot_ItemSlotOffHand, false)
		}
		mask |= core.ProcMaskMeleeOH
	}

	shaman.setupItemSwapImbue(proto.ShamanImbue_WindfuryWeapon, windfuryEnchantID)

	dpm := shaman.NewDynamicLegacyProcForTempEnchant(windfuryEnchantID, 0, shaman.getWindfuryFixedProcChance)

	mhSpell := shaman.newWindfuryImbueSpell(true)
	ohSpell := shaman.newWindfuryImbueSpell(false)

	aura := shaman.makeWFProcTriggerAura(dpm, &mask, mhSpell, ohSpell)

	if mask.Matches(core.ProcMaskMeleeMH) {
		aura.NewExclusiveEffect(buffs.WindfuryTotemCategory, false, core.ExclusiveEffect{
			Priority: shaman.WindfuryAPBonus * 2, // Need to be higher than Windfury Totem priority
		})
	}

	shaman.RegisterOnItemSwapWithImbue(windfuryEnchantID, &mask, aura)
}

var flametongueImbue = spellData.FlametongueWeaponTriggered.Highest()
var frostbrandImbue = spellData.FrostbrandWeaponTriggered.Highest()

func (shaman *Shaman) newFlametongueImbueSpell(weapon *core.Item) *core.Spell {
	return shaman.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: flametongueImbue.ID},
		SpellSchool: core.SpellSchoolFire,
		// The damage logs as Flametongue Attack (10444), Magic in SpellCategories; it crits for 1.5x
		// (2.0x with Elemental Fury, see talents_elemental.go).
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamageProc,
		ClassSpellMask:   SpellMaskFlametongueWeapon,
		Flags:            core.SpellFlagPassiveSpell | core.SpellFlagProc | SpellFlagShamanSpell,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 0.10000000149,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if weapon.SwingSpeed != 0 {
				// The proc's dummy value is hundredths of damage per second of weapon speed, speed held
				// to 1.3-4.0 (the tooltip's "(X / 77 - 1) to (X / 25)"): 16344's 2810 is 35 to 112 at 60.
				// A beta log bears the scale out at rank 1 (8026, 4.4 a second): 100 hits from a 2.5
				// speed weapon with no spell power averaged 11.2 (foreverlogs.gg report 2671).
				speed := min(max(weapon.SwingSpeed, 1.3), 4)
				baseDamage := speed * flametongueImbue.EffectN(1).Average(core.CharacterLevel) / 100
				spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			}
		},
	})
}

func (shaman *Shaman) makeFTProcTriggerAura(itemSlot proto.ItemSlot, triggerProcMask core.ProcMask, flameTongueSpell *core.Spell) *core.Aura {
	aura := shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:               fmt.Sprintf("Flametongue Imbue %s", itemSlot),
		ProcMask:           triggerProcMask,
		IsWeaponProc:       true,
		Outcome:            core.OutcomeLanded,
		Callback:           core.CallbackOnSpellHitDealt,
		TriggerImmediately: true,

		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			flameTongueSpell.Cast(sim, result.Target)
		},
	})

	shaman.RegisterItemSwapCallback([]proto.ItemSlot{itemSlot}, func(sim *core.Simulation, is proto.ItemSlot) {
		if is == proto.ItemSlot_ItemSlotMainHand {
			mh := shaman.MainHand()
			mhSwap := shaman.ItemSwap.GetUnequippedItemBySlot(is)
			if mh.TempEnchant != flametongueEnchantID {
				// The new main hand does not have flametongue on, so deactivate
				aura.Deactivate(sim)
				return
			}
			if mhSwap.TempEnchant != flametongueEnchantID {
				// The new main hand has flametongue on and the swapped one does not, so need to activate
				aura.Activate(sim)
				return
			}
		}
		if is == proto.ItemSlot_ItemSlotOffHand {
			oh := shaman.OffHand()
			ohSwap := shaman.ItemSwap.GetUnequippedItemBySlot(is)
			if oh.TempEnchant != flametongueEnchantID {
				// The new offhand does not have flametongue on, so deactivate
				aura.Deactivate(sim)
				return
			}
			if ohSwap.TempEnchant != flametongueEnchantID {
				// The new offhand has flametongue on and the swapped one does not, so need to activate
				aura.Activate(sim)
				return
			}

		}
	})

	return aura
}

// TODO: To be implemented. Not verified against Forever.
func (shaman *Shaman) RegisterFlametongueImbue(procMask core.ProcMask) {
	if procMask == core.ProcMaskUnknown && !shaman.ItemSwap.IsEnabled() {
		return
	}

	for _, itemSlot := range core.AllWeaponSlots() {
		var weapon *core.Item
		var triggerProcMask core.ProcMask
		switch {
		case shaman.SelfBuffs.ImbueMH == proto.ShamanImbue_FlametongueWeapon && itemSlot == proto.ItemSlot_ItemSlotMainHand:
			weapon = shaman.MainHand()
			triggerProcMask = core.ProcMaskMeleeMH
		case shaman.SelfBuffs.ImbueOH == proto.ShamanImbue_FlametongueWeapon && itemSlot == proto.ItemSlot_ItemSlotOffHand:
			weapon = shaman.OffHand()
			triggerProcMask = core.ProcMaskMeleeOH
		}

		if weapon == nil {
			continue
		}

		weapon.TempEnchant = flametongueEnchantID

		if shaman.ItemSwap.IsEnabled() {
			shaman.ItemSwap.AddTempEnchant(flametongueEnchantID, itemSlot, false)
		}

		flameTongueSpell := shaman.newFlametongueImbueSpell(weapon)
		aura := shaman.makeFTProcTriggerAura(itemSlot, triggerProcMask, flameTongueSpell)
		if itemSlot == proto.ItemSlot_ItemSlotMainHand {
			aura.NewExclusiveEffect(buffs.WindfuryTotemCategory, false, core.ExclusiveEffect{
				Priority: shaman.WindfuryAPBonus * 2, // Need to be higher than Windfury Totem priority
			})
		}
	}

	shaman.setupItemSwapImbue(proto.ShamanImbue_FlametongueWeapon, flametongueEnchantID)
}

func (shaman *Shaman) newFrostbrandImbueSpell() *core.Spell {
	return shaman.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: frostbrandImbue.ID},
		SpellSchool:    frostbrandImbue.SpellSchool(),
		DefenseType:    core.DefenseTypeMagic, // Frostbrand Attack (25501 / 38617) is Magic in SpellCategories
		ClassSpellMask: SpellMaskFrostbrandWeapon,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagPassiveSpell | core.SpellFlagProc | SpellFlagShamanSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: frostbrandImbue.DamageEffect().Coeff(),
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, frostbrandImbue.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
		},
	})
}

// TODO: To be implemented. Not verified against Forever.
func (shaman *Shaman) RegisterFrostbrandImbue(procMask core.ProcMask) {
	if procMask == core.ProcMaskUnknown && !shaman.ItemSwap.IsEnabled() {
		return
	}

	mH := shaman.MainHand()
	if mH != nil && shaman.SelfBuffs.ImbueMH == proto.ShamanImbue_FrostbrandWeapon {
		mH.TempEnchant = frostbrandEnchantID
		if shaman.ItemSwap.IsEnabled() {
			shaman.ItemSwap.AddTempEnchant(frostbrandEnchantID, proto.ItemSlot_ItemSlotMainHand, false)
		}
	}
	oH := shaman.OffHand()
	if oH != nil && shaman.SelfBuffs.ImbueOH == proto.ShamanImbue_FrostbrandWeapon {
		oH.TempEnchant = frostbrandEnchantID
		if shaman.ItemSwap.IsEnabled() {
			shaman.ItemSwap.AddTempEnchant(frostbrandEnchantID, proto.ItemSlot_ItemSlotOffHand, false)
		}
	}

	shaman.setupItemSwapImbue(proto.ShamanImbue_FrostbrandWeapon, frostbrandEnchantID)

	dpm := shaman.NewDynamicLegacyProcForTempEnchant(frostbrandEnchantID, 9.0, func(pm core.ProcMask) float64 { return 0 })

	fbSpell := shaman.newFrostbrandImbueSpell()

	aura := shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Frostbrand Imbue",
		Callback:           core.CallbackOnSpellHitDealt,
		IsWeaponProc:       true,
		Outcome:            core.OutcomeLanded,
		DPM:                dpm,
		TriggerImmediately: true,

		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			fbSpell.Cast(sim, result.Target)
		},
	})

	shaman.RegisterOnItemSwapWithImbue(frostbrandEnchantID, &procMask, aura)
}

var rockbiterImbue = spellData.RockbiterWeaponTriggered.Highest()

// Rockbiter Weapon (rank 7, cast as 16316) is not a proc. The client's imbue passive 16313 is a
// permanent aura, and the tooltip reads "increasing melee attack power by 653 and allowing melee
// attacks to cause additional threat when using that weapon". Elemental Weapons raises the effect
// (spell mod class mask SpellMaskRockbiterWeapon, effect 1), which acts here on the attack power
// value itself since nothing casts a Rockbiter spell.
//
// One aura serves both hands: the passive is a single spell, so a second Rockbiter weapon does not
// add the attack power twice. The threat half is left out, since nothing here models threat for
// a DPS spec, and the 5 minute imbue duration is not modelled for any imbue.
func (shaman *Shaman) RegisterRockbiterImbue(procMask core.ProcMask) {
	if procMask == core.ProcMaskUnknown && !shaman.ItemSwap.IsEnabled() {
		return
	}

	imbued := false
	mH := shaman.MainHand()
	if mH != nil && shaman.SelfBuffs.ImbueMH == proto.ShamanImbue_RockbiterWeapon {
		mH.TempEnchant = rockbiterEnchantID
		if shaman.ItemSwap.IsEnabled() {
			shaman.ItemSwap.AddTempEnchant(rockbiterEnchantID, proto.ItemSlot_ItemSlotMainHand, false)
		}
		imbued = true
	}
	oH := shaman.OffHand()
	if oH != nil && shaman.SelfBuffs.ImbueOH == proto.ShamanImbue_RockbiterWeapon {
		oH.TempEnchant = rockbiterEnchantID
		if shaman.ItemSwap.IsEnabled() {
			shaman.ItemSwap.AddTempEnchant(rockbiterEnchantID, proto.ItemSlot_ItemSlotOffHand, false)
		}
		imbued = true
	}

	shaman.setupItemSwapImbue(proto.ShamanImbue_RockbiterWeapon, rockbiterEnchantID)

	if !imbued {
		return
	}

	attackPower := rockbiterImbue.EffectN(1).Average(core.CharacterLevel) *
		(1 + spellData.ElementalWeapons.EffectAt(1).FractionAt(shaman.Talents.ElementalWeapons))
	core.MakePermanent(shaman.NewTemporaryStatsAura(
		"Rockbiter Weapon",
		core.ActionID{SpellID: rockbiterImbue.ID},
		stats.Stats{stats.AttackPower: attackPower},
		core.NeverExpires,
	).Aura)
}
