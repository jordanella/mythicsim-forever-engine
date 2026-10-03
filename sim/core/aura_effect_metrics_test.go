package core

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
)

func TestAuraEffectMetricsSuppressionAndResume(t *testing.T) {
	target := newExclusiveTestTarget()
	makeAura := func(label string, id int32, priority float64) *Aura {
		aura := target.RegisterAura(Aura{Label: label, ActionID: ActionID{SpellID: id}, Duration: NeverExpires})
		aura.NewExclusiveEffect("MinorArmorReductionArmorAdd", false, ExclusiveEffect{Priority: priority})
		return aura
	}
	first := makeAura("First", 1, 505)
	second := makeAura("Second", 2, 505)
	sim := &Simulation{CurrentTime: -10 * time.Second}
	first.Activate(sim)
	sim.CurrentTime = 30 * time.Second
	second.Activate(sim)
	if !first.IsActive() || !second.IsActive() {
		t.Fatal("suppression must preserve both auras")
	}
	sim.CurrentTime = 70 * time.Second
	second.Deactivate(sim)
	sim.CurrentTime = 100 * time.Second
	target.doneIteration(sim)
	metrics := target.GetMetricsProto()
	if metrics[0].UptimeSecondsAvg != 100 || metrics[0].Effects[0].UptimeSecondsAvg != 60 || metrics[1].Effects[0].UptimeSecondsAvg != 40 {
		t.Fatalf("incorrect aura/effect uptime: %v", metrics)
	}
	// A fully suppressed iteration must contribute zero, not disappear from the average.
	sim.CurrentTime = 0
	target.auraTracker.reset(sim)
	first.Activate(sim)
	second.Activate(sim)
	sim.CurrentTime = 100 * time.Second
	target.doneIteration(sim)
	metrics = target.GetMetricsProto()
	if metrics[0].Effects[0].UptimeSecondsAvg != 30 || metrics[1].Effects[0].UptimeSecondsAvg != 70 {
		t.Fatalf("incorrect average after reset: %v", metrics)
	}
}

func TestAuraEffectMetricsPrepullAndLazyExpiry(t *testing.T) {
	target := newExclusiveTestTarget()
	aura := target.RegisterAura(Aura{Label: "Short", ActionID: ActionID{SpellID: 1}, Duration: 10 * time.Second})
	aura.NewExclusiveEffect("Armor", false, ExclusiveEffect{Priority: 1})
	sim := &Simulation{CurrentTime: -5 * time.Second}
	aura.Activate(sim)
	sim.CurrentTime = 20 * time.Second
	target.doneIteration(sim)
	if got := target.GetMetricsProto()[0].Effects[0].UptimeSecondsAvg; got != 5 {
		t.Fatalf("uptime %v, want 5 seconds after pull before expiry", got)
	}
}

func TestAuraEffectMetricsConcurrentMerge(t *testing.T) {
	combiner := &raidSimResultCombiner{}
	base := &proto.AuraMetrics{AggregatorData: &proto.AggregatorData{}}
	for i, seconds := range []float64{0, 100} {
		weight := []float64{0.25, 0.75}[i]
		combiner.combineAuraMetrics(base, &proto.AuraMetrics{
			AggregatorData: &proto.AggregatorData{N: 1},
			Effects:        []*proto.AuraEffectMetrics{{Category: "Armor", UptimeSecondsAvg: seconds}},
		}, weight, i == 1)
	}
	if got := base.Effects[0].UptimeSecondsAvg; got != 75 {
		t.Fatalf("weighted effect uptime %v, want 75", got)
	}
}
