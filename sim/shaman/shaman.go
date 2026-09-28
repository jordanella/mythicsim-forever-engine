package shaman

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

var TalentTreeSizes = [3]int{16, 18, 16}

const (
	SpellFlagShamanSpell = core.SpellFlagAgentReserved1
	SpellFlagShock       = core.SpellFlagAgentReserved2
	SpellFlagInstant     = core.SpellFlagAgentReserved3
	SpellFlagFocusable   = core.SpellFlagAgentReserved4
)

func NewShaman(character *core.Character, talents string, selfBuffs SelfBuffs) *Shaman {
	shaman := &Shaman{
		Character: *character,
		Talents:   &proto.ShamanTalents{},
		Totems:    &proto.ShamanTotems{},
		SelfBuffs: selfBuffs,
	}

	core.FillTalentsProto(shaman.Talents.ProtoReflect(), talents, TalentTreeSizes)

	// Add Shaman stat dependencies
	shaman.AddStatDependency(stats.BonusArmor, stats.Armor, 1)
	shaman.AddStatDependency(stats.Agility, stats.PhysicalCritPercent, core.CritPerAgiMaxLevel[shaman.Class])
	shaman.AddStatDependency(stats.Agility, stats.DodgeRating, 1.0/25*core.DodgeRatingPerDodgePercent)
	shaman.EnableManaBarWithModifier()

	shaman.AddStatDependency(stats.Strength, stats.AttackPower, 2.0)

	shaman.WindfuryAPBonus = 333.0 // 16361, Windfury Weapon rank 4

	return shaman
}

// A hard cast holds the melee swing (core.AutoAttacks.HoldMeleeForCast): on the Forever beta a
// Lightning Bolt cast between swings resets the swing timer as it completes, and a swing due during
// the cast lands as it completes. A bolt Maelstrom Weapon makes instant leaves the swing alone.
func (shaman *Shaman) holdMeleeForCast(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
	if castTime := shaman.ApplyCastSpeedForSpell(cast.CastTime, spell); castTime > 0 {
		shaman.AutoAttacks.HoldMeleeForCast(sim, sim.CurrentTime+castTime)
	}
}

func (shaman *Shaman) GetImbueProcMask(imbue proto.ShamanImbue) core.ProcMask {
	var mask core.ProcMask
	if shaman.SelfBuffs.ImbueMH == imbue || shaman.SelfBuffs.ImbueMHSwap == imbue {
		mask |= core.ProcMaskMeleeMH
	}
	if shaman.SelfBuffs.ImbueOH == imbue {
		mask |= core.ProcMaskMeleeOH
	}
	return mask
}

// Which buffs this shaman is using.
type SelfBuffs struct {
	ShieldProcrate float64
	ImbueMH        proto.ShamanImbue
	ImbueOH        proto.ShamanImbue
	ImbueMHSwap    proto.ShamanImbue
	ImbueOHSwap    proto.ShamanImbue
}

// Indexes into NextTotemDrops for self buffs
const (
	AirTotem int = iota
	EarthTotem
	FireTotem
	WaterTotem
)

// Shaman represents a shaman character.
type Shaman struct {
	core.Character

	WindfuryAPBonus float64

	Talents   *proto.ShamanTalents
	SelfBuffs SelfBuffs

	Totems *proto.ShamanTotems

	// The expiration time of each totem (earth, air, fire, water).
	TotemExpirations [4]time.Duration

	LightningBoltOverloads map[int32]*core.Spell

	ChainLightningOverloads map[int32][]*core.Spell

	// Added to Chain Lightning's per-bounce damage multiplier (Gift of the Gathering Storm).
	ChainLightningBounceBonus float64

	Stormstrike           *core.Spell
	StormstrikeCastResult *core.SpellResult

	LightningShieldAura *core.Aura
	WaterShieldAura     *core.Aura
	ShieldSelfProcSpell *core.Spell

	EarthShock *core.Spell
	FlameShock *core.Spell
	FrostShock *core.Spell

	StormStrikeDebuffAuras core.AuraArray

	TotemOfWrath *core.Spell
	MagmaTotem   *core.Spell
	// Always nil: its registrar is commented out in totems.go, upstream of the fork.
	HealingStreamTotem *core.Spell
	SearingTotem       *core.Spell
	TremorTotem        *core.Spell
	SearingReplaced    bool // Used for cancelling searing dot if the totem is replaced during prepull

	EarthTotemAura *core.Aura
	WaterTotemAura *core.Aura
	AirTotemAura   *core.Aura
}

// Implemented by each Shaman spec.
type ShamanAgent interface {
	core.Agent

	// The Shaman controlled by this Agent.
	GetShaman() *Shaman
}

func (shaman *Shaman) GetCharacter() *core.Character {
	return &shaman.Character
}

func (shaman *Shaman) AddRaidBuffs(raidBuffs *proto.RaidBuffs) {
}

// The talented Mana Tide Totem (16190) is the party buff's totem, dropped by this shaman.
// ponytail: it does not take Mana Spring's water slot for its 12 s; model the slot if a sim ever
// shows the overlap mattering.
func (shaman *Shaman) AddPartyBuffs(partyBuffs *proto.PartyBuffs) {
	if shaman.Talents.ManaTideTotem {
		partyBuffs.ManaTideTotems++
	}
}

func (shaman *Shaman) Initialize() {
	shaman.registerChainLightningSpell()
	shaman.registerLightningBoltSpell()
	shaman.registerShieldsSpells()
	shaman.registerMagmaTotemSpell()
	shaman.registerSearingTotemSpell()
	shaman.registerFireNovaSpell()
	shaman.registerWindfuryTotemSpell()
	shaman.registerStrengthOfEarthTotemSpell()
	shaman.registerGraceOfAirTotemSpell()
	shaman.registerManaSpringTotemSpell()
	shaman.registerShocks()
}

func (shaman *Shaman) ApplyTalents() {
	shaman.registerElementalTalents()
	shaman.registerEnhancementTalents()
	shaman.registerRestorationTalents()
}

func (shaman *Shaman) Reset(sim *core.Simulation) {
	shaman.TotemExpirations[FireTotem] = -10 * time.Hour
	shaman.TotemExpirations[AirTotem] = -10 * time.Hour
	shaman.TotemExpirations[EarthTotem] = -10 * time.Hour
	shaman.TotemExpirations[WaterTotem] = -10 * time.Hour
}

func (shaman *Shaman) OnEncounterStart(sim *core.Simulation) {
	shaman.startShieldProcPeriodicAction(sim)
}

func (shaman *Shaman) GetOverloadChance() float64 {
	if shaman.Talents.LightningOverload == 0 {
		return 0.0
	}
	return spellData.LightningOverload.FractionAt(shaman.Talents.LightningOverload)
}

const (
	SpellMaskNone             int64 = 0
	SpellMaskFlameShockDirect int64 = 1 << iota
	SpellMaskFlameShockDot
	SpellMaskLightningBolt
	SpellMaskLightningBoltOverload
	SpellMaskChainLightning
	SpellMaskChainLightningOverload
	SpellMaskEarthShock
	SpellMaskLightningShield
	SpellMaskMagmaTotem
	SpellMaskSearingTotem
	SpellMaskFireNova
	SpellMaskFlametongueTotem
	SpellMaskStormstrikeCast
	SpellMaskStormstrikeDamage
	SpellMaskEarthShield
	SpellMaskFrostShock
	SpellMaskFlametongueWeapon
	SpellMaskWindfuryWeapon
	SpellMaskFrostbrandWeapon
	SpellMaskRockbiterWeapon
	SpellMaskElementalMastery
	SpellMaskShamanisticRage
	SpellMaskBasicTotem
	SpellMaskShieldSelfProc
	SpellMaskLavaBurst

	SpellMaskStormstrike = SpellMaskStormstrikeCast | SpellMaskStormstrikeDamage
	SpellMaskFlameShock  = SpellMaskFlameShockDirect | SpellMaskFlameShockDot
	SpellMaskFire        = SpellMaskFlameShock | SpellMaskFireNova | SpellMaskLavaBurst
	SpellMaskNature      = SpellMaskLightningBolt | SpellMaskLightningBoltOverload | SpellMaskChainLightning | SpellMaskChainLightningOverload | SpellMaskEarthShock
	SpellMaskFrost       = SpellMaskFrostShock
	SpellMaskOverload    = SpellMaskLightningBoltOverload | SpellMaskChainLightningOverload
	SpellMaskShock       = SpellMaskFlameShock | SpellMaskEarthShock | SpellMaskFrostShock
	SpellMaskFireTotem   = SpellMaskMagmaTotem | SpellMaskSearingTotem
	SpellMaskTotem       = SpellMaskFireTotem | SpellMaskBasicTotem
	SpellMaskImbue       = SpellMaskFrostbrandWeapon | SpellMaskWindfuryWeapon | SpellMaskFlametongueWeapon | SpellMaskRockbiterWeapon
)

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
