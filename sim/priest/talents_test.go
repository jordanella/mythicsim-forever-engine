package priest

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

func penanceCost(t *testing.T, talents string) float64 {
	player := core.WithSpec(&proto.Player{
		Race:          proto.Race_RaceUndead,
		Class:         proto.Class_ClassPriest,
		Equipment:     &proto.EquipmentSpec{},
		Consumables:   &proto.ConsumesSpec{},
		TalentsString: talents,
		Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_DpsPriest{DpsPriest: &proto.DpsPriest{Options: &proto.DpsPriest_Options{ClassOptions: &proto.PriestOptions{}}}})
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	priest := sim.Raid.Parties[0].Players[0].(PriestAgent).GetPriest()
	penance := priest.GetSpell(core.ActionID{SpellID: spellData.Penance.Highest().ID})
	if penance == nil {
		t.Fatalf("%s: no Penance", talents)
	}
	return penance.Cost.GetCurrentCost()
}

// Improved Healing 3/3 takes 15% off Penance.
func TestImprovedHealingDiscountsPenance(t *testing.T) {
	base := penanceCost(t, "504020031305001-13505100202-50002")
	discounted := penanceCost(t, "504020031305001-13505100032-50002")
	if !core.WithinToleranceFloat64(base*0.85, discounted, 1e-6) {
		t.Errorf("Penance costs %v with Improved Healing 3, %v without; want 15%% off", discounted, base)
	}
}

// Chastise hits humanoids only, and its five ranks share one 2 minute cooldown.
func TestChastiseHumanoidsAndSharedCooldown(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Race:        proto.Race_RaceUndead,
		Class:       proto.Class_ClassPriest,
		Equipment:   &proto.EquipmentSpec{},
		Consumables: &proto.ConsumesSpec{},
		Rotation:    &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_DpsPriest{DpsPriest: &proto.DpsPriest{Options: &proto.DpsPriest_Options{ClassOptions: &proto.PriestOptions{}}}})
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Targets = []*proto.Target{
		{Name: "beast", Level: 63, MobType: proto.MobType_MobTypeBeast},
		{Name: "humanoid", Level: 63, MobType: proto.MobType_MobTypeHumanoid},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  encounter,
	}, simsignals.CreateSignals())
	sim.Reset()

	priest := sim.Raid.Parties[0].Players[0].(PriestAgent).GetPriest()
	other, humanoid := sim.Encounter.AllTargetUnits[0], sim.Encounter.AllTargetUnits[1]
	top := priest.GetSpell(core.ActionID{SpellID: 1277335})
	low := priest.GetSpell(core.ActionID{SpellID: 1277331})
	if top.CanCast(sim, other) {
		t.Fatal("Chastise castable on a non-humanoid")
	}
	if !top.Cast(sim, humanoid) {
		t.Fatal("Chastise not castable on a humanoid")
	}
	if low.IsReady(sim) {
		t.Error("rank 1 ready after rank 5 was cast: the ranks should share the cooldown")
	}
	if dealt := top.SpellMetrics[humanoid.UnitIndex].TotalDamage; dealt < 271 {
		t.Errorf("Chastise dealt %.1f, want at least rank 5's 272", dealt)
	}
}

// Dark Sacrifice pays the row's 320 plus a fifth of Spirit on each of its five ticks, as a beta log
// shows at rank 1 (80 + 72 / 5 = 94.4 a tick).
func TestDarkSacrificeTicksBasePlusSpirit(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Race:        proto.Race_RaceUndead,
		Class:       proto.Class_ClassPriest,
		Equipment:   &proto.EquipmentSpec{},
		Consumables: &proto.ConsumesSpec{},
		Rotation:    &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_DpsPriest{DpsPriest: &proto.DpsPriest{Options: &proto.DpsPriest_Options{ClassOptions: &proto.PriestOptions{}}}})
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	priest := sim.Raid.Parties[0].Players[0].(PriestAgent).GetPriest()
	spell := priest.GetSpell(core.ActionID{SpellID: 1277328})
	priest.SpendMana(sim, priest.CurrentMana(), priest.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	spell.Cast(sim, nil)
	hot := spell.SelfHot()
	if hot.BaseTickCount != 5 {
		t.Fatalf("Dark Sacrifice ticks %d times, want 5", hot.BaseTickCount)
	}
	for range 5 {
		hot.TickOnce(sim)
	}
	want := 5 * (320 + priest.GetStat(stats.Spirit)/5)
	if got := priest.CurrentMana(); !core.WithinToleranceFloat64(want, got, 1e-6) {
		t.Errorf("Dark Sacrifice restored %.1f mana, want %.1f", got, want)
	}
}
