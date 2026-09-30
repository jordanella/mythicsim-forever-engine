package warlock

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

func brandTestSim(t *testing.T, summon proto.WarlockOptions_Summon, points int, owners int) (*core.Simulation, []*Warlock) {
	t.Helper()
	// Brand is the fourteenth entry in the Demonology tree. No other damage talents.
	talents := "-0000000000000" + string(rune('0'+points))
	player := core.WithSpec(&proto.Player{
		Race: proto.Race_RaceGnome, Class: proto.Class_ClassWarlock,
		Equipment: &proto.EquipmentSpec{}, Consumables: &proto.ConsumesSpec{},
		TalentsString: talents, Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{Summon: summon}}}})
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	if owners == 2 {
		raid.Parties[0].Players = append(raid.Parties[0].Players, player)
	}
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Targets = append(encounter.Targets, encounter.Targets[0])
	sim := core.NewSim(&proto.RaidSimRequest{SimOptions: &proto.SimOptions{RandomSeed: 100}, Raid: raid, Encounter: encounter}, simsignals.CreateSignals())
	sim.Reset()
	var warlocks []*Warlock
	for _, agent := range sim.Raid.Parties[0].Players {
		w := agent.(WarlockAgent).GetWarlock()
		if int(w.Talents.DemonicBrand) != points {
			t.Fatalf("fixture Brand points = %d, want %d", w.Talents.DemonicBrand, points)
		}
		warlocks = append(warlocks, w)
	}
	return sim, warlocks
}

func brandApply(sim *core.Simulation, w *Warlock, target *core.Unit) {
	w.SearingPain.CalcAndDealDamage(sim, target, 1, w.SearingPain.OutcomeAlwaysHit)
}

func TestDemonicBrandPetHits(t *testing.T) {
	for _, tc := range []struct {
		name              string
		summon            proto.WarlockOptions_Summon
		attackID, brandID int32
		school            core.SpellSchool
	}{
		{"Firebolt", proto.WarlockOptions_Imp, 11763, 1293698, core.SpellSchoolFire},
		{"LashOfPain", proto.WarlockOptions_Succubus, 11780, 1293697, core.SpellSchoolShadow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, ws := brandTestSim(t, tc.summon, 3, 1)
			w, target := ws[0], sim.Encounter.AllTargetUnits[0]
			pet := w.ActivePet
			attack := pet.GetSpell(core.ActionID{SpellID: tc.attackID})
			if attack == nil {
				t.Fatalf("missing pet attack %d", tc.attackID)
			}
			brand := pet.GetSpell(core.ActionID{SpellID: tc.brandID})
			if brand == nil || brand.SpellSchool != tc.school {
				t.Fatal("wrong Brand event or school")
			}
			aura := w.DemonicBrandAuras.Get(target)
			// A missed Searing Pain must not apply the debuff.
			w.SearingPain.CalcAndDealDamage(sim, target, 1, w.SearingPain.OutcomeAlwaysMiss)
			if aura.IsActive() {
				t.Fatal("missed Searing Pain applied Brand")
			}
			brandApply(sim, w, target)
			if aura.GetStacks() != 6 {
				t.Fatal("rank three must grant six charges")
			}
			attack.CalcAndDealDamage(sim, target, 1, attack.OutcomeAlwaysMiss)
			if aura.GetStacks() != 6 {
				t.Fatal("missed pet hit consumed a charge")
			}
			// Owner damage must not consume the pet's charges.
			w.ShadowBolt.CalcAndDealDamage(sim, target, 1, w.ShadowBolt.OutcomeAlwaysHit)
			if aura.GetStacks() != 6 {
				t.Fatal("owner consumed Brand")
			}
			// Low hit cannot cause a second miss roll on the triggered damage.
			pet.AddStatDynamic(sim, stats.SpellHitPercent, -100)
			for i := 0; i < 7; i++ {
				attack.CalcAndDealDamage(sim, target, 1, attack.OutcomeAlwaysHit)
			}
			metrics := brand.SpellMetrics[target.UnitIndex]
			if metrics.Hits != 6 || metrics.Misses != 0 || metrics.Crits != 0 {
				t.Fatalf("Brand outcomes = %+v, want six hits, no misses or crits", metrics)
			}
			if aura.IsActive() || pet.DemonicBrandAura.IsActive() {
				t.Fatal("spent Brand still active")
			}
		})
	}
}

func TestDemonicBrandTargetAndOwnerIsolation(t *testing.T) {
	sim, ws := brandTestSim(t, proto.WarlockOptions_Imp, 1, 2)
	w, other := ws[0], ws[1]
	a, b := sim.Encounter.AllTargetUnits[0], sim.Encounter.AllTargetUnits[1]
	firebolt := w.Imp.GetSpell(core.ActionID{SpellID: 11763})
	otherBolt := other.Imp.GetSpell(core.ActionID{SpellID: 11763})
	brandApply(sim, w, a)
	firebolt.CalcAndDealDamage(sim, b, 1, firebolt.OutcomeAlwaysHit)
	otherBolt.CalcAndDealDamage(sim, a, 1, otherBolt.OutcomeAlwaysHit)
	if w.DemonicBrandAuras.Get(a).GetStacks() != 2 {
		t.Fatal("wrong target or another owner's pet consumed Brand")
	}
	brandApply(sim, w, b)
	firebolt.CalcAndDealDamage(sim, a, 1, firebolt.OutcomeAlwaysHit)
	if w.DemonicBrandAuras.Get(a).GetStacks() != 1 || w.DemonicBrandAuras.Get(b).GetStacks() != 2 {
		t.Fatal("target charges not independent")
	}
	brandApply(sim, w, a)
	if w.DemonicBrandAuras.Get(a).GetStacks() != 2 {
		t.Fatal("Searing Pain did not refresh charges")
	}
	if w.DemonicBrandAuras.Get(a).Duration != 10*time.Second {
		t.Fatal("wrong duration")
	}
	w.DemonicBrandAuras.Get(a).Deactivate(sim)
	firebolt.CalcAndDealDamage(sim, a, 1, firebolt.OutcomeAlwaysHit)
	brand := w.Imp.GetSpell(core.ActionID{SpellID: 1293698})
	if brand.SpellMetrics[a.UnitIndex].Hits != 1 {
		t.Fatal("inactive Brand triggered")
	}
}

func TestDemonicBrandMeleeAndRanks(t *testing.T) {
	for points := 0; points <= 3; points++ {
		sim, ws := brandTestSim(t, proto.WarlockOptions_Succubus, points, 1)
		w, target := ws[0], sim.Encounter.AllTargetUnits[0]
		if points == 0 {
			if w.Succubus.GetSpell(core.ActionID{SpellID: 1293697}) != nil || w.Succubus.DemonicBrandAura != nil {
				t.Fatal("Brand registered without talent")
			}
			continue
		}
		brandApply(sim, w, target)
		aura := w.DemonicBrandAuras.Get(target)
		if aura.GetStacks() != int32(2*points) {
			t.Fatalf("rank %d: got %d charges", points, aura.GetStacks())
		}
		melee := w.Succubus.AutoAttacks.MHAuto()
		melee.CalcAndDealDamage(sim, target, 1, melee.OutcomeAlwaysHit)
		if aura.GetStacks() != int32(2*points-1) {
			t.Fatal("melee did not consume Brand")
		}
		lash := w.Succubus.GetSpell(core.ActionID{SpellID: 11780})
		lash.DealPeriodicDamage(sim, lash.CalcDamage(sim, target, 1, lash.OutcomeAlwaysHit))
		if aura.GetStacks() != int32(2*points-1) {
			t.Fatal("periodic damage consumed Brand")
		}
	}
}

func TestDemonicBrandUsesMatchingOwnerSpellPower(t *testing.T) {
	for _, summon := range []proto.WarlockOptions_Summon{proto.WarlockOptions_Imp, proto.WarlockOptions_Succubus} {
		sim, ws := brandTestSim(t, summon, 3, 1)
		w, target := ws[0], sim.Encounter.AllTargetUnits[0]
		id, matching, other := int32(1293697), stats.ShadowDamage, stats.FireDamage
		if summon == proto.WarlockOptions_Imp {
			id, matching, other = 1293698, stats.FireDamage, stats.ShadowDamage
		}
		spell := w.ActivePet.GetSpell(core.ActionID{SpellID: id})
		spell.Flags |= core.SpellFlagIgnoreResists | core.SpellFlagIgnoreModifiers
		damage := func() float64 {
			sim.Reseed(123)
			before := spell.SpellMetrics[target.UnitIndex].TotalDamage
			spell.Cast(sim, target)
			return spell.SpellMetrics[target.UnitIndex].TotalDamage - before
		}
		base := damage()
		w.AddStatDynamic(sim, other, 100)
		if got := damage(); math.Abs(got-base) > 1e-6 {
			t.Fatalf("wrong school changed Brand: %f vs %f", got, base)
		}
		w.AddStatDynamic(sim, matching, 100)
		if got := damage() - base; math.Abs(got-7.8) > 1e-6 {
			t.Fatalf("school coefficient = %f, want 7.8 per 100 SP", got)
		}
	}
}
