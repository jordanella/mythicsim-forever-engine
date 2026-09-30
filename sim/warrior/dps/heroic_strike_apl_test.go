package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// A custom rotation that names Heroic Strike without the queue tag (what a spell picker
// produces) has to queue it onto the next swing. Resolving the untagged hit instead fired
// Heroic Strike off the global cooldown on top of every white swing: a Heroic Strike only
// rotation read 1253 DPS on a gear set whose preset reads about 800.
func TestUntaggedHeroicStrikeQueuesOntoTheSwing(t *testing.T) {
	mainHandSwings := func(t *testing.T, priority []*proto.APLListItem) (white float64, heroic float64) {
		t.Helper()
		player := &proto.Player{
			Name:          "Heroic Strike",
			Race:          proto.Race_RaceOrc,
			Class:         proto.Class_ClassWarrior,
			Equipment:     DualWieldGear.GearSet,
			TalentsString: FuryTalents,
			Spec:          DefaultOptions,
			Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: priority},
		}
		result := core.RunRaidSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{Iterations: 50, RandomSeed: 7},
		})
		if result.Error != nil {
			t.Fatalf("sim failed: %s", result.Error.Message)
		}
		for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
			casts := 0.0
			for _, target := range action.Targets {
				casts += float64(target.Casts)
			}
			id := action.Id
			switch {
			case id.GetOtherId() == proto.OtherAction_OtherActionAttack && id.Tag == 1:
				white += casts / 50
			case id.GetSpellId() == 25286 && id.Tag == 0:
				heroic += casts / 50
			}
		}
		return white, heroic
	}

	baseline, _ := mainHandSwings(t, nil)
	white, heroic := mainHandSwings(t, []*proto.APLListItem{{Action: &proto.APLAction{
		Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: &proto.ActionID{
			RawId: &proto.ActionID_SpellId{SpellId: 25286},
		}}},
	}}})

	if heroic == 0 {
		t.Fatal("the rotation never cast Heroic Strike")
	}
	// Queued, each Heroic Strike takes the place of a white swing, so the main hand swings about as
	// often as it does with no rotation at all. Heroic Strike rage is spent, not generated, so allow
	// a little room for Flurry and Enrage uptime differences.
	if total := white + heroic; total > baseline*1.15 {
		t.Errorf("%.1f white swings and %.1f Heroic Strikes a fight, against %.1f swings with no rotation: "+
			"Heroic Strike is landing on top of the swing instead of replacing it", white, heroic, baseline)
	}
}
