package rogue

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/stats"
)

var PVPSet = core.NewItemSet(core.ItemSet{
	Name: "Gladiator's Vestments",
	ID:   577,
	Bonuses: map[int32]core.ApplySetBonus{
		4: func(agent core.Agent, setBonusAura *core.Aura) {
			rogue := agent.(RogueAgent).GetRogue()
			rogue.HasPvpEnergy = true
		},
	},
})

var Dungeon3 = core.NewItemSet(core.ItemSet{
	Name: "Assassination Armor",
	ID:   620,
	Bonuses: map[int32]core.ApplySetBonus{
		2: func(agent core.Agent, setBonusAura *core.Aura) {
			// Your Cheap Shot and Kidney Shot attacks grant you 160 haste rating for 6 sec.
			// NYI
		},
		4: func(agent core.Agent, setBonusAura *core.Aura) {
			// Your Eviscerate ability costs 10 less energy.
			setBonusAura.AttachSpellMod(core.SpellModConfig{
				Kind:      core.SpellMod_PowerCost_Flat,
				ClassMask: RogueSpellEviscerate,
				IntValue:  -10,
			})
		},
	},
})

var Tier4 = core.NewItemSet(core.ItemSet{
	Name: "Netherblade",
	ID:   621,
	Bonuses: map[int32]core.ApplySetBonus{
		2: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.
				ApplyOnGain(func(aura *core.Aura, sim *core.Simulation) {
					agent.(RogueAgent).GetRogue().SliceAndDiceBonusDuration += time.Second * 3
				}).
				ApplyOnExpire(func(aura *core.Aura, sim *core.Simulation) {
					agent.(RogueAgent).GetRogue().SliceAndDiceBonusDuration -= time.Second * 3
				})
		},
		4: func(agent core.Agent, setBonusAura *core.Aura) {
			rogue := agent.(RogueAgent).GetRogue()
			pointMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 37168})
			setBonusAura.AttachProcTrigger(core.ProcTrigger{
				Name:           "Netherblade Combo Point",
				ActionID:       core.ActionID{SpellID: 37168},
				ProcChance:     0.15,
				ClassSpellMask: RogueSpellFinisher,
				Callback:       core.CallbackOnApplyEffects,
				Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					rogue.AddComboPoints(sim, 1, pointMetrics)
				},
			})

		},
	},
})

var Tier5 = core.NewItemSet(core.ItemSet{
	Name: "Deathmantle",
	Bonuses: map[int32]core.ApplySetBonus{
		2: func(agent core.Agent, setBonusAura *core.Aura) {
			agent.(RogueAgent).GetRogue().DeathmantleBonus = 40
		},
		4: func(agent core.Agent, setBonusAura *core.Aura) {
			rogue := agent.(RogueAgent).GetRogue()
			mod := rogue.GetOrRegisterAura(core.Aura{
				Label:    "Coup de Grace",
				Duration: time.Second * 15,
				ActionID: core.ActionID{SpellID: 37171},
				OnApplyEffects: func(aura *core.Aura, sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					if spell.Matches(RogueSpellFinisher) {
						aura.Deactivate(sim)
					}
				},
			}).AttachSpellMod(core.SpellModConfig{
				Kind:       core.SpellMod_PowerCost_Pct_Add,
				ClassMask:  RogueSpellFinisher,
				FloatValue: -1,
			})
			setBonusAura.AttachProcTrigger(core.ProcTrigger{
				Name:             "Deathmantle Proc Trigger",
				CanProcFromProcs: true, // 37171 carries the bit.
				ProcMask:         core.ProcMaskMelee,
				Outcome:          core.OutcomeLanded,
				Callback:         core.CallbackOnSpellHitDealt,
				DPM:              rogue.NewLegacyPPMManager(0.5, core.ProcMaskMelee),
				Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					mod.Activate(sim)
				},
			})
		},
	},
})

var Tier6 = core.NewItemSet(core.ItemSet{
	Name: "Slayer's Armor",
	Bonuses: map[int32]core.ApplySetBonus{
		2: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.
				ApplyOnGain(func(aura *core.Aura, sim *core.Simulation) {
					agent.(RogueAgent).GetRogue().SliceAndDiceBonusFlat += 0.05
				}).
				ApplyOnExpire(func(aura *core.Aura, sim *core.Simulation) {
					agent.(RogueAgent).GetRogue().SliceAndDiceBonusFlat -= 0.05
				})
		},
		4: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachSpellMod(core.SpellModConfig{
				Kind:       core.SpellMod_DamageDone_Flat,
				ClassMask:  RogueSpellBackstab | RogueSpellSinisterStrike | RogueSpellMutilate | RogueSpellHemorrhage,
				FloatValue: 0.06,
			})
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                   Forever / Classic sets (ported from master)
///////////////////////////////////////////////////////////////////////////

var rogueAllResistances8 = stats.Stats{
	stats.ArcaneResistance: 8,
	stats.FireResistance:   8,
	stats.FrostResistance:  8,
	stats.NatureResistance: 8,
	stats.ShadowResistance: 8,
}

var ItemSetNightslayerArmor = core.NewItemSet(core.ItemSet{
	Name: "Nightslayer Armor",
	Bonuses: map[int32]core.ApplySetBonus{
		// Reduces the cooldown of your Vanish ability by 30 sec.
		3: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachSpellMod(core.SpellModConfig{
				Kind:      core.SpellMod_Cooldown_Flat,
				ClassMask: RogueSpellVanish,
				TimeValue: -time.Second * 30,
			})
		},
		// Increases your maximum Energy by 10. Applied at build time, before the energy bar resets,
		// and only when the set is worn at the start (not for an item-swap-only set).
		// ponytail: static max energy, make it dynamic if item swaps with this set ever matter.
		5: func(agent core.Agent, setBonusAura *core.Aura) {
			c := agent.GetCharacter()
			if c.HasEnergyBar() && setBonusAura.BuildPhase == core.CharacterBuildPhaseGear {
				c.UpdateMaxEnergy(nil, 10, nil)
			}
		},
		// Heals the rogue for 500 when Vanish is performed.
		8: func(agent core.Agent, setBonusAura *core.Aura) {
			c := agent.GetCharacter()
			healthMetrics := c.NewHealthMetrics(core.ActionID{SpellID: 23582})
			setBonusAura.AttachProcTrigger(core.ProcTrigger{
				Name:           "Clean Escape",
				Callback:       core.CallbackOnCastComplete,
				ClassSpellMask: RogueSpellVanish,
				Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
					c.GainHealth(sim, 500, healthMetrics)
				},
			})
		},
	},
})

var ItemSetBloodfangArmor = core.NewItemSet(core.ItemSet{
	Name: "Bloodfang Armor",
	Bonuses: map[int32]core.ApplySetBonus{
		// Increases the chance to apply poisons to your target by 5%.
		3: func(agent core.Agent, setBonusAura *core.Aura) {
			rogue := agent.(RogueAgent).GetRogue()
			setBonusAura.AttachAdditivePseudoStatBuff(&rogue.additivePoisonBonusChance, 0.05)
		},
		// Improves the threat reduction of Feint by 25%. Feint is not modelled.
		5: func(agent core.Agent, setBonusAura *core.Aura) {},
		// Chance on melee hit (1 PPM) to deal 283 to 317 damage and heal the rogue for 50 every sec for 6 sec.
		8: func(agent core.Agent, setBonusAura *core.Aura) {
			c := agent.GetCharacter()

			bloodfangHeal := c.GetOrRegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 23580},
				SpellSchool: core.SpellSchoolPhysical,
				DefenseType: core.DefenseTypeMelee,
				ProcMask:    core.ProcMaskEmpty,
				Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

				DamageMultiplier: 1,

				Hot: core.DotConfig{
					Aura: core.Aura{
						Label: "Bloodfang",
					},
					NumberOfTicks: 6,
					TickLength:    time.Second,
					OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
						dot.SnapshotHeal(target, 50)
					},
					OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
						dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.OutcomeTick)
					},
				},
				ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					spell.Hot(&c.Unit).Apply(sim)
				},
			})

			procSpell := c.GetOrRegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 23581},
				SpellSchool: core.SpellSchoolPhysical,
				DefenseType: core.DefenseTypeMelee,
				ProcMask:    core.ProcMaskEmpty,
				Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

				DamageMultiplier: 1,
				ThreatMultiplier: 1,

				ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					spell.CalcAndDealDamage(sim, target, sim.Roll(283, 317), spell.OutcomeMagicCrit)
				},
			})

			setBonusAura.AttachProcTrigger(core.ProcTrigger{
				Name:         "Bloodfang",
				Callback:     core.CallbackOnSpellHitDealt,
				Outcome:      core.OutcomeLanded,
				IsWeaponProc: true,
				DPM:          c.NewLegacyPPMManager(1, core.ProcMaskMelee),
				Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
					procSpell.Cast(sim, result.Target)
					bloodfangHeal.Cast(sim, result.Target)
				},
			})
		},
	},
})

var ItemSetMadcapsOutfit = core.NewItemSet(core.ItemSet{
	Name: "Madcap's Outfit",
	Bonuses: map[int32]core.ApplySetBonus{
		// +20 Attack Power.
		2: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachStatsBuff(stats.Stats{stats.AttackPower: 20, stats.RangedAttackPower: 20})
		},
		// Decreases the cooldown of Blind by 20 sec. Blind is not modelled.
		3: func(agent core.Agent, setBonusAura *core.Aura) {},
		// Decreases the energy cost of Eviscerate and Rupture by 5.
		5: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachSpellMod(core.SpellModConfig{
				Kind:      core.SpellMod_PowerCost_Flat,
				ClassMask: RogueSpellEviscerate | RogueSpellRupture,
				IntValue:  -5,
			})
		},
	},
})

var ItemSetEmblemsOfVeiledShadows = core.NewItemSet(core.ItemSet{
	Name: "Emblems of Veiled Shadows",
	Bonuses: map[int32]core.ApplySetBonus{
		// Decreases the energy cost of Slice and Dice by 10.
		3: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachSpellMod(core.SpellModConfig{
				Kind:      core.SpellMod_PowerCost_Flat,
				ClassMask: RogueSpellSliceAndDice,
				IntValue:  -10,
			})
		},
	},
})

var ItemSetDarkmantleArmor = core.NewItemSet(core.ItemSet{
	Name: "Darkmantle Armor",
	Bonuses: map[int32]core.ApplySetBonus{
		// +8 All Resistances.
		2: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachStatsBuff(rogueAllResistances8)
		},
		// Vitality: 15 health every 5 sec. Nothing to apply in a boss fight.
		3: func(agent core.Agent, setBonusAura *core.Aura) {},
		// Chance on melee attack (1 PPM) to restore 20 energy.
		4: func(agent core.Agent, setBonusAura *core.Aura) {
			c := agent.GetCharacter()
			actionID := core.ActionID{SpellID: 27787}
			energyMetrics := c.NewEnergyMetrics(actionID)
			setBonusAura.AttachProcTrigger(core.ProcTrigger{
				ActionID: actionID,
				Name:     "Rogue Armor Energize",
				Callback: core.CallbackOnSpellHitDealt,
				Outcome:  core.OutcomeLanded,
				DPM:      c.NewLegacyPPMManager(1, core.ProcMaskMeleeWhiteHit),
				Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
					if c.HasEnergyBar() {
						c.AddEnergy(sim, 20, energyMetrics)
					}
				},
			})
		},
		// Crafted Shadows: breaks a Snare when struck.
		5: func(agent core.Agent, setBonusAura *core.Aura) {},
		// +40 Attack Power.
		6: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachStatsBuff(stats.Stats{stats.AttackPower: 40, stats.RangedAttackPower: 40})
		},
	},
})

var ItemSetDeathdealersEmbrace = core.NewItemSet(core.ItemSet{
	Name: "Deathdealer's Embrace",
	Bonuses: map[int32]core.ApplySetBonus{
		// Reduces the cooldown of Evasion by 1 min. Evasion is not modelled.
		3: func(agent core.Agent, setBonusAura *core.Aura) {},
		// 15% increased damage to your Eviscerate ability.
		5: func(agent core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachSpellMod(core.SpellModConfig{
				Kind:       core.SpellMod_DamageDone_Flat,
				ClassMask:  RogueSpellEviscerate,
				FloatValue: 0.15,
			})
		},
	},
})

// PvP sets: +40 Attack Power and +20 Stamina at the given piece counts. The 4 piece Gouge
// cooldown reduction is not modelled (Gouge is never cast).
func roguePvPSet(name string, apAt, stamAt int32) *core.ItemSet {
	return core.NewItemSet(core.ItemSet{
		Name: name,
		Bonuses: map[int32]core.ApplySetBonus{
			apAt: func(agent core.Agent, setBonusAura *core.Aura) {
				setBonusAura.AttachStatsBuff(stats.Stats{stats.AttackPower: 40, stats.RangedAttackPower: 40})
			},
			4: func(agent core.Agent, setBonusAura *core.Aura) {},
			stamAt: func(agent core.Agent, setBonusAura *core.Aura) {
				setBonusAura.AttachStatBuff(stats.Stamina, 20)
			},
		},
	})
}

var ItemSetChampionsGuard = roguePvPSet("Champion's Guard", 2, 6)
var ItemSetLieutenantCommandersGuard = roguePvPSet("Lieutenant Commander's Guard", 2, 6)
var ItemSetWarlordsVestments = roguePvPSet("Warlord's Vestments", 6, 2)
var ItemSetFieldMarshalsVestments = roguePvPSet("Field Marshal's Vestments", 6, 2)

func init() {
	// Renataki's Charm of Trickery
	// https://www.wowhead.com/forever/item=19954/renatakis-charm-of-trickery
	//
	// Use: Instantly increases your energy by 60 (24532). 3 min cooldown, 10 sec on the burst
	// trinket category.
	core.NewItemEffect(19954, func(agent core.Agent) {
		rogue := agent.(RogueAgent).GetRogue()
		energyMetrics := rogue.NewEnergyMetrics(core.ActionID{SpellID: 24532})

		spell := rogue.RegisterSpell(core.SpellConfig{
			ActionID: core.ActionID{ItemID: 19954},
			ProcMask: core.ProcMaskEmpty,
			Flags:    core.SpellFlagNoOnCastComplete,

			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    rogue.NewTimer(),
					Duration: time.Minute * 3,
				},
				SharedCD: core.Cooldown{
					Timer:    rogue.GetOffensiveTrinketCD(),
					Duration: time.Second * 10,
				},
			},

			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
				rogue.AddEnergy(sim, 60, energyMetrics)
			},
		})

		rogue.AddMajorCooldown(core.MajorCooldown{
			Spell: spell,
			Type:  core.CooldownTypeDPS,
			ShouldActivate: func(_ *core.Simulation, _ *core.Character) bool {
				// Room for all 60, so none of it is lost to the cap.
				return rogue.CurrentEnergy() <= rogue.MaximumEnergy()-60
			},
		})
	})
}
