package warrior

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

var TalentTreeSizes = [3]int{17, 17, 18}

type WarriorInputs struct {
	UseBattleShout bool
	DefaultStance  proto.WarriorStance

	StartingRage   float64
	QueueDelay     int32
	StanceSnapshot bool
	HasBsT2        bool
}

const (
	SpellMaskNone int64 = 0
	// Abilities that don't cost rage and aren't attacks
	SpellMaskBattleShout int64 = 1 << iota
	SpellMaskBerserkerRage
	SpellMaskRecklessness
	SpellMaskDeathWish
	SpellMaskRetaliation
	SpellMaskRetaliationHit
	SpellMaskShieldWall
	SpellMaskLastStand
	SpellMaskCharge
	SpellMaskIntercept
	SpellMaskDemoralizingShout

	// Stances
	SpellMaskBattleStance
	SpellMaskBerserkerStance
	SpellMaskDefensiveStance

	// Special attacks
	SpellMaskRend
	SpellMaskDeepWounds
	SpellMaskSweepingStrikes
	SpellMaskSweepingStrikesHit
	SpellMaskSweepingStrikesNormalizedHit
	SpellMaskHeroicStrike
	SpellMaskCleave
	SpellMaskExecute
	SpellMaskOverpower
	SpellMaskRevenge
	SpellMaskSlam
	SpellMaskSunderArmor
	SpellMaskThunderClap
	SpellMaskWhirlwind
	SpellMaskWhirlwindOh
	SpellMaskShieldSlam
	SpellMaskConcussionBlow
	SpellMaskShieldBash
	SpellMaskBloodthirst
	SpellMaskMortalStrike
	SpellMaskShieldBlock
	SpellMaskHamstring
	SpellMaskPummel
	SpellMaskMockingBlow
	SpellMaskChallengingShout
	SpellMaskIntimidatingShout
	SpellMaskDisarm
	SpellMaskTaunt
	SpellMaskVictoryRush
	SpellMaskSpearingStrike

	WarriorSpellLast
	WarriorSpellsAll = WarriorSpellLast<<1 - 1

	SpellMaskDirectDamageSpells = SpellMaskSweepingStrikesHit | SpellMaskSweepingStrikesNormalizedHit |
		SpellMaskCleave | SpellMaskExecute | SpellMaskHeroicStrike | SpellMaskOverpower |
		SpellMaskRevenge | SpellMaskSlam | SpellMaskShieldBash | SpellMaskSunderArmor |
		SpellMaskThunderClap | SpellMaskWhirlwind | SpellMaskWhirlwindOh | SpellMaskShieldSlam |
		SpellMaskBloodthirst | SpellMaskMortalStrike | SpellMaskIntercept | SpellMaskRetaliationHit |
		SpellMaskMockingBlow | SpellMaskVictoryRush | SpellMaskSpearingStrike |
		SpellMaskHamstring | SpellMaskPummel

	SpellMaskDamageSpells = SpellMaskDirectDamageSpells | SpellMaskDeepWounds | SpellMaskRend

	SpellMaskOffensiveAbilities = SpellMaskHeroicStrike | SpellMaskRend | SpellMaskShieldBash |
		SpellMaskCleave | SpellMaskDisarm | SpellMaskWhirlwind | SpellMaskSunderArmor | SpellMaskSlam |
		SpellMaskHamstring | SpellMaskExecute | SpellMaskPummel | SpellMaskRevenge | SpellMaskOverpower |
		SpellMaskThunderClap | SpellMaskMockingBlow | SpellMaskMortalStrike | SpellMaskConcussionBlow |
		SpellMaskShieldSlam | SpellMaskRetaliation | SpellMaskIntercept | SpellMaskBloodthirst
	// Focused Rage's class mask (29787) adds these to the offensive abilities and leaves out Retaliation.
	SpellMaskFocusedRage = SpellMaskOffensiveAbilities&^SpellMaskRetaliation | SpellMaskDemoralizingShout |
		SpellMaskDeathWish | SpellMaskSweepingStrikes | SpellMaskSpearingStrike | SpellMaskChallengingShout |
		SpellMaskIntimidatingShout
	SpellMaskShouts = SpellMaskBattleShout | SpellMaskDemoralizingShout | SpellMaskIntimidatingShout | SpellMaskChallengingShout
)

type Warrior struct {
	core.Character

	ClassSpellScaling float64

	Talents *proto.WarriorTalents

	WarriorInputs

	// Current state
	Stance                 Stance
	thunderClapEffectBonus float64

	BattleShout       *core.Spell
	DemoralizingShout *core.Spell
	BattleStance      *core.Spell
	DefensiveStance   *core.Spell
	BerserkerStance   *core.Spell

	Rend                            *core.Spell
	DeepWounds                      *core.Spell
	MortalStrike                    *core.Spell
	SweepingStrikesNormalizedAttack *core.Spell

	HeroicStrike *core.Spell
	Cleave       *core.Spell

	curQueueAura       *core.Aura
	curQueuedAutoSpell *core.Spell
	// The sim often re-enables Heroic Strike in an unrealistic amount of time, which can cause an
	// unrealistic immediate double hit around extra attacks.
	queuedRealismICD *core.Cooldown

	MockingBlow       *core.Spell
	ChallengingShout  *core.Spell
	IntimidatingShout *core.Spell
	Disarm            *core.Spell
	Taunt             *core.Spell
	VictoryRush       *core.Spell

	EnrageAura *core.Aura

	SweepingStrikesAura *core.Aura
	OverpowerAura       *core.Aura

	DemoralizingShoutAuras core.AuraArray
	SunderArmorAuras       core.AuraArray
}

func (warrior *Warrior) GetCharacter() *core.Character {
	return &warrior.Character
}

func (warrior *Warrior) AddRaidBuffs(raidBuffs *proto.RaidBuffs) {
}

func (warrior *Warrior) AddPartyBuffs(_ *proto.PartyBuffs) {
}

func (warrior *Warrior) Initialize() {
	warrior.registerRecklessness()
	warrior.registerShieldWall()
	warrior.registerRetaliation()

	warrior.registerBerserkerRage()
	warrior.registerBloodrage()
	warrior.registerCharge()
	warrior.registerIntercept()
	warrior.registerPummel()
	warrior.registerHamstring()
	warrior.registerDisarm()
	warrior.registerTaunt()

	warrior.registerRend()
	warrior.registerSunderArmor()
	warrior.registerHeroicStrike()
	warrior.registerCleave()
	warrior.registerOverpower()
	warrior.registerSlam()
	warrior.registerWhirlwind()
	warrior.registerExecute()
	warrior.registerThunderClap()
	warrior.registerRevenge()
	warrior.registerShieldBlock()
	warrior.registerShieldBash()
	warrior.registerMockingBlow()
	warrior.registerVictoryRush()

	warrior.registerStances()
	warrior.registerBattleShout()
	warrior.registerDemoralizingShout()
	warrior.registerChallengingShout()
	warrior.registerIntimidatingShout()
}

func (warrior *Warrior) Reset(_ *core.Simulation) {
	warrior.curQueueAura = nil
	warrior.curQueuedAutoSpell = nil

	switch warrior.DefaultStance {
	case proto.WarriorStance_WarriorStanceBattle:
		warrior.Stance = BattleStance
	case proto.WarriorStance_WarriorStanceDefensive:
		warrior.Stance = DefensiveStance
	case proto.WarriorStance_WarriorStanceBerserker:
		warrior.Stance = BerserkerStance
	}
}

func (warrior *Warrior) OnEncounterStart(sim *core.Simulation) {}

func (warrior *Warrior) GetMainHandType() proto.HandType {
	mh := warrior.GetMHWeapon()

	if mh != nil && (mh.HandType == proto.HandType_HandTypeTwoHand) {
		return proto.HandType_HandTypeTwoHand
	}

	return proto.HandType_HandTypeOneHand
}

func NewWarrior(character *core.Character, options *proto.WarriorOptions, talents string, inputs WarriorInputs) *Warrior {
	warrior := &Warrior{
		Character:     *character,
		Talents:       &proto.WarriorTalents{},
		WarriorInputs: inputs,
	}
	core.FillTalentsProto(warrior.Talents.ProtoReflect(), talents, TalentTreeSizes)

	warrior.EnableRageBar(core.RageBarOptions{
		MaxRage:            100,
		BaseRageMultiplier: 1,
		StartingRage:       inputs.StartingRage,
	})

	warrior.EnableAutoAttacks(warrior, core.AutoAttackOptions{
		MainHand:       warrior.WeaponFromMainHand(),
		OffHand:        warrior.WeaponFromOffHand(),
		AutoSwingMelee: true,
		ReplaceMHSwing: warrior.TryHSOrCleave,
	})

	warrior.PseudoStats.CanParry = true
	// Base parry and block (5%, less 0.04% a level of attacker skill above the defense) come from
	// the attack table (core/target.go); adding 5% here counted them twice. Dodge is Classic's at
	// level 60, as on master: no base dodge, 20 Agility a percent. Worth a check on a beta
	// character sheet.

	warrior.AddStatDependency(stats.Strength, stats.AttackPower, 2)
	// Block value from Strength is Classic's Str/20 - 1, as master (vmangos GetShieldBlockValue).
	warrior.AddStatDependency(stats.Strength, stats.BlockValue, 1/20.0)
	warrior.AddStat(stats.BlockValue, -1)
	warrior.AddStatDependency(stats.Agility, stats.PhysicalCritPercent, core.CritPerAgiMaxLevel[character.Class])
	warrior.AddStatDependency(stats.Agility, stats.DodgeRating, core.CritPerAgiMaxLevel[character.Class]*core.DodgeRatingPerDodgePercent)
	warrior.AddStatDependency(stats.BonusArmor, stats.Armor, 1)

	warrior.queuedRealismICD = &core.Cooldown{
		Timer:    warrior.NewTimer(),
		Duration: time.Millisecond * time.Duration(inputs.QueueDelay),
	}

	return warrior
}

func (warrior *Warrior) CastNormalizedSweepingStrikesAttack(results core.SpellResultSlice, sim *core.Simulation) {
	if warrior.SweepingStrikesAura != nil && warrior.SweepingStrikesAura.IsActive() {
		for _, result := range results {
			if result.Landed() {
				warrior.SweepingStrikesNormalizedAttack.Cast(sim, warrior.Env.NextActiveTargetUnit(result.Target))
				warrior.SweepingStrikesAura.RemoveStack(sim)
				break
			}
		}
	}
}

// Agent is a generic way to access underlying warrior on any of the agents.
type WarriorAgent interface {
	GetWarrior() *Warrior
}

// The recovery the ability waits out. A warrior ability can state it on a shared category -
// Bloodthirst and Mortal Strike both run off category 971 - and leave its own column at zero, so the
// cooldown is whichever of the two the client filled in, as the family table read it.
func cooldownOf(s *spelldata.Spell) time.Duration {
	return max(s.Cooldown(), s.CategoryCooldown())
}
