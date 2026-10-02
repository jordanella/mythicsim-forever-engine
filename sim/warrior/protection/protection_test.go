package protection

import (
	"testing"

	"github.com/wowsims/forever/sim/arenalib"
	"github.com/wowsims/forever/sim/common"
	_ "github.com/wowsims/forever/sim/common" // imported to get item effects included.
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func init() {
	RegisterProtectionWarrior()
	common.RegisterAllEffects()
}

// The two builds our Forever sim ships for the tank: its default Protection build and 1/0/50.
var DefaultProtectionTalents = "31--255533123330201351"
var ProtectionTalents = "1--255533123331221351"

func TestProtectionWarrior(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:           proto.Class_ClassWarrior,
			Race:            proto.Race_RaceOrc,
			OtherRaces:      []proto.Race{proto.Race_RaceHuman},
			GearSet:         DefaultGear,
			Talents:         DefaultProtectionTalents,
			OtherTalentSets: []core.TalentsCombo{{Label: "Protection", Talents: ProtectionTalents}},
			Consumables:     DefaultConsumables,
			SpecOptions:     core.SpecOptionsCombo{Label: "Protection", SpecOptions: DefaultOptions},
			Rotation:        core.GetAplRotation("../../../ui/specs/warrior/protection/apls", "protection"),

			IndividualBuffs: core.FullTankIndividualBuffs,
			// Without this the boss never attacks, so nothing the spec does in response to being hit
			// can fire and the damage taken metrics are all zero.
			IsTank:          true,
			InFrontOfTarget: true,

			ItemFilter: core.ItemFilter{
				ArmorType: proto.ArmorType_ArmorTypePlate,
				WeaponTypes: []proto.WeaponType{
					proto.WeaponType_WeaponTypeAxe,
					proto.WeaponType_WeaponTypeSword,
					proto.WeaponType_WeaponTypeMace,
					proto.WeaponType_WeaponTypeDagger,
					proto.WeaponType_WeaponTypeFist,
					proto.WeaponType_WeaponTypeShield,
				},
				HandTypes: []proto.HandType{
					proto.HandType_HandTypeMainHand,
					proto.HandType_HandTypeOffHand,
					proto.HandType_HandTypeOneHand,
				},
				// Enchant effects sim/common registers that the generated database has no row for.
				EnchantBlacklist: []int32{963, 1899, 1900, 2523, 2613, 2621, 2673, 2714, 2722, 2723, 2724, 2939, 2940, 3225, 3273},
			},
		},
	}))
}

// A statless axe and a shield. The generated item database does not carry the Forever gear our sim
// tests with; Shield Slam and Shield Block need the shield.
var DefaultGear = core.GearSetCombo{Label: "AxeAndShield", GearSet: axeAndShield()}

func axeAndShield() *proto.EquipmentSpec {
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: 15240} // Demon's Claw, 61-115, 2.30
	items[proto.ItemSlot_ItemSlotOffHand] = &proto.ItemSpec{Id: 18168}  // Force Reactive Disk
	return &proto.EquipmentSpec{Items: items}
}

var DefaultOptions = &proto.Player_ProtectionWarrior{
	ProtectionWarrior: &proto.ProtectionWarrior{
		Options: &proto.ProtectionWarrior_Options{
			ClassOptions: &proto.WarriorOptions{
				QueueDelay:     250,
				UseBattleShout: true,
				DefaultStance:  proto.WarriorStance_WarriorStanceDefensive,
			},
		},
	},
}

var DefaultConsumables = &proto.ConsumesSpec{
	PotId:   22849,
	FlaskId: 22854,
	FoodId:  27667,
}

// The arena entry for this spec. Without ARENA_OUT set it only checks every build's damage against the spell manifest; see sim/arenalib.
func TestArena(t *testing.T) {
	arenalib.Run(t, arenalib.Spec{
		Dir:         "tank_warrior",
		UI:          "warrior/protection",
		Class:       proto.Class_ClassWarrior,
		Race:        proto.Race_RaceOrc,
		SpecOptions: DefaultOptions,
		Role:        arenalib.Melee,
		IsTank:      true,
	})
}
