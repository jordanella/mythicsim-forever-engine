package mage

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/arenalib"
	"github.com/wowsims/forever/sim/common"
	_ "github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func init() {
	RegisterMage()
	common.RegisterAllEffects()
}

func TestArcane(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{mageSuite("arcane", ArcaneTalents)}))
}

func TestFire(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{mageSuite("fire", FireTalents)}))
}

func TestFrost(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{mageSuite("frost", FrostTalents)}))
}

// The community builds our Forever sim ranks: Arcane 35/0/16, Fire 0/35/16 and Frost 14/0/37.
var ArcaneTalents = "055005023100311531--005500033"
var FireTalents = "-03552020130133151-005500033"
var FrostTalents = "050005013--0555003301001301251"

func mageSuite(apl string, talents string) core.CharacterSuiteConfig {
	return core.CharacterSuiteConfig{
		Class:      proto.Class_ClassMage,
		Race:       proto.Race_RaceGnome,
		OtherRaces: []proto.Race{proto.Race_RaceTroll},
		SpecOptions: core.SpecOptionsCombo{Label: "MageArmor", SpecOptions: &proto.Player_Mage{
			Mage: &proto.Mage{
				Options: &proto.Mage_Options{
					ClassOptions: &proto.MageOptions{
						DefaultMageArmor: proto.MageArmor_MageArmorMageArmor,
					},
				},
			},
		}},
		// Naked: the generated item database does not carry most of the pre-raid set our Forever sim
		// tests with yet, and gives the rest TBC-shaped stats.
		GearSet:  core.GearSetCombo{Label: "Naked", GearSet: &proto.EquipmentSpec{}},
		Talents:  talents,
		Rotation: core.GetAplRotation("../../ui/specs/mage/dps/apls", apl),
		ItemFilter: core.ItemFilter{
			WeaponTypes: []proto.WeaponType{
				proto.WeaponType_WeaponTypeDagger,
				proto.WeaponType_WeaponTypeSword,
				proto.WeaponType_WeaponTypeOffHand,
				proto.WeaponType_WeaponTypeStaff,
			},
			ArmorType: proto.ArmorType_ArmorTypeCloth,
			RangedWeaponTypes: []proto.RangedWeaponType{
				proto.RangedWeaponType_RangedWeaponTypeWand,
			},
			EnchantBlacklist: []int32{2673, 3225, 3273},
		},
	}
}

// Hot Streak (400625) has one charge: the Pyroblast its stacks speed up spends all of them.
func TestHotStreakSpentByPyroblast(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome, TalentsString: FireTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec:     &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	pyroblastRank := spellData.Pyroblast.Highest()
	pyroblast := mage.GetSpell(core.ActionID{SpellID: pyroblastRank.ID})

	mage.HotStreakAura.Activate(sim)
	mage.HotStreakAura.SetStacks(sim, 3)
	if !pyroblast.Cast(sim, mage.CurrentTarget) {
		t.Fatal("Pyroblast did not cast")
	}
	if want := pyroblastRank.CastTime() / 4; mage.Hardcast.Expires != want {
		t.Errorf("Pyroblast cast ends at %v, want %v with 3 stacks", mage.Hardcast.Expires, want)
	}

	for sim.CurrentTime < 5*time.Second && mage.HotStreakAura.IsActive() {
		sim.Step()
	}
	if mage.HotStreakAura.IsActive() {
		t.Errorf("Hot Streak still up at %v after Pyroblast finished", sim.CurrentTime)
	}
}

// Frostfire Bolt (1237313) casts, lands its bolt and dot, and takes Improved Fireball's cast time cut
// exactly as Fireball does: the client's mask on 11069 names both. Seed 2 is one where the bolt is
// not resisted (Frostfire Bolt rolls binary resistance here, which shifts which seeds hit).
func TestFrostfireBolt(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 2},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome, TalentsString: FireTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec:     &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	if mage.Talents.ImprovedFireball == 0 {
		t.Fatal("FireTalents no longer take Improved Fireball; pick a build that does")
	}
	fireball := mage.GetSpell(core.ActionID{SpellID: spellData.Fireball.Highest().ID})
	ffbRank := spellData.FrostfireBolt.Highest()
	ffb := mage.GetSpell(core.ActionID{SpellID: ffbRank.ID})
	if ffb == nil {
		t.Fatal("Frostfire Bolt is not registered")
	}
	if want := ffbRank.CastTime() - (spellData.Fireball.Highest().CastTime() - fireball.DefaultCast.CastTime); ffb.DefaultCast.CastTime != want {
		t.Errorf("Frostfire Bolt casts in %v, want %v (Improved Fireball's cut)", ffb.DefaultCast.CastTime, want)
	}

	if !ffb.Cast(sim, mage.CurrentTarget) {
		t.Fatal("Frostfire Bolt did not cast")
	}
	for sim.CurrentTime < 8*time.Second {
		sim.Step()
	}
	if ffb.SpellMetrics[0].TotalDamage <= 0 || !ffb.Dot(mage.CurrentTarget).IsActive() {
		t.Errorf("Frostfire Bolt dealt %v and its dot is up: %v", ffb.SpellMetrics[0].TotalDamage, ffb.Dot(mage.CurrentTarget).IsActive())
	}
}

// The arena entry for this spec. Without ARENA_OUT set it only checks every build's damage against the spell manifest; see sim/arenalib.
func TestArena(t *testing.T) {
	arenalib.Run(t, arenalib.Spec{
		Dir:   "mage",
		UI:    "mage/dps",
		Class: proto.Class_ClassMage,
		Race:  proto.Race_RaceGnome,
		SpecOptions: &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{
			ClassOptions: &proto.MageOptions{DefaultMageArmor: proto.MageArmor_MageArmorMageArmor},
		}}},
		Role:               arenalib.Caster,
		DistanceFromTarget: 30,
	})
}
