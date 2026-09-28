package enhancement

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// On the Forever beta a hard-cast Lightning Bolt holds the melee swing: a swing that comes due
// during the cast lands as the cast completes, and a cast that completes first resets the swing
// timer. So no swing lands inside a cast, and the first swing after one lands either as it completes
// or a full swing later. An untalented, unhasted Orc with a 3.8 s Arcanite Reaper chain-casts until
// it runs out of mana, then melees.
func TestLightningBoltHoldsTheSwing(t *testing.T) {
	const swing = 3.8
	rotation := core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[{"action":{"castSpell":{"spellId":{"spellId":15208}}}}]}`)
	player := &proto.Player{
		Name: "Shaman", Class: proto.Class_ClassShaman, Race: proto.Race_RaceOrc,
		Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: 12784}}},
		Buffs:     &proto.IndividualBuffs{}, Consumables: &proto.ConsumesSpec{},
		Spec: &proto.Player_EnhancementShaman{EnhancementShaman: &proto.EnhancementShaman{
			Options: &proto.EnhancementShaman_Options{ClassOptions: &proto.ShamanOptions{}},
		}},
		Rotation: rotation,
	}
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 3, DebugFirstIteration: true},
	})
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}

	line := regexp.MustCompile(`^\[([0-9.]+)\] \[Shaman \(#1\)\] (.*)$`)
	type cast struct{ start, end float64 }
	var casts []cast
	var swings []float64
	var castStart float64 = -1
	started := 0
	for _, raw := range strings.Split(result.Logs, "\n") {
		m := line.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		at, _ := strconv.ParseFloat(m[1], 64)
		switch msg := m[2]; {
		case strings.HasPrefix(msg, "Casting {SpellID: 15208}") && !strings.Contains(msg, "Cast Time = 0s"):
			if castStart >= 0 {
				t.Fatalf("the bolt started at %.2f never completed before the next started at %.2f", castStart, at)
			}
			castStart = at
			started++
		case strings.HasPrefix(msg, "Completed cast {SpellID: 15208}") && castStart >= 0:
			casts = append(casts, cast{castStart, at})
			castStart = -1
		case strings.HasPrefix(msg, "Completed cast {OtherID: 3, Tag: 1}"):
			swings = append(swings, at)
		}
	}
	if len(casts) < 5 || len(swings) < 5 {
		t.Fatalf("%d hard casts and %d swings; the fight did not exercise the hold", len(casts), len(swings))
	}

	const eps = 0.011 // the log's two decimals
	held, reset := 0, 0
	previous := -swing
	for _, at := range swings {
		lastEnd := -1.0
		for _, c := range casts {
			if at > c.start+eps && at < c.end-eps {
				t.Fatalf("a swing landed at %.2f inside the cast from %.2f to %.2f", at, c.start, c.end)
			}
			if c.end <= at+eps {
				lastEnd = c.end
			}
		}
		switch {
		case lastEnd >= 0 && at < lastEnd+eps:
			held++ // waited out the cast and landed as it completed
		case lastEnd > previous+eps:
			// a cast completed since the last swing, before this one was due: it reset the timer
			if at < lastEnd+swing-eps {
				t.Fatalf("the swing at %.2f came less than a swing after the cast completing at %.2f", at, lastEnd)
			}
			reset++
		}
		previous = at
	}
	if held == 0 || reset == 0 {
		t.Fatalf("held %d and reset %d swings; want both", held, reset)
	}
	if started != len(casts) {
		t.Fatalf("%d bolts started, %d completed", started, len(casts))
	}
}
