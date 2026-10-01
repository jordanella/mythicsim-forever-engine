package core

import (
	"math"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

func applyRaceEffects(agent Agent) {
	character := agent.GetCharacter()

	switch character.Race {
	case proto.Race_RaceDwarf:
		applyWeaponSpecialization(character, "Mace Specialization", 1259719, 1, proto.WeaponType_WeaponTypeMace)
		applyCreatureTypeSlaying(character, proto.MobType_MobTypeBeast)

		actionID := ActionID{SpellID: 20594}

		stoneFormAura := character.RegisterAura(Aura{
			Label:    "Stoneform",
			ActionID: actionID,
			Duration: time.Second * 8,
		}).AttachMultiplicativePseudoStatBuff(&character.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexPhysical], 0.9)

		spell := character.RegisterSpell(SpellConfig{
			ActionID: actionID,
			Flags:    SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				DefaultCast: Cast{
					GCD: GCDDefault,
				},
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute * 3,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				stoneFormAura.Activate(sim)
			},

			RelatedSelfBuff: stoneFormAura,
		})

		character.AddMajorCooldown(MajorCooldown{
			Spell: spell,
			Type:  CooldownTypeSurvival,
		})
	case proto.Race_RaceGnome:
		applyExpansiveMind(character)
		applyEureka(character)
	case proto.Race_RaceHuman:
		character.MultiplyStat(stats.Spirit, 1.05)
		applyWeaponSpecialization(character, "Sword Specialization", 20597, 2, proto.WeaponType_WeaponTypeSword)
	case proto.Race_RaceNightElf:
		character.PseudoStats.BaseDodgeChance += 0.01
		character.PseudoStats.MovementSpeedMultiplier *= 1.02

		RegisterTemporaryStatsOnUseCD(character,
			"Elune's Light",
			stats.Stats{
				stats.PhysicalCritPercent: 10,
				stats.SpellCritPercent:    10,
			},
			time.Second*15,
			SpellConfig{
				ActionID: ActionID{SpellID: 1259799},
				Cast: CastConfig{
					CD: Cooldown{
						Timer:    character.NewTimer(),
						Duration: time.Minute * 3,
					},
				},
			})
	case proto.Race_RaceOrc:
		applyWeaponSpecialization(character, "Axe Specialization", 20574, 1, proto.WeaponType_WeaponTypeAxe)

		bloodFuryID := ActionID{SpellID: 20572}

		bloodFuryAura := character.NewTemporaryStatMultiplierAura(Aura{
			Label:    "Blood Fury",
			ActionID: bloodFuryID,
			Duration: time.Second * 15,
		}, []StatMultiplier{
			{Stat: stats.AttackPower, Multiplier: 1.1},
			{Stat: stats.RangedAttackPower, Multiplier: 1.1},
			{Stat: stats.SpellDamage, Multiplier: 1.1},
			{Stat: stats.HealingPower, Multiplier: 1.1},
		})

		bloodFuryCD := Cooldown{
			Timer:    character.NewTimer(),
			Duration: time.Minute * 2,
		}
		bloodFuryAura.Icd = &bloodFuryCD

		bloodFury := character.RegisterSpell(SpellConfig{
			ActionID: bloodFuryID,
			Flags:    SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				CD: bloodFuryCD,
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				bloodFuryAura.Activate(sim)
			},

			RelatedSelfBuff: bloodFuryAura.Aura,
		})

		character.AddMajorCooldown(MajorCooldown{
			Spell:    bloodFury,
			Type:     CooldownTypeDPS,
			BuffAura: bloodFuryAura,
		})

		shatterCurseID := ActionID{SpellID: 1299026}

		shatterCurseAura := character.RegisterAura(Aura{
			Label:    "Shatter Curse",
			ActionID: shatterCurseID,
			Duration: time.Second * 8,
		})
		for _, school := range []stats.SchoolIndex{
			stats.SchoolIndexArcane,
			stats.SchoolIndexFire,
			stats.SchoolIndexFrost,
			stats.SchoolIndexHoly,
			stats.SchoolIndexNature,
			stats.SchoolIndexShadow,
		} {
			shatterCurseAura.AttachMultiplicativePseudoStatBuff(&character.PseudoStats.SchoolDamageTakenMultiplier[school], 0.85)
		}

		shatterCurse := character.RegisterSpell(SpellConfig{
			ActionID: shatterCurseID,
			Flags:    SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				DefaultCast: Cast{
					GCD: GCDDefault,
				},
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute * 3,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				shatterCurseAura.Activate(sim)
			},

			RelatedSelfBuff: shatterCurseAura,
		})

		character.AddMajorCooldown(MajorCooldown{
			Spell: shatterCurse,
			Type:  CooldownTypeSurvival,
		})
	case proto.Race_RaceTauren:
		character.MultiplyStat(stats.Health, 1.05)
		character.AddStat(stats.PhysicalHitPercent, 1)
		character.AddStat(stats.SpellHitPercent, 1)
	case proto.Race_RaceTroll:
		applyCreatureTypeSlaying(character, proto.MobType_MobTypeBeast)

		actionID := ActionID{SpellID: 20554}

		berserkingAura := character.RegisterAura(Aura{
			Label:    "Berserking",
			ActionID: actionID,
			Duration: time.Second * 10,
		}).
			AttachMultiplyAttackSpeed(1.1).
			AttachMultiplyCastSpeed(1.1)

		berserking := character.RegisterSpell(SpellConfig{
			ActionID: actionID,
			Flags:    SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute * 3,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				berserkingAura.Activate(sim)
			},

			RelatedSelfBuff: berserkingAura,
		})

		character.AddMajorCooldown(MajorCooldown{
			Spell: berserking,
			Type:  CooldownTypeDPS,
		})
	case proto.Race_RaceUndead:
		applyTouchOfTheGrave(character)
	case proto.Race_RaceHighOrderSkyborne:
		applySkyborneSharedRacials(character)

		energizedAura := character.RegisterAura(Aura{
			Label:    "Energized",
			ActionID: ActionID{SpellID: 1270842},
			Duration: time.Second * 15,
			OnGain: func(aura *Aura, sim *Simulation) {
				if aura.Unit.HasManaBar() {
					aura.Unit.MultiplyManaRegenSpeed(sim, 2)
				}
			},
			OnExpire: func(aura *Aura, sim *Simulation) {
				if aura.Unit.HasManaBar() {
					aura.Unit.MultiplyManaRegenSpeed(sim, 0.5)
				}
			},
		})

		// Client 1.60.1.70058: Read Ley Line has a 2 sec cast (SpellMisc CastingTimeIndex).
		character.RegisterSpell(SpellConfig{
			ActionID: ActionID{SpellID: 1259705},
			Flags:    SpellFlagAPL | SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				DefaultCast: Cast{
					GCD: GCDDefault,
					CastTime: time.Second * 2,
				},
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute * 2,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				energizedAura.Activate(sim)
			},

			RelatedSelfBuff: energizedAura,
		})
	case proto.Race_RaceWindshaperSkyborne:
		applySkyborneSharedRacials(character)

		elementalBlessingAura := character.RegisterAura(Aura{
			Label:    "Elemental Blessing",
			ActionID: ActionID{SpellID: 1259688},
			Duration: time.Second * 30,
			OnGain: func(aura *Aura, sim *Simulation) {
				aura.Unit.MultiplyMovementSpeed(sim, 1.1)
			},
			OnExpire: func(aura *Aura, sim *Simulation) {
				aura.Unit.MultiplyMovementSpeed(sim, 1/1.1)
			},
		})

		// Client 1.60.1.70058: Skysight has a 0.5 sec cast (SpellMisc CastingTimeIndex).
		character.RegisterSpell(SpellConfig{
			ActionID: ActionID{SpellID: 1259686},
			Flags:    SpellFlagAPL | SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				DefaultCast: Cast{
					GCD: GCDDefault,
					CastTime: time.Millisecond * 500,
				},
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute * 2,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				elementalBlessingAura.Activate(sim)
			},

			RelatedSelfBuff: elementalBlessingAura,
		})
	}
}

func applySkyborneSharedRacials(character *Character) {
	applyCreatureTypeSlaying(character, proto.MobType_MobTypeElemental)

	character.PseudoStats.AttackSpeedMultiplier *= 1.01
	character.PseudoStats.CastSpeedMultiplier *= 1.01
}

// Beast Slaying (troll 20557), Big Game Hunter (dwarf 1259721) and Elemental Insight (Skyborne
// 1259707) are one A_MOD_DAMAGE_DONE_VERSUS 5 each in client 1.60.1.70009, with no crit damage
// effect; the hunter's Humanoid/Monster Slaying carry that as a second aura (547), these don't.
func applyCreatureTypeSlaying(character *Character, mobType proto.MobType) {
	character.Env.RegisterPostFinalizeEffect(func() {
		for _, at := range character.AttackTables {
			if at.Defender.MobType == mobType {
				at.DamageDealtMultiplier *= 1.05
			}
		}
	})
}

func applyWeaponSpecialization(character *Character, label string, spellID int32, critPercent float64, weaponType proto.WeaponType) {
	hasWeaponEquipped := func() bool {
		mh, oh := character.MainHand(), character.OffHand()
		return !character.disableWeaponSpec && ((mh != nil && mh.WeaponType == weaponType) || (oh != nil && oh.WeaponType == weaponType))
	}

	aura := character.RegisterAura(Aura{
		Label:      label,
		ActionID:   ActionID{SpellID: spellID},
		Duration:   NeverExpires,
		BuildPhase: Ternary(hasWeaponEquipped(), CharacterBuildPhaseBase, CharacterBuildPhaseNone),
	}).
		AttachStatBuff(stats.PhysicalCritPercent, critPercent).
		AttachStatBuff(stats.SpellCritPercent, critPercent)

	if hasWeaponEquipped() {
		MakePermanent(aura)
	}

	character.RegisterItemSwapCallback(AllWeaponSlots(), func(sim *Simulation, _ proto.ItemSlot) {
		if hasWeaponEquipped() {
			aura.Activate(sim)
		} else {
			aura.Deactivate(sim)
		}
	})
}

func applyExpansiveMind(character *Character) {
	switch character.Class {
	case proto.Class_ClassWarrior:
		character.rageBar.maxRage *= 1.05
	case proto.Class_ClassRogue:
		character.energyBar.maxEnergy *= 1.05
	case proto.Class_ClassPriest, proto.Class_ClassMage, proto.Class_ClassWarlock:
		character.MultiplyStat(stats.Mana, 1.05)
	}
}

// The client scopes Eureka! to each class's damaging abilities with a spell family mask. Core cannot
// see those masks, so this takes every class ability that deals damage (or heals, for a priest).
func applyEureka(character *Character) {
	var spellID int32
	var resourceType proto.ResourceType
	procMask := ProcMaskSpecial
	// Build 70009 states -10% cost (effect 0, SPELLMOD_COST) on all five class variants; core
	// cannot import spelldata, so the value is mirrored here.
	const costReduction = 0.1

	switch character.Class {
	case proto.Class_ClassRogue:
		spellID, resourceType = 1259812, proto.ResourceType_ResourceTypeEnergy
	case proto.Class_ClassWarrior:
		spellID, resourceType = 1259813, proto.ResourceType_ResourceTypeRage
	case proto.Class_ClassMage:
		spellID, resourceType = 1259817, proto.ResourceType_ResourceTypeMana
	case proto.Class_ClassWarlock:
		spellID, resourceType = 1259821, proto.ResourceType_ResourceTypeMana
	case proto.Class_ClassPriest:
		spellID, resourceType = 1259823, proto.ResourceType_ResourceTypeMana
		procMask |= ProcMaskSpellHealing
	default:
		return
	}

	actionID := ActionID{SpellID: spellID}
	const anyClassSpell = math.MaxInt64

	costMod := character.AddDynamicMod(SpellModConfig{
		Kind:         SpellMod_PowerCost_Pct,
		ClassMask:    anyClassSpell,
		ProcMask:     procMask,
		ResourceType: resourceType,
		FloatValue:   -costReduction,
	})
	damageMod := character.AddDynamicMod(SpellModConfig{
		Kind:       SpellMod_DamageDone_Pct,
		ClassMask:  anyClassSpell,
		ProcMask:   procMask,
		FloatValue: 0.1,
	})

	aura := character.RegisterAura(Aura{
		Label:     "Eureka!",
		ActionID:  actionID,
		Duration:  time.Second * 15,
		MaxStacks: 3,
		OnGain: func(aura *Aura, sim *Simulation) {
			aura.SetStacks(sim, 3)
			costMod.Activate()
			damageMod.Activate()
		},
		OnExpire: func(_ *Aura, _ *Simulation) {
			costMod.Deactivate()
			damageMod.Deactivate()
		},
		OnCastComplete: func(aura *Aura, sim *Simulation, spell *Spell) {
			if spell.Matches(anyClassSpell) && spell.ProcMask.Matches(procMask) {
				aura.RemoveStack(sim)
			}
		},
	})

	spell := character.RegisterSpell(SpellConfig{
		ActionID: actionID,
		Flags:    SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: time.Minute * 2,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},

		RelatedSelfBuff: aura,
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell: spell,
		Type:  CooldownTypeDPS,
	})
}

func applyTouchOfTheGrave(character *Character) {
	auraID, procChance := int32(1260201), 0.1
	switch character.Class {
	case proto.Class_ClassWarrior, proto.Class_ClassPaladin, proto.Class_ClassRogue:
		auraID, procChance = 1260189, 0.05
	}

	drainID := ActionID{SpellID: 1260198}
	healthMetrics := character.NewHealthMetrics(drainID)

	drain := character.RegisterSpell(SpellConfig{
		ActionID:    drainID,
		SpellSchool: SpellSchoolShadow,
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskEmpty,
		Flags:       SpellFlagProc | SpellFlagPassiveSpell | SpellFlagIgnoreAttackerModifiers | SpellFlagNoSpellMods,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			result := spell.CalcAndDealDamage(sim, target, 0.05*spell.Unit.MaxHealth(), spell.OutcomeMagicHit)
			if result.Landed() && spell.Unit.HasHealthBar() {
				spell.Unit.GainHealth(sim, result.Damage, healthMetrics)
			}
		},
	})

	character.MakeProcTriggerAura(ProcTrigger{
		Name:       "Touch of the Grave",
		ActionID:   ActionID{SpellID: auraID},
		Callback:   CallbackOnSpellHitDealt,
		ProcMask:   ProcMaskMelee | ProcMaskRanged | ProcMaskSpellDamage,
		Outcome:    OutcomeLanded,
		ProcChance: procChance,
		ICD:        time.Second,
		Handler: func(sim *Simulation, spell *Spell, result *SpellResult) {
			drain.Cast(sim, result.Target)
		},
	})
}
