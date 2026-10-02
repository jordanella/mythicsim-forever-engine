package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// A Fury build with Dual Wield Specialization 5/5 and no Furious Precision.
const dualWieldSpecTalents = "20315003-25050005151010501"

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

// The 70170 hotfixes moved Dual Wield Specialization's off-hand hit chance to Furious Precision (spell
// 1323963, 4/7/10 at 1/2/3 ranks): it is the off hand's alone, so the two hands differ by exactly the
// talent's 10% at 3/3, and Dual Wield Specialization on its own no longer separates them. The bonus is read
// off the spells, because PhysicalHitChance clamps a hand's chance at zero (the target's hit suppression).
func TestFuriousPrecisionHitChanceIsOffHandOnly(t *testing.T) {
	handsDiffer := func(talents string) float64 {
		_, war := newRageWarrior(t, DualWieldGear.GearSet, talents)
		return war.AutoAttacks.OHAuto().BonusHitPercent - war.AutoAttacks.MHAuto().BonusHitPercent
	}

	// DpsTalents spends Furious Precision 3/3 beside Dual Wield Specialization 5/5.
	if got := handsDiffer(DpsTalents); !core.WithinToleranceFloat64(10, got, 0.000001) {
		t.Errorf("with Furious Precision 3/3 the hands differ by %v hit percent, want 10", got)
	}
	if got := handsDiffer(dualWieldSpecTalents); got != 0 {
		t.Errorf("Dual Wield Specialization alone separates the hands by %v hit percent, want 0", got)
	}
}
