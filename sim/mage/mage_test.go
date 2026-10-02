package mage

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/arenalib"
	"github.com/wowsims/forever/sim/common"
	_ "github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
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

// Frostbolt rolls the spread its client row states, as the beta's do (Gromnie, foreverlogs 2679:
// rank 4 non-crits land anywhere from 65 to 74), where it used to deal the row's average every cast.
func TestFrostboltRollsItsRow(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome, TalentsString: FrostTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec:     &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	frostbolt := mage.GetSpell(core.ActionID{SpellID: spellData.Frostbolt.Highest().ID})
	m := &frostbolt.SpellMetrics[0]
	seen := map[float64]bool{}
	for i := 0; i < 30; i++ {
		damage, hits, crits, resisted := m.TotalDamage, m.Hits, m.Crits, m.ResistedHits
		frostbolt.ApplyEffects(sim, mage.CurrentTarget, frostbolt)
		for end := sim.CurrentTime + 5*time.Second; sim.CurrentTime < end; {
			sim.Step()
		}
		if m.Hits > hits && m.Crits == crits && m.ResistedHits == resisted {
			seen[math.Round(m.TotalDamage-damage)] = true
		}
	}
	if len(seen) < 3 {
		t.Errorf("Frostbolt non-crits dealt only %v; it should roll its row's spread", seen)
	}
}

// Master of Elements (29074) refunds once per cast: its 9 ms ProcCategoryRecovery stops a Cone of
// Cold that crits three targets from refunding three times.
func TestMasterOfElementsRefundsOncePerCast(t *testing.T) {
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Targets = append(encounter.Targets, core.NewDefaultTarget(), core.NewDefaultTarget())
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome, TalentsString: FireTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec:     &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: encounter,
	}, simsignals.CreateSignals())
	sim.Reset()

	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	if mage.Talents.MasterOfElements == 0 {
		t.Fatal("FireTalents no longer take Master of Elements; pick a build that does")
	}
	coneOfCold := mage.GetSpell(core.ActionID{SpellID: spellData.ConeOfCold.Highest().ID})
	mage.AddStatDynamic(sim, stats.SpellCritPercent, 100)
	mage.SpendMana(sim, mage.CurrentMana()/2, mage.NewManaMetrics(coneOfCold.ActionID))

	before, crits := mage.CurrentMana(), coneOfCold.SpellMetrics
	if !coneOfCold.Cast(sim, mage.CurrentTarget) {
		t.Fatal("Cone of Cold did not cast")
	}
	totalCrits := int32(0)
	for i := range crits {
		totalCrits += crits[i].Crits
	}
	if totalCrits < 2 {
		t.Fatalf("Cone of Cold crit %d targets; the test needs at least 2", totalCrits)
	}
	want := float64(coneOfCold.Cost.BaseCost) * spellData.MasterOfElements.FractionAt(mage.Talents.MasterOfElements)
	if got := mage.CurrentMana() - before + coneOfCold.CurCast.Cost; math.Abs(got-want) > 0.01 {
		t.Errorf("Master of Elements refunded %.1f mana off %d crits, want one refund of %.1f", got, totalCrits, want)
	}
}

// Arcane Instability (15058) and Arcane Power (12042) name Frostfire Bolt in their SPELLMOD_DAMAGE
// mask but not their SPELLMOD_DOT one (client 1.60.1.70170): the bolt's hit takes the bonus, its DoT
// does not.
func TestArcaneBonusesSkipFrostfireBoltDot(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome, TalentsString: ArcaneTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec:     &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	if mage.Talents.ArcaneInstability == 0 || !mage.Talents.ArcanePower {
		t.Fatal("ArcaneTalents no longer take Arcane Instability and Arcane Power; pick a build that does")
	}
	ffb := mage.GetSpell(core.ActionID{SpellID: spellData.FrostfireBolt.Highest().ID})
	table := mage.AttackTables[mage.CurrentTarget.UnitIndex]
	gap := func() float64 {
		return ffb.AttackerDamageMultiplier(table, false) - ffb.AttackerDamageMultiplier(table, true)
	}

	instability := spellData.ArcaneInstability.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).FractionAt(mage.Talents.ArcaneInstability)
	// Piercing Ice's hit/DoT split, if the build takes it (TestPiercingIceDotBonusStaysAtBasePoints).
	if points := mage.Talents.PiercingIce; points > 0 {
		instability += spellData.PiercingIce.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).FractionAt(points) -
			spellData.PiercingIce.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DOT)).FractionAt(points)
	}
	if got := gap(); math.Abs(got-instability) > 1e-9 {
		t.Errorf("Frostfire Bolt hit - DoT multiplier = %.4f, want %.4f on the hit alone", got, instability)
	}
	mage.ArcanePowerAura.Activate(sim)
	power := spellData.ArcanePower.Highest().Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).Average(core.CharacterLevel) / 100
	if got := gap(); math.Abs(got-instability-power) > 1e-9 {
		t.Errorf("with Arcane Power, Frostfire Bolt hit - DoT multiplier = %.4f, want %.4f", got, instability+power)
	}
}

// Piercing Ice (11151): effect 0 (SPELLMOD_DAMAGE) scales 2/4/6%, but effect 1 (SPELLMOD_DOT, Blizzard
// and Frostfire Bolt) has no rank curve in client 1.60.1.70170 and stays at 2%. Frostfire Bolt's hit
// takes the full bonus, its DoT 2%.
func TestPiercingIceDotBonusStaysAtBasePoints(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome, TalentsString: FrostTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec:     &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	if mage.Talents.PiercingIce < 2 {
		t.Fatal("FrostTalents no longer take 2+ points of Piercing Ice; pick a build that does")
	}
	hit := spellData.PiercingIce.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DAMAGE)).FractionAt(mage.Talents.PiercingIce)
	dot := spellData.PiercingIce.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_DOT)).FractionAt(mage.Talents.PiercingIce)
	if dot != 0.02 || hit <= dot {
		t.Fatalf("Piercing Ice rank %d: hit %.3f, DoT %.3f; client 70170 has 2/4/6%% and a flat 2%%", mage.Talents.PiercingIce, hit, dot)
	}

	ffb := mage.GetSpell(core.ActionID{SpellID: spellData.FrostfireBolt.Highest().ID})
	table := mage.AttackTables[mage.CurrentTarget.UnitIndex]
	if got := ffb.AttackerDamageMultiplier(table, false) - ffb.AttackerDamageMultiplier(table, true); math.Abs(got-(hit-dot)) > 1e-9 {
		t.Errorf("Frostfire Bolt hit - DoT multiplier = %.4f, want %.4f (Piercing Ice %.2f on the hit, %.2f on the DoT)", got, hit-dot, hit, dot)
	}
}
