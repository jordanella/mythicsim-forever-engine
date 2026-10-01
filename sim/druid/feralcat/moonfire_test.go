package feralcat

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/druid"
)

// A Feral Cat druid knows Moonfire (a baseline druid spell, castable only in caster form), so a rotation
// can leave the form, refresh Moonfire and shift back. Before this the Cat agent never registered the
// spell: every line naming it was dropped as "does not know spell", and a condition that named its dot
// lost that operand, which turned a Moonfire weave into a bare powershift (a player's 427 DPS run).
func TestCatCanCastMoonfireFromCatForm(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Cat", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf, TalentsString: DefaultTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{}, Spec: DefaultSpecOptions,
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	cat := sim.Raid.Parties[0].Players[0].(druid.DruidAgent).GetDruid()
	if cat.Moonfire == nil {
		t.Fatal("the Cat druid has no Moonfire")
	}
	if !cat.InForm(druid.Cat) {
		t.Fatal("the Cat druid does not start in Cat Form")
	}

	for !cat.GCD.IsReady(sim) && sim.CurrentTime < 5*time.Second {
		sim.Step()
	}
	mana := cat.CurrentMana()
	if !cat.Moonfire.Cast(sim, cat.CurrentTarget) {
		t.Fatal("Moonfire did not cast from Cat Form")
	}
	if cat.InForm(druid.Cat) || cat.CatFormAura.IsActive() {
		t.Error("casting Moonfire from Cat Form left the druid in Cat Form")
	}
	if cat.CurrentMana() >= mana {
		t.Errorf("Moonfire cost no mana (%v before, %v after)", mana, cat.CurrentMana())
	}
	if dot := cat.Moonfire.Dot(cat.CurrentTarget); dot == nil || !dot.IsActive() {
		t.Error("Moonfire did not leave its dot on the target")
	}
}

// The same weave written as the player's APL: leave Cat Form while Moonfire is down, cast it, shift
// back once it is up. Every line has to survive to the engine, and the dot has to do the damage.
func TestMoonfireWeaveRunsFromAnAPL(t *testing.T) {
	const apl = `{
		"type": "TypeAPL",
		"priorityList": [
			{"action": {"cancelAura": {"auraId": {"spellId": 768}}, "condition": {"and": {"vals": [
				{"auraIsActive": {"auraId": {"spellId": 768}}},
				{"not": {"val": {"dotIsActive": {"spellId": {"spellId": 9835, "rank": 10}}}}}
			]}}}},
			{"action": {"castSpell": {"spellId": {"spellId": 9835, "rank": 10}}, "condition": {"and": {"vals": [
				{"not": {"val": {"auraIsActive": {"auraId": {"spellId": 768}}}}},
				{"not": {"val": {"dotIsActive": {"spellId": {"spellId": 9835, "rank": 10}}}}}
			]}}}},
			{"action": {"castSpell": {"spellId": {"spellId": 768}}, "condition": {"and": {"vals": [
				{"dotIsActive": {"spellId": {"spellId": 9835, "rank": 10}}},
				{"not": {"val": {"auraIsActive": {"auraId": {"spellId": 768}}}}}
			]}}}},
			{"action": {"castSpell": {"spellId": {"spellId": 9830}}}}
		]
	}`
	player := &proto.Player{
		Name: "Cat", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf, TalentsString: DefaultTalents,
		Equipment: &proto.EquipmentSpec{}, Spec: DefaultSpecOptions,
		Rotation: core.APLRotationFromJsonString(apl),
	}
	// A short fight: a naked cat has 2614 mana, enough for two weaves (Moonfire 375, Cat Form 684).
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Duration = 24
	res := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  encounter,
		SimOptions: &proto.SimOptions{Iterations: 20, RandomSeed: 1},
	})
	if res.Error != nil {
		t.Fatal(res.Error.Message)
	}

	var casts, shifts, tickDamage float64
	for _, action := range res.RaidMetrics.Parties[0].Players[0].Actions {
		switch {
		case action.Id.GetSpellId() == 9835 && action.Id.GetTag() == 0:
			for _, target := range action.Targets {
				casts += float64(target.Casts)
			}
		case action.Id.GetSpellId() == 9835 && action.Id.GetTag() == 1:
			for _, target := range action.Targets {
				tickDamage += target.Damage
			}
		case action.Id.GetSpellId() == 768:
			for _, target := range action.Targets {
				shifts += float64(target.Casts)
			}
		}
	}
	// A 12 sec dot over 24 seconds: casts at about 0, 12 and 24 sec, not none and not one every global.
	if perFight := casts / 20; perFight < 2 || perFight > 4 {
		t.Errorf("%.1f Moonfire casts a fight, want 2 to 4", perFight)
	}
	if tickDamage == 0 {
		t.Error("the Moonfire dot dealt no damage")
	}
	// The druid shifts back into the form after Moonfire (the last cast lands as the fight ends).
	if shifts < 20 {
		t.Errorf("%v shifts into Cat Form over 20 fights, want at least one a fight", shifts)
	}
}
