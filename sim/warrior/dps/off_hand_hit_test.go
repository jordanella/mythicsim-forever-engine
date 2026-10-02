package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// The reference Fury build: Dual Wield Specialization 5/5.
const dualWieldSpecTalents = "20315003-250500035151310051"

// Client 1.60.1.70170: "Queueing Heroic Strike will no longer increase off-hand hit chance". The sim never
// did: Heroic Strike's own swing rolls on the special attack table, but the flag that drops the dual wield
// penalty is only lifted for that swing, so an off-hand auto lands against the same table whether or not a
// Heroic Strike is queued.
func TestQueuedHeroicStrikeDoesNotRaiseOffHandHitChance(t *testing.T) {
	sim, war := newRageWarrior(t, DualWieldGear.GearSet, dualWieldSpecTalents)
	table := war.AttackTables[sim.Encounter.ActiveTargetUnits[0].UnitIndex]
	offHand := war.AutoAttacks.OHAuto()
	mainHand := war.AutoAttacks.MHAuto()

	offBefore := offHand.GetPhysicalMissChance(table)
	mainBefore := mainHand.GetPhysicalMissChance(table)

	queue := war.GetAuraByID(core.ActionID{SpellID: 25286}.WithTag(1))
	if queue == nil {
		t.Fatal("the Heroic Strike queue aura is not registered")
	}
	queue.Activate(sim)
	if !queue.IsActive() {
		t.Fatal("the Heroic Strike queue did not activate")
	}
	if war.PseudoStats.DisableDWMissPenalty {
		t.Error("queueing Heroic Strike lifted the dual wield miss penalty for every swing")
	}

	if got := offHand.GetPhysicalMissChance(table); got != offBefore {
		t.Errorf("an off-hand swing's miss chance went from %v to %v with Heroic Strike queued", offBefore, got)
	}
	if got := mainHand.GetPhysicalMissChance(table); got != mainBefore {
		t.Errorf("a main-hand auto's miss chance went from %v to %v with Heroic Strike queued", mainBefore, got)
	}
}

// Dual Wield Specialization's hit chance (spell 23584, curve 2/4/6/8/10) is the off-hand's alone. The
// 70170 client lists it as a known issue that the aura currently reaches both hands; the sim applies the
// intended off-hand-only bonus, so the two hands differ by exactly the talent's 10% at 5/5.
func TestDualWieldSpecializationHitChanceIsOffHandOnly(t *testing.T) {
	sim, war := newRageWarrior(t, DualWieldGear.GearSet, dualWieldSpecTalents)
	table := war.AttackTables[sim.Encounter.ActiveTargetUnits[0].UnitIndex]

	mainHit := war.AutoAttacks.MHAuto().PhysicalHitChance(table)
	offHit := war.AutoAttacks.OHAuto().PhysicalHitChance(table)

	if !core.WithinToleranceFloat64(0.10, offHit-mainHit, 0.000001) {
		t.Errorf("off hand hit chance %v against main hand %v: the difference is %v, want 0.10", offHit, mainHit, offHit-mainHit)
	}
}
