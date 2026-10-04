package core

import (
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"google.golang.org/protobuf/encoding/protojson"
)

// Every way a rotation builds an item (the priority list, prepull, sequences, strict sequences,
// schedules, groups and group references) must leave it with a readiness function: IsReady calls it
// without a nil check.
func TestEveryRotationItemGetsAReadinessCheck(t *testing.T) {
	aura := `{"activateAura":{"auraId":{"spellId":18499}}}`
	rotationJSON := `{
		"type": "TypeAPL",
		"prepullActions": [{"action": ` + aura + `, "doAtValue": {"const": {"val": "-1s"}}}],
		"priorityList": [
			{"action": ` + aura + `},
			{"action": {"condition": {"const": {"val": "true"}}, "activateAura": {"auraId": {"spellId": 12292}}}},
			{"action": {"sequence": {"name": "seq", "actions": [` + aura + `, {"strictSequence": {"actions": [` + aura + `]}}]}}},
			{"action": {"schedule": {"schedule": "10s", "innerAction": ` + aura + `}}},
			{"action": {"groupReference": {"groupName": "grp"}}}
		],
		"groups": [{"name": "grp", "actions": [{"action": ` + aura + `}, {"action": {"sequence": {"name": "inner", "actions": [` + aura + `]}}}]}]
	}`
	rotation := &proto.APLRotation{}
	if err := protojson.Unmarshal([]byte(rotationJSON), rotation); err != nil {
		t.Fatalf("parsing rotation: %v", err)
	}

	sim := NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{
			Parties: []*proto.Party{{
				Players: []*proto.Player{{
					Name:      "Warrior",
					Class:     proto.Class_ClassWarrior,
					Buffs:     &proto.IndividualBuffs{},
					Spec:      &proto.Player_ProtectionWarrior{},
					Equipment: &proto.EquipmentSpec{},
					Rotation:  rotation,
				}},
				Buffs: &proto.PartyBuffs{},
			}},
		},
		Encounter: &proto.Encounter{
			Targets:  []*proto.Target{{Name: "target", Level: 63, MobType: proto.MobType_MobTypeHumanoid}},
			Duration: 180,
		},
	}, simsignals.CreateSignals())
	sim.Reset()

	rot := sim.Raid.Parties[0].Players[0].GetCharacter().Rotation
	actions := append(rot.allPrepullActions(), rot.allAPLActions()...)
	// 1 prepull; the priority list's 5 items, 3 nested in the sequence, 1 in the schedule, and the
	// group's 3 reached through its reference; the group's 3 again from rot.groups.
	if len(actions) != 16 {
		t.Fatalf("expected 16 actions across prepull, priority list and groups, got %d", len(actions))
	}
	for _, action := range actions {
		if action.ready == nil {
			t.Errorf("%s has no readiness function", action.impl)
		}
	}
}
