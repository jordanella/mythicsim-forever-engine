package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/arenalib"
	"github.com/wowsims/forever/sim/common"
	_ "github.com/wowsims/forever/sim/common" // imported to get item effects included.
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func init() {
	RegisterDpsWarrior()
	common.RegisterAllEffects()
}

// The three builds our Forever sim ships for the DPS warrior: its default DPS build, Fury 17/34/0
// and Arms 39/12/0.
var DpsTalents = "30305013-05553005150010501"
var FuryTalents = "30305213-55051105050010501"
var ArmsTalents = "32305213132515201-5502"

func TestFury(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		warriorSuite(DualWieldGear, DpsTalents, []core.TalentsCombo{{Label: "Fury", Talents: FuryTalents}}),
	}))
}

func TestArms(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		warriorSuite(TwoHandGear, ArmsTalents, nil),
	}))
}

// Weapons and nothing else. The generated item database does not carry the Forever gear our sim
// tests with and gives the rest TBC-shaped stats, so the comparison against master runs on the same
// statless axes on both sides instead.
var DualWieldGear = core.GearSetCombo{Label: "DualWield", GearSet: weaponsOnly(15240, 15238)}
var TwoHandGear = core.GearSetCombo{Label: "TwoHand", GearSet: weaponsOnly(15273, 0)}

func weaponsOnly(mainHand int32, offHand int32) *proto.EquipmentSpec {
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: mainHand} // Demon's Claw 61-115 2.3, or Death Striker 109-165 2.8
	items[proto.ItemSlot_ItemSlotOffHand] = &proto.ItemSpec{Id: offHand}   // Warlord's Axe 55-104 2.3
	return &proto.EquipmentSpec{Items: items}
}

func warriorSuite(gear core.GearSetCombo, talents string, otherTalents []core.TalentsCombo) core.CharacterSuiteConfig {
	return core.CharacterSuiteConfig{
		Class:           proto.Class_ClassWarrior,
		Race:            proto.Race_RaceOrc,
		OtherRaces:      []proto.Race{proto.Race_RaceHuman},
		GearSet:         gear,
		Talents:         talents,
		OtherTalentSets: otherTalents,
		Consumables:     DefaultConsumables,
		SpecOptions:     core.SpecOptionsCombo{Label: "DPS", SpecOptions: DefaultOptions},

		Rotation: core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "dps_reck"),
		OtherRotations: []core.RotationCombo{
			core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "dps_no_reck"),
		},

		ItemFilter: core.ItemFilter{
			ArmorType: proto.ArmorType_ArmorTypePlate,
			WeaponTypes: []proto.WeaponType{
				proto.WeaponType_WeaponTypeAxe,
				proto.WeaponType_WeaponTypeSword,
				proto.WeaponType_WeaponTypeMace,
				proto.WeaponType_WeaponTypeDagger,
				proto.WeaponType_WeaponTypeFist,
			},
			HandTypes: []proto.HandType{
				proto.HandType_HandTypeMainHand,
				proto.HandType_HandTypeOffHand,
				proto.HandType_HandTypeOneHand,
				proto.HandType_HandTypeTwoHand,
			},
			// Enchant effects sim/common registers that the generated database has no row for.
			EnchantBlacklist: []int32{963, 1899, 1900, 2523, 2613, 2621, 2673, 2714, 2722, 2723, 2724, 2939, 2940, 3225, 3273},
		},
	}
}

var DefaultOptions = &proto.Player_DpsWarrior{
	DpsWarrior: &proto.DpsWarrior{
		Options: &proto.DpsWarrior_Options{
			ClassOptions: &proto.WarriorOptions{
				StartingRage:   50,
				QueueDelay:     250,
				UseBattleShout: true,
				DefaultStance:  proto.WarriorStance_WarriorStanceBerserker,
			},
		},
	},
}

var DefaultConsumables = &proto.ConsumesSpec{
	PotId:   22838,
	FlaskId: 22854,
	FoodId:  27658,
}

// The arena entry for this spec. Without ARENA_OUT set it only checks every build's damage against the spell manifest; see sim/arenalib.
func TestArena(t *testing.T) {
	arenalib.Run(t, arenalib.Spec{
		Dir:         "warrior",
		UI:          "warrior/dps",
		Class:       proto.Class_ClassWarrior,
		Race:        proto.Race_RaceOrc,
		SpecOptions: DefaultOptions,
		Role:        arenalib.Melee,
	})
}
