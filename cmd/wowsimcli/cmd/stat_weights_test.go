package cmd

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/wowsims/forever/sim"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
	googleproto "google.golang.org/protobuf/proto"
)

func TestStatWeightsCommand(t *testing.T) {
	sim.RegisterAll()
	encounter := googleproto.Clone(core.MakeSingleTargetEncounter(0)).(*proto.Encounter)
	encounter.Duration = 10
	request := &proto.StatWeightsRequest{
		Player: &proto.Player{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome,
			Equipment: &proto.EquipmentSpec{},
			Spec:      &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{{Action: &proto.APLAction{
				Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 25306}}}},
			}}}},
		},
		Encounter: encounter, SimOptions: &proto.SimOptions{Iterations: 100, RandomSeed: 42},
		StatsToWeigh: []proto.Stat{proto.Stat_StatSpellDamage}, EpReferenceStat: proto.Stat_StatSpellDamage,
	}
	raw, err := protojson.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	in, out := filepath.Join(dir, "request.json"), filepath.Join(dir, "result.json")
	if err = os.WriteFile(in, raw, 0644); err != nil {
		t.Fatal(err)
	}
	command := newStatWeightsCommand()
	command.SetArgs([]string{"--strict", "--infile", in, "--outfile", out})
	if err = command.Execute(); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	result := &proto.StatWeightsResult{}
	if err = protojson.Unmarshal(raw, result); err != nil {
		t.Fatal(err)
	}
	if result.Error != nil || result.Dps == nil {
		t.Fatalf("invalid result: %v", result)
	}
	weight := result.Dps.Weights.Stats[proto.Stat_StatSpellDamage]
	ep := result.Dps.EpValues.Stats[proto.Stat_StatSpellDamage]
	if weight <= 0 || math.IsNaN(weight) || math.IsInf(weight, 0) || math.Abs(ep-1) > 1e-9 {
		t.Fatalf("spell damage weight=%v EP=%v, want positive weight and EP=1", weight, ep)
	}
}

func TestStatWeightsRequestValidation(t *testing.T) {
	valid := `{"player":{},"encounter":{},"simOptions":{"iterations":100},"statsToWeigh":["StatSpellDamage"]`
	for _, extra := range []string{`,"misspelt":true}`, `,"epReferenceStat":"StatMisspelt"}`} {
		if _, err := loadStatWeightsRequest([]byte(valid+extra), true); err == nil {
			t.Errorf("strict request accepted %s", extra)
		}
	}
	if _, err := loadStatWeightsRequest([]byte(valid+`,"misspelt":true}`), false); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{}`, `{"player":{},"encounter":{},"simOptions":{"iterations":1},"statsToWeigh":["StatSpellDamage"]}`, `{"player":{},"encounter":{},"simOptions":{"iterations":100}}`} {
		if _, err := loadStatWeightsRequest([]byte(raw), true); err == nil {
			t.Errorf("accepted incomplete request %s", raw)
		}
	}
}
