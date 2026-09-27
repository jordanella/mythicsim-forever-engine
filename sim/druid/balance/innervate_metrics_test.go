package balance

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func TestInnervateAttributesActualTicks(t *testing.T) {
	for _, external := range []bool{false, true} {
		label := "Innervates (Player)"
		if external {
			label = "Innervates (External)"
		}
		t.Run(label, func(t *testing.T) {
			player := &proto.Player{Name: "Innervate accounting", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf, Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{Innervates: 1}, Spec: &proto.Player_BalanceDruid{BalanceDruid: &proto.BalanceDruid{Options: &proto.BalanceDruid_Options{ClassOptions: &proto.DruidOptions{}}}}, Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL}}
			sim := core.NewSim(&proto.RaidSimRequest{SimOptions: &proto.SimOptions{RandomSeed: 42}, Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}), Encounter: core.MakeSingleTargetEncounter(0)}, simsignals.CreateSignals())
			char := sim.Raid.Parties[0].Players[0].GetCharacter()
			sim.Reset()
			char.SpendMana(sim, 1000, char.NewManaMetrics(core.ActionID{SpellID: 1}))
			char.PseudoStats.FiveSecondRuleRefreshTime = time.Second * 5
			baseline := char.ManaRegenPerSecondWhileCasting() * 2
			aura := char.GetAura(label)
			if aura == nil {
				t.Fatal("missing aura")
			}
			if external {
				aura.Activate(sim)
			} else if !char.GetSpell(core.ActionID{SpellID: 29166}).Cast(sim, char.CurrentTarget) {
				t.Fatal("self cast failed")
			}
			offered := char.ManaRegenPerSecondWhileCasting() * 2
			before := char.CurrentMana()
			char.ManaTick(sim)
			rows := char.GetMetricsProto().Resources
			var source *proto.ResourceMetrics
			var total float64
			for _, r := range rows {
				if r.Type != proto.ResourceType_ResourceTypeMana {
					continue
				}
				if r.Gain > 0 {
					total += r.ActualGain
				}
				if r.Id.GetSpellId() == 29166 && r.Gain > 0 {
					source = r
				}
			}
			if source == nil || source.ActualGain <= 1 {
				t.Fatalf("missing visible Innervate gain: %v", source)
			}
			wantBonus := offered - baseline
			if math.Abs(source.Gain-wantBonus) > 1e-7 {
				t.Fatalf("bonus = %v, want %v", source.Gain, wantBonus)
			}
			wantTotal := char.CurrentMana() - before
			if math.Abs(total-wantTotal) > 1e-7 {
				t.Fatalf("double-counted regeneration: %v, want %v", total, wantTotal)
			}
			events := source.Events
			aura.Deactivate(sim)
			char.ManaTick(sim)
			for _, r := range char.GetMetricsProto().Resources {
				if r.Id.GetSpellId() == 29166 && r.Gain > 0 && r.Events != events {
					t.Fatal("expired Innervate received a tick")
				}
			}
		})
	}
}
