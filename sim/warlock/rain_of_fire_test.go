package warlock

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// A gearless Warlock channelling nothing but Rain of Fire against three targets: every channel
// ticks four times, each tick hits every target, and a tick deals the client's 221 Fire damage
// before resistances and crits.
func TestRainOfFireRainsOnEveryTarget(t *testing.T) {
	const channel, tick, targets, iterations = 11678, 1282385, 3, 20
	player := core.WithSpec(&proto.Player{
		Race:        proto.Race_RaceOrc,
		Class:       proto.Class_ClassWarlock,
		Equipment:   &proto.EquipmentSpec{},
		Consumables: &proto.ConsumesSpec{},
		Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{{
			Action: &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
				SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: channel}},
			}}},
		}}},
	}, &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{
		Summon: proto.WarlockOptions_NoSummon,
	}}}})
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Duration = 30
	for len(encounter.Targets) < targets {
		encounter.Targets = append(encounter.Targets, core.NewDefaultTarget())
	}
	result := core.RunRaidSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 7},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  encounter,
	})
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}

	var channels, landed, damage float64
	perTarget := map[int32]float64{}
	for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
		switch action.Id.GetSpellId() {
		case channel:
			for _, tm := range action.Targets {
				channels += float64(tm.Casts)
			}
		case tick:
			for _, tm := range action.Targets {
				if tm.UnitIndex >= targets {
					continue
				}
				hits := float64(tm.Hits + tm.Crits)
				perTarget[tm.UnitIndex] += float64(tm.Hits + tm.Crits + tm.Misses)
				landed += hits
				damage += tm.Damage
			}
		}
	}
	if channels == 0 {
		t.Fatal("Rain of Fire was never cast")
	}
	for i := int32(0); i < targets; i++ {
		// The fight can end mid-channel, so allow a short last channel per iteration.
		if got, want := perTarget[i], 4*channels; got < want-4*iterations || got > want {
			t.Errorf("target %d: %.0f ticks for %.0f channels, want about %.0f", i, got, channels, want)
		}
	}
	if avg := damage / landed; avg < 150 || avg > 400 {
		t.Errorf("%.0f damage per landed tick, want near the client's 221", avg)
	}
}
