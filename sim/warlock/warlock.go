package warlock

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

var TalentTreeSizes = [3]int{17, 19, 16}

type Warlock struct {
	core.Character
	Talents *proto.WarlockTalents
	Options *proto.WarlockOptions

	// Base Spells
	Corruption  *core.Spell
	DrainLife   *core.Spell
	Hellfire    *core.Spell
	Immolate    *core.Spell
	Incinerate  *core.Spell
	SearingPain *core.Spell
	ShadowBolt  *core.Spell
	Soulfire    *core.Spell

	LifeTap *core.Spell

	// Curses and banes. Forever splits them: a curse and a bane hold the target at the same time,
	// and each replaces only its own kind.
	CurseOfAgony             *core.Spell
	CurseOfDoom              *core.Spell
	CurseOfElements          *core.Spell
	CurseOfElementsAuras     core.AuraArray
	CurseOfRecklessness      *core.Spell
	CurseOfRecklessnessAuras core.AuraArray

	// Talent Tree Spells
	AmplifyCurse *core.Spell
	Conflagrate  *core.Spell
	Shadowburn   *core.Spell
	SiphonLife   *core.Spell
	Wrack        *core.Spell

	// Auras
	AmplifyCurseAura        *core.Aura
	DecimationAura          *core.Aura
	NightfallProcAura       *core.Aura
	ImprovedShadowBoltAuras core.AuraArray
	DemonicBrandAuras       core.AuraArray
	SoulLinkAura            *core.Aura
	MasterDemonologistAura  *core.Aura

	// Pets
	ActivePet  *WarlockPet
	BasePets   []*WarlockPet
	Felhunter  *WarlockPet
	Imp        *WarlockPet
	Succubus   *WarlockPet
	Voidwalker *WarlockPet

	// Armors
	DemonArmor *core.Aura

	currentActiveCurse core.AuraArray
	currentActiveBane  core.AuraArray

	CorruptionTickBaseDamage float64
	ImmolateTickBaseDamage   float64
	T5_4PC_Multiplier        map[int32]map[*core.Spell]float64
}

func (warlock *Warlock) GetCharacter() *core.Character {
	return &warlock.Character
}

func (warlock *Warlock) GetWarlock() *Warlock {
	return warlock
}

func RegisterWarlock() {
	core.RegisterAgentFactory(
		proto.Player_Warlock{},
		proto.Spec_SpecWarlock,
		func(character *core.Character, options *proto.Player, raid *proto.Raid) core.Agent {
			return NewWarlock(character, options, options.GetWarlock().Options.ClassOptions, raid)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_Warlock)
			if !ok {
				panic("Invalid spec value for Warlock!")
			}
			player.Spec = playerSpec
		},
	)
}

func (warlock *Warlock) Initialize() {
	// Curses and banes
	warlock.registerCurseOfElements()
	warlock.registerCurseOfDoom()
	warlock.registerCurseOfAgony()
	warlock.registerCurseOfRecklessness()

	warlock.registerCorruption()
	warlock.registerDeathCoil()
	warlock.registerDrainLife()
	warlock.registerHellfire()
	warlock.registerImmolate()
	warlock.registerIncinerate()
	warlock.registerLifeTap()
	warlock.registerRainOfFire()
	warlock.registerShadowBolt()
	warlock.registerSearingPain()
	warlock.registerSiphonLifeSpell()
	warlock.registerSoulfire()
	warlock.registerWrack()

	warlock.registerArmors()
	warlock.registerPetAbilities()

	warlock.PseudoStats.SelfHealingMultiplier = 1.0
}

func (warlock *Warlock) AddRaidBuffs(raidBuffs *proto.RaidBuffs) {
}

// A summoned Imp gives the party Blood Pact (11767). Forever's Improved Imp (18694) no longer raises it.
func (warlock *Warlock) AddPartyBuffs(partyBuffs *proto.PartyBuffs) {
	if warlock.Options.Summon == proto.WarlockOptions_Imp && !warlock.Options.SacrificeSummon {
		partyBuffs.BloodPact = true
	}
}

func (warlock *Warlock) Reset(sim *core.Simulation) {
	warlock.currentActiveCurse = make(core.AuraArray, len(sim.Environment.AllUnits))
	warlock.currentActiveBane = make(core.AuraArray, len(sim.Environment.AllUnits))
}

func (warlock *Warlock) OnEncounterStart(sim *core.Simulation) {}

func NewWarlock(character *core.Character, options *proto.Player, warlockOptions *proto.WarlockOptions, raid *proto.Raid) *Warlock {
	warlock := &Warlock{
		Character: *character,
		Talents:   &proto.WarlockTalents{},
		Options:   warlockOptions,
	}

	core.FillTalentsProto(warlock.Talents.ProtoReflect(), options.TalentsString, TalentTreeSizes)

	if raid.Debuffs != nil {
		switch warlock.Options.CurseOptions {
		case proto.WarlockOptions_Elements:
			raid.Debuffs.CurseOfElements = false
		case proto.WarlockOptions_Recklessness:
			raid.Debuffs.CurseOfRecklessness = false
		}
	}

	warlock.EnableManaBar()
	warlock.AddStatDependency(stats.Strength, stats.AttackPower, 1)
	warlock.AddStatDependency(stats.Agility, stats.PhysicalCritPercent, core.CritPerAgiMaxLevel[character.Class])

	if !warlock.Options.SacrificeSummon {
		warlock.registerPets()
	}

	warlock.T5_4PC_Multiplier = make(map[int32]map[*core.Spell]float64)

	return warlock
}

func (warlock *Warlock) AfflictionCount(target *core.Unit) float64 {
	return float64(len(target.GetAurasWithTag("Affliction")))
}

// Forever keeps a curse and a bane on the target at once, so each kind only replaces its own.
// The caller still starts its own effect: a dot wants Apply, a plain debuff wants Activate.
func (warlock *Warlock) takeCurseSlot(sim *core.Simulation, target *core.Unit, aura *core.Aura) {
	warlock.takeSlot(sim, warlock.currentActiveCurse, target, aura)
}

func (warlock *Warlock) takeBaneSlot(sim *core.Simulation, target *core.Unit, aura *core.Aura) {
	warlock.takeSlot(sim, warlock.currentActiveBane, target, aura)
}

func (warlock *Warlock) takeSlot(sim *core.Simulation, slot core.AuraArray, target *core.Unit, aura *core.Aura) {
	if active := slot.Get(target); active != nil && active != aura {
		active.Deactivate(sim)
	}
	slot[target.UnitIndex] = aura
}

// Agent is a generic way to access underlying warlock on any of the agents.
type WarlockAgent interface {
	GetWarlock() *Warlock
}

const (
	WarlockSpellFlagNone    int64 = 0
	WarlockSpellConflagrate int64 = 1 << iota
	WarlockSpellShadowBolt
	WarlockSpellImmolate
	WarlockSpellImmolateDot
	WarlockSpellIncinerate
	WarlockSpellSoulFire
	WarlockSpellShadowBurn
	WarlockSpellLifeTap
	WarlockSpellCorruption
	WarlockSpellCurseOfAgony
	WarlockSpellCurseOfElements
	WarlockSpellDrainLife
	WarlockSpellHellfire
	WarlockSpellImmolationAura
	WarlockSpellSearingPain
	WarlockSpellSummonDoomguard
	WarlockSpellDoomguardDoomBolt
	WarlockSpellSummonImp
	WarlockSpellImpFireBolt
	WarlockSpellSummonFelhunter
	WarlockSpellFelHunterShadowBite
	WarlockSpellSummonSuccubus
	WarlockSpellSuccubusLashOfPain
	WarlockSpellVoidwalkerTorment
	WarlockSpellSummonInfernal
	WarlockSpellRainOfFire
	WarlockSpellCurseOfDoom
	WarlockSpellCurseOfRecklessness
	WarlockSpellCurseOfWeakness
	WarlockSpellSiphonLife
	WarlockSpellDrainSoul
	WarlockSpellDeathCoil
	WarlockSpellWrack
	WarlockSpellAll int64 = 1<<iota - 1

	WarlockShadowDamage = WarlockSpellCorruption | WarlockSpellDrainLife | WarlockSpellCurseOfAgony |
		WarlockSpellCurseOfDoom | WarlockSpellShadowBolt | WarlockSpellShadowBurn | WarlockSpellSiphonLife |
		WarlockSpellDeathCoil | WarlockSpellDrainSoul | WarlockSpellWrack

	WarlockPeriodicShadowDamage = WarlockSpellCorruption | WarlockSpellDrainLife | WarlockSpellCurseOfAgony |
		WarlockSpellCurseOfDoom | WarlockSpellSiphonLife | WarlockSpellDrainSoul | WarlockSpellWrack

	WarlockFireDamage = WarlockSpellConflagrate | WarlockSpellImmolate | WarlockSpellIncinerate | WarlockSpellSoulFire |
		WarlockSpellSearingPain | WarlockSpellImmolateDot | WarlockSpellHellfire | WarlockSpellRainOfFire

	WarlockDoT = WarlockSpellCorruption |
		WarlockSpellDrainLife | WarlockSpellCurseOfAgony | WarlockSpellImmolateDot

	WarlockSummonSpells = WarlockSpellSummonImp | WarlockSpellSummonSuccubus | WarlockSpellSummonFelhunter

	WarlockAllSummons = WarlockSummonSpells | WarlockSpellSummonInfernal | WarlockSpellSummonDoomguard

	WarlockContagionSpells = WarlockSpellCurseOfAgony | WarlockSpellCorruption

	// The drain effects Improved Drains and Soul Siphon pay out on.
	WarlockDrainSpells = WarlockSpellDrainLife | WarlockSpellDrainSoul | WarlockSpellWrack

	// Nightfall rolls off the periodic damage of these, per the beta client's talent text.
	WarlockNightfallSpells = WarlockSpellCorruption | WarlockSpellDrainLife | WarlockSpellDrainSoul | WarlockSpellWrack

	WarlockCurses = WarlockSpellCurseOfElements | WarlockSpellCurseOfRecklessness | WarlockSpellCurseOfWeakness

	WarlockBanes = WarlockSpellCurseOfAgony | WarlockSpellCurseOfDoom

	WarlockAfflictionSpells = WarlockSpellCorruption | WarlockSpellCurseOfAgony | WarlockSpellCurseOfDoom |
		WarlockSpellCurseOfRecklessness | WarlockSpellCurseOfElements | WarlockSpellDrainLife | WarlockSpellDrainSoul |
		WarlockSpellDeathCoil | WarlockSpellSiphonLife | WarlockSpellWrack | WarlockSpellLifeTap

	WarlockDemonologySpells = WarlockAllSummons

	WarlockDestructionSpells = WarlockSpellHellfire | WarlockSpellImmolate | WarlockSpellImmolateDot |
		WarlockSpellIncinerate | WarlockSpellRainOfFire | WarlockSpellSearingPain |
		WarlockSpellShadowBolt | WarlockSpellSoulFire | WarlockSpellConflagrate | WarlockSpellShadowBurn
)

// Called to handle custom resources
type WarlockSpellCastedCallback func(resultList core.SpellResultSlice, spell *core.Spell, sim *core.Simulation)

// The tick outcome the family table picked: a crit roll where the client marks Periodic Can Crit, a
// plain tick otherwise, and never a per-tick hit roll. The store's TickOutcome adds that hit roll on
// magic ticks, which would move every dot.
func periodicTickOutcome(s *spelldata.Spell, dot *core.Dot) core.OutcomeApplier {
	switch {
	case s.PeriodicCanCrit() && s.DefenseTypeCore() == core.DefenseTypeMagic:
		return dot.Spell.OutcomeTickMagicCrit
	case s.PeriodicCanCrit():
		return dot.Spell.OutcomeTickPhysicalCrit
	default:
		return dot.OutcomeTick
	}
}
