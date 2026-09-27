package core

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core/stats"
)

func TestFrostfireSchools(t *testing.T) {
	attacker := &Unit{Level: 60, PseudoStats: stats.NewPseudoStats()}
	defender := &Unit{Level: 60, PseudoStats: stats.NewPseudoStats()}
	spell := &Spell{Unit: attacker, SpellSchool: SpellSchoolFrostfire, SchoolIndex: stats.SchoolIndexFire, ClassSpellMask: 1}
	for _, resist := range [][2]float64{{150, 30}, {30, 150}, {150, 0}, {90, 90}} {
		defender.stats[stats.FireResistance], defender.stats[stats.FrostResistance] = resist[0], resist[1]
		defender.stats[stats.Strength] = 999 // The old mixed-school fallback accidentally read Strength.
		attacker.stats[stats.SpellPiercing] = 10
		want := max(0, min(resist[0], resist[1])-10) / 300
		if got := defender.resistCoeff(spell, attacker, true); math.Abs(got-want) > 1e-9 {
			t.Errorf("resists %v: got %v, want %v", resist, got, want)
		}
	}
	for _, power := range [][2]float64{{50, 100}, {100, 50}, {100, 100}} {
		attacker.stats[stats.FireDamage], attacker.stats[stats.FrostDamage] = power[0], power[1]
		if got := spell.SpellSchoolBonusDamage(); got != 100 {
			t.Errorf("power %v: got %v, want 100 once", power, got)
		}
	}
	attacker.PseudoStats.SchoolBonusHitChance[stats.SchoolIndexFire] = 5
	attacker.PseudoStats.SchoolBonusHitChance[stats.SchoolIndexFrost] = 5
	if got := spell.SpellHitChance(defender); got != .05 {
		t.Errorf("Elemental Precision hit = %v, want .05 once", got)
	}
	at := &AttackTable{Attacker: attacker, Defender: defender, DamageDealtMultiplier: 1, DamageTakenMultiplier: 1}
	spell.DamageMultiplier, spell.DamageMultiplierAdditive = 1, 1
	for _, multipliers := range [][2]float64{{1.1, 1}, {1, 1.1}, {1.1, 1.1}} {
		defender.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexFire] = multipliers[0]
		defender.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexFrost] = multipliers[1]
		if got := spell.TargetDamageMultiplier(nil, at, false); got != 1.1 {
			t.Errorf("target school multipliers %v: got %v, want 1.1 once", multipliers, got)
		}
		attacker.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire] = multipliers[0]
		attacker.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFrost] = multipliers[1]
		if got := spell.AttackerDamageMultiplier(at, false); got != 1.1 {
			t.Errorf("attacker school multipliers %v: got %v, want 1.1 once", multipliers, got)
		}
	}
}
