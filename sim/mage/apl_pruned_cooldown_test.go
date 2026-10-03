package mage

import (
	"fmt"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func TestAPLPrunedSequenceCooldownsDoNotAutocast(t *testing.T) {
	const arcanePower, frostbolt = 12042, 25304
	cast := `"castSpell":{"spellId":{"spellId":12042}}`
	sequence := `"sequence":{"name":"cooldowns","actions":[{` + cast + `}]}`
	strictSequence := `"strictSequence":{"actions":[{` + cast + `}]}`
	actions := []struct {
		name string
		json string
	}{
		{"direct spell", cast},
		{"sequence", sequence},
		{"strict sequence", strictSequence},
		{"nested sequences", `"sequence":{"name":"outer","actions":[{` + strictSequence + `}]}`},
	}
	conditions := []struct {
		name      string
		json      string
		wantCasts bool
	}{
		{"unsupported condition", `{"currentSolarEnergy":{}}`, false},
		{"constant false", `{"const":{"val":"false"}}`, false},
		{"missing spell", `{"spellIsReady":{"spellId":{"spellId":1}}}`, false},
		{"constant true", `{"const":{"val":"true"}}`, true},
	}
	for _, action := range actions {
		for _, condition := range conditions {
			t.Run(action.name+"/"+condition.name, func(t *testing.T) {
				rotation := core.APLRotationFromJsonString(fmt.Sprintf(`{
					"priorityList":[
						{"action":{"condition":%s,%s}},
						{"action":{"autocastOtherCooldowns":{}}},
						{"action":{"castSpell":{"spellId":{"spellId":25304}}}}
					]
				}`, condition.json, action.json))
				player := &proto.Player{
					Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome,
					TalentsString: ArcaneTalents, Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
					Spec:     &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
					Rotation: rotation,
				}
				result := core.RunRaidSim(&proto.RaidSimRequest{
					Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
					Encounter:  core.MakeSingleTargetEncounter(0),
					SimOptions: &proto.SimOptions{Iterations: 2, RandomSeed: 1},
				})
				if result.Error != nil {
					t.Fatalf("sim failed: %s", result.Error.Message)
				}
				casts := make(map[int32]int32)
				for _, spell := range result.RaidMetrics.Parties[0].Players[0].Actions {
					for _, target := range spell.Targets {
						casts[spell.Id.GetSpellId()] += target.Casts
					}
				}
				if casts[frostbolt] == 0 {
					t.Fatal("fallback Frostbolt never cast")
				}
				if (casts[arcanePower] > 0) != condition.wantCasts {
					t.Errorf("Arcane Power cast %d times; want casts: %v", casts[arcanePower], condition.wantCasts)
				}
			})
		}
	}
}
