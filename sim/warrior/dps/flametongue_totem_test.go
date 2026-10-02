package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// PartyBuffs.FlametongueTotem is another shaman's Flametongue Totem (patch 70), so any melee spec can
// assume one. Its party aura (15036) states 100% on a landed melee auto attack and the tooltip names the
// main hand: a main-hand auto adds the fire hit 16389 states, and nothing else a dual wielder swings does.
func TestPartyFlametongueTotemHitsOnMainHandAutoAttacksOnly(t *testing.T) {
	setup := func(t *testing.T, party *proto.PartyBuffs) (*core.Simulation, *DpsWarrior) {
		t.Helper()

		items := make([]*proto.ItemSpec, 16)
		for i := range items {
			items[i] = &proto.ItemSpec{}
		}
		items[14] = &proto.ItemSpec{Id: 3414} // Crested Scepter, a 2.6 speed mace
		items[15] = &proto.ItemSpec{Id: 9384} // Stonevault Shiv, a 1.5 speed dagger
		player := &proto.Player{
			Name:          "Flametongue",
			Race:          proto.Race_RaceOrc,
			Class:         proto.Class_ClassWarrior,
			Equipment:     &proto.EquipmentSpec{Items: items},
			TalentsString: FuryTalents,
			Spec:          DefaultOptions,
			Rotation:      core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "dps_reck").Rotation,
		}
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, party, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{RandomSeed: 1},
		}, simsignals.CreateSignals())
		sim.Reset()
		return sim, sim.Raid.Parties[0].Players[0].(*DpsWarrior)
	}

	t.Run("a party that brings no totem leaves it off", func(t *testing.T) {
		_, war := setup(t, &proto.PartyBuffs{})
		if war.GetAura("Flametongue Totem Trigger") != nil || war.GetAura("Flametongue Totem") != nil {
			t.Fatal("a Flametongue Totem is registered without the party buff")
		}
	})

	// The trigger hears each landed hit the way the sim delivers it; each offer is one hit of the kind.
	const offers = 200
	for _, row := range []struct {
		name     string
		procMask core.ProcMask
		hits     int
	}{
		{"a main-hand auto adds the hit", core.ProcMaskMeleeMHAuto, offers},
		{"an off-hand auto adds none", core.ProcMaskMeleeOHAuto, 0},
		{"a main-hand special adds none, since the row states autos", core.ProcMaskMeleeMHSpecial, 0},
		{"an off-hand special adds none", core.ProcMaskMeleeOHSpecial, 0},
		{"a ranged auto adds none", core.ProcMaskRangedAuto, 0},
		{"a spell adds none", core.ProcMaskSpellDamage, 0},
	} {
		t.Run(row.name, func(t *testing.T) {
			sim, war := setup(t, &proto.PartyBuffs{FlametongueTotem: true})
			trigger, totem := war.GetAura("Flametongue Totem Trigger"), war.GetAura("Flametongue Totem")
			if trigger == nil || totem == nil || !trigger.IsActive() || !totem.IsActive() {
				t.Fatalf("trigger %v, totem %v; want both registered and up at the pull", trigger, totem)
			}
			attack := war.GetSpell(core.ActionID{SpellID: 16389})
			if attack == nil {
				t.Fatal("the totem's hit (16389) is not registered")
			}
			target := sim.Encounter.AllTargetUnits[0]
			spell := &core.Spell{ProcMask: row.procMask}
			for i := 0; i < offers; i++ {
				trigger.OnSpellHitDealt(trigger, sim, spell, &core.SpellResult{Target: target, Outcome: core.OutcomeHit, Damage: 100})
			}
			metrics := &attack.SpellMetrics[target.UnitIndex]
			if got := int(metrics.Hits + metrics.Crits + metrics.Misses); got != row.hits {
				t.Errorf("%d hits for %d offers, want %d", got, offers, row.hits)
			}
		})
	}
}
