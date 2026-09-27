package mage

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func TestInnervateAccountingOverlapsEvocation(t *testing.T) {
	for _, evocationFirst := range []bool{false, true} {
		name := "innervate-first"
		if evocationFirst {
			name = "evocation-first"
		}
		t.Run(name, func(t *testing.T) {
			player := &proto.Player{Name: name, Class: proto.Class_ClassMage, Race: proto.Race_RaceHuman, Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{Innervates: 1}, Spec: &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}}, Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL}}
			sim := core.NewSim(&proto.RaidSimRequest{SimOptions: &proto.SimOptions{RandomSeed: 42}, Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}), Encounter: core.MakeSingleTargetEncounter(0)}, simsignals.CreateSignals())
			sim.Reset()
			char := sim.Raid.Parties[0].Players[0].GetCharacter()
			char.SpendMana(sim, 1000, char.NewManaMetrics(core.ActionID{SpellID: 1}))
			char.PseudoStats.FiveSecondRuleRefreshTime = time.Second * 5
			innervate, evocation := char.GetAura("Innervates (External)"), char.GetAura("Evocation Regen")
			// Capture the actual Evocation-only rate, including its client-derived bonus.
			evocation.Activate(sim)
			baseline := char.ManaRegenPerSecondWhileCasting() * 2
			if !evocationFirst {
				evocation.Deactivate(sim)
			}
			innervate.Activate(sim)
			if !evocationFirst {
				evocation.Activate(sim)
			}
			offered := char.ManaRegenPerSecondWhileCasting() * 2
			before := char.CurrentMana()
			char.ManaTick(sim)
			var source *proto.ResourceMetrics
			for _, r := range char.GetMetricsProto().Resources {
				if r.Id.GetSpellId() == 29166 {
					source = r
				}
			}
			if source == nil {
				t.Fatal("missing Innervate resource source")
			}
			bonus := offered - baseline
			actual := math.Max(0, char.CurrentMana()-before-baseline)
			if math.Abs(source.Gain-bonus) > 1e-7 || math.Abs(source.ActualGain-actual) > 1e-7 {
				t.Fatalf("wrong overlapping attribution: %v, want offered=%v actual=%v", source, bonus, actual)
			}
			// Expiring the channel must also remove its contribution from the baseline.
			evocation.Deactivate(sim)
			events := source.Events
			char.ManaTick(sim)
			for _, r := range char.GetMetricsProto().Resources {
				if r.Id.GetSpellId() == 29166 && r.Events != events {
					t.Fatal("zero casting regeneration created a bonus event")
				}
			}
		})
	}
}
