package core

import (
	"testing"

	"github.com/wowsims/forever/sim/core/stats"
)

func TestArmorDamageReductionCap(t *testing.T) {
	// Boss attacker: armorConstant = 400 + 85*63 = 5755, so 75% reduction is
	// reached at 3*5755 = 17265 armor.
	attacker := Unit{Type: EnemyUnit, Level: 63}
	tolerance := 0.0001

	modifierForArmor := func(armor float64) float64 {
		defender := Unit{
			Type:         PlayerUnit,
			Level:        60,
			initialStats: stats.Stats{stats.Armor: armor},
			PseudoStats:  stats.NewPseudoStats(),
		}
		defender.stats = defender.initialStats
		return NewAttackTable(&attacker, &defender).GetArmorDamageModifier(nil)
	}

	if modifier := modifierForArmor(11510); !WithinToleranceFloat64(1.0/3.0, modifier, tolerance) {
		t.Fatalf("Expected %f damage taken below the cap, got %f", 1.0/3.0, modifier)
	}
	if modifier := modifierForArmor(17265); !WithinToleranceFloat64(0.25, modifier, tolerance) {
		t.Fatalf("Expected %f damage taken at the cap, got %f", 0.25, modifier)
	}
	if modifier := modifierForArmor(50000); !WithinToleranceFloat64(0.25, modifier, tolerance) {
		t.Fatalf("Expected %f damage taken above the cap, got %f", 0.25, modifier)
	}
}

func TestArmorConstantIsLevel60(t *testing.T) {
	// A level 60 attacker: 400 + 85*60 = 5500, so 5500 armor halves the damage.
	// TBC's 467.5*60 - 22167.5 = 5882.5 would leave 48.3%.
	attacker := Unit{Type: PlayerUnit, Level: 60}
	defender := Unit{Type: EnemyUnit, Level: 63, initialStats: stats.Stats{stats.Armor: 5500}, PseudoStats: stats.NewPseudoStats()}
	defender.stats = defender.initialStats
	if modifier := NewAttackTable(&attacker, &defender).GetArmorDamageModifier(nil); !WithinToleranceFloat64(0.5, modifier, 0.0001) {
		t.Fatalf("Expected 0.5 damage taken, got %f", modifier)
	}
}

func TestGlancingBlowsAreLevel60(t *testing.T) {
	// 40% glances for 65% damage against a boss, 10% per level less below it.
	attacker := Unit{Type: PlayerUnit, Level: 60}
	for level, want := range map[int32][2]float64{63: {0.40, 0.65}, 62: {0.30, 0.85}, 61: {0.20, 0.95}, 60: {0.10, 0.95}} {
		defender := Unit{Type: EnemyUnit, Level: level}
		table := NewAttackTable(&attacker, &defender)
		if table.BaseGlanceChance != want[0] || table.GlanceMultiplier != want[1] {
			t.Errorf("level %d: glance %.2f for %.2f damage, want %.2f for %.2f", level, table.BaseGlanceChance, table.GlanceMultiplier, want[0], want[1])
		}
		// The roll's ends, from Classic's formula at 300 weapon skill.
		gap := 5 * float64(level-60)
		low, high := min(1.3-0.05*gap, 0.91), max(min(1.2-0.03*gap, 0.99), 0.2)
		if !WithinToleranceFloat64(low, table.GlanceMultiplier-table.GlanceSpread, 1e-9) || !WithinToleranceFloat64(high, table.GlanceMultiplier+table.GlanceSpread, 1e-9) {
			t.Errorf("level %d: glance keeps %.2f-%.2f, want %.2f-%.2f", level, table.GlanceMultiplier-table.GlanceSpread, table.GlanceMultiplier+table.GlanceSpread, low, high)
		}
	}
}
