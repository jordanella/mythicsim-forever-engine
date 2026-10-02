package feralbear

import (
	"testing"

	"github.com/wowsims/forever/sim/arenalib"
	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func init() {
	RegisterFeralBearDruid()
	common.RegisterAllEffects()
}

func TestFeralBear(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassDruid,
			Race:       proto.Race_RaceNightElf,
			OtherRaces: []proto.Race{proto.Race_RaceTauren},

			// Naked: the generated item database does not carry the Forever gear our sim tests
			// with yet, and gives the rest TBC-shaped stats.
			GearSet: core.GearSetCombo{Label: "Naked", GearSet: &proto.EquipmentSpec{}},

			Talents:     DefaultTalents,
			SpecOptions: core.SpecOptionsCombo{Label: "Standard", SpecOptions: DefaultSpecOptions},

			Rotation: core.GetAplRotation("../../../ui/specs/druid/feralbear/apls", "default"),

			Consumables: DefaultConsumables,

			Profession1: proto.Profession_Engineering,
			Profession2: proto.Profession_Enchanting,

			IndividualBuffs: core.FullTankIndividualBuffs,

			IsTank:          true,
			InFrontOfTarget: true,

			ItemFilter: core.ItemFilter{
				ArmorType: proto.ArmorType_ArmorTypeLeather,
				WeaponTypes: []proto.WeaponType{
					proto.WeaponType_WeaponTypeDagger,
					proto.WeaponType_WeaponTypeFist,
					proto.WeaponType_WeaponTypeMace,
					proto.WeaponType_WeaponTypeStaff,
				},
				RangedWeaponTypes: []proto.RangedWeaponType{
					proto.RangedWeaponType_RangedWeaponTypeIdol,
				},
			},

			EPReferenceStat: proto.Stat_StatAgility,
			StatsToWeigh: []proto.Stat{
				proto.Stat_StatHealth,
				proto.Stat_StatStamina,
				proto.Stat_StatAgility,
				proto.Stat_StatStrength,
				proto.Stat_StatAttackPower,
				proto.Stat_StatArmor,
				proto.Stat_StatBonusArmor,
				proto.Stat_StatDodgeRating,
				proto.Stat_StatDefenseRating,
				proto.Stat_StatMeleeHitRating,
				proto.Stat_StatExpertiseRating,
			},
		},
	}))
}

// Our Forever sim's Bear Tank build, 0/31/20.
const DefaultTalents = "-50032302120132010501-0550325"

var DefaultSpecOptions = &proto.Player_FeralBearDruid{
	FeralBearDruid: &proto.FeralBearDruid{
		Options: &proto.FeralBearDruid_Options{
			StartingRage: 25,
		},
	},
}

var DefaultConsumables = &proto.ConsumesSpec{
	GuardianElixirId: 9088, // Gift of Arthas
	GoblinSapper:     true,
}

// The arena entry for this spec. Without ARENA_OUT set it only checks every build's damage against the spell manifest; see sim/arenalib.
func TestArena(t *testing.T) {
	arenalib.Run(t, arenalib.Spec{
		Dir:         "feral_tank_druid",
		UI:          "druid/feralbear",
		Class:       proto.Class_ClassDruid,
		Race:        proto.Race_RaceTauren,
		SpecOptions: DefaultSpecOptions,
		Role:        arenalib.Melee,
		IsTank:      true,
	})
}
