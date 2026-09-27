package core

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

func attributionUnit() *Unit {
	u := &Unit{stats: stats.Stats{stats.Mana: 2000, stats.Spirit: 100, stats.MP5: 50}, PseudoStats: stats.NewPseudoStats(), Metrics: NewUnitMetrics()}
	u.manaBar.unit = u
	u.spiritRegenPerSpirit = .1
	u.manaRegenMultiplier = 1
	u.manaCastingMetrics = u.NewManaMetrics(ActionID{OtherID: proto.OtherAction_OtherActionManaRegen, Tag: 1})
	u.manaNotCastingMetrics = u.NewManaMetrics(ActionID{OtherID: proto.OtherAction_OtherActionManaRegen, Tag: 2})
	return u
}

func startAttribution(u *Unit) *ResourceMetrics {
	m := u.NewManaMetrics(ActionID{SpellID: 29166})
	u.StartSpiritRegenAttribution(m)
	u.PseudoStats.SpiritRegenMultiplier *= 5
	u.PseudoStats.ForceFullSpiritRegen = true
	u.UpdateManaRegenRates()
	return m
}

func TestSpiritRegenAttributionConservesMana(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		casting                         bool
		fraction, room, baseline, bonus float64
	}{
		{"casting", true, 0, 1000, 20, 100},
		{"casting-with-talent", true, .3, 1000, 26, 94},
		{"not-casting", false, .3, 1000, 40, 80},
		{"partly-capped", true, .3, 50, 26, 94},
		{"baseline-fills-cap", true, .3, 10, 26, 94},
		{"fully-capped", false, .3, 0, 40, 80},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := attributionUnit()
			u.currentMana = u.MaxMana() - tc.room
			u.PseudoStats.SpiritRegenRateCasting = tc.fraction
			if tc.casting {
				u.PseudoStats.FiveSecondRuleRefreshTime = time.Second
			}
			source := startAttribution(u)
			u.ManaTick(&Simulation{})
			actual := math.Min(tc.room, tc.baseline+tc.bonus)
			normalActual := math.Min(tc.room, tc.baseline)
			normal := u.manaNotCastingMetrics
			if tc.casting {
				normal = u.manaCastingMetrics
			}
			closeMana(t, "bonus offered", source.Gain, tc.bonus)
			closeMana(t, "bonus actual", source.ActualGain, actual-normalActual)
			closeMana(t, "normal offered", normal.Gain, tc.baseline)
			closeMana(t, "normal actual", normal.ActualGain, normalActual)
			closeMana(t, "total gained", u.Metrics.ManaGained, actual)
			closeMana(t, "current mana", u.CurrentMana(), u.MaxMana()-tc.room+actual)
			if source.Events != 1 || !source.isManaRegen {
				t.Fatal("bonus must be one passive regen event")
			}
		})
	}
}

func TestSpiritRegenAttributionTracksOtherEffects(t *testing.T) {
	for _, tc := range []struct {
		name            string
		before, after   func(*Unit)
		baseline, bonus float64
	}{
		{"evocation-before", func(u *Unit) { u.AddSpiritRegenMultiplier(15); u.SetForceFullSpiritRegen(true) }, nil, 340, 1280},
		{"evocation-after", nil, func(u *Unit) { u.AddSpiritRegenMultiplier(15); u.SetForceFullSpiritRegen(true) }, 340, 80},
		{"form-after", nil, func(u *Unit) { u.MultiplySpiritRegenMultiplier(.5) }, 30, 40},
		{"leave-form", func(u *Unit) { u.MultiplySpiritRegenMultiplier(.5) }, func(u *Unit) { u.DivideSpiritRegenMultiplier(.5) }, 40, 80},
		{"changed-spirit-and-mp5", nil, func(u *Unit) { u.stats[stats.Spirit] = 200; u.stats[stats.MP5] = 100 }, 80, 160},
		{"regen-speed", nil, func(u *Unit) { u.MultiplyManaRegenSpeed(&Simulation{}, 2) }, 80, 160},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := attributionUnit()
			if tc.before != nil {
				tc.before(u)
			}
			m := startAttribution(u)
			if tc.after != nil {
				tc.after(u)
			}
			u.UpdateManaRegenRates()
			u.ManaTick(&Simulation{})
			closeMana(t, "baseline", u.manaNotCastingMetrics.Gain, tc.baseline)
			closeMana(t, "bonus", m.Gain, tc.bonus)
			closeMana(t, "conservation", u.Metrics.ManaGained, tc.baseline+tc.bonus)
		})
	}
}

func TestSpiritRegenAttributionStopsAndResets(t *testing.T) {
	u := attributionUnit()
	m := startAttribution(u)
	u.ManaTick(&Simulation{})
	u.StopSpiritRegenAttribution()
	u.PseudoStats.SpiritRegenMultiplier /= 5
	u.PseudoStats.ForceFullSpiritRegen = false
	u.UpdateManaRegenRates()
	u.ManaTick(&Simulation{})
	if m.Events != 1 {
		t.Fatal("expired source received another tick")
	}
	closeMana(t, "ordinary ticks", u.manaNotCastingMetrics.Gain, 80)
	startAttribution(u)
	u.manaBar.reset()
	if u.spiritRegenAttribution != nil {
		t.Fatal("source leaked into next iteration")
	}
}

func closeMana(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-8 {
		t.Fatalf("%s = %.10f, want %.10f", name, got, want)
	}
}
