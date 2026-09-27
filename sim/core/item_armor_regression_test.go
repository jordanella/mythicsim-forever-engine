//go:build with_db

package core

import (
	"github.com/wowsims/forever/sim/core/stats"
	"testing"
)

// Check the embedded binary used by real simulations, not only generated JSON.
func TestForeverReferenceArmor(t *testing.T) {
	for id, want := range map[int32]float64{10038: 48, 12632: 218, 12640: 565, 14136: 81, 14146: 60, 14152: 96, 14153: 96, 15062: 148, 15063: 103, 15065: 142, 16711: 115, 16723: 341} {
		item := NewItem(ItemSpec{ID: id})
		if got := item.Stats[stats.Armor]; got != want {
			t.Errorf("%d %s armor = %v, want %v", id, item.Name, got, want)
		}
	}
	cape := NewItem(ItemSpec{ID: 13397})
	if cape.Stats[stats.Armor] != 43 || cape.Stats[stats.BonusArmor] != 50 {
		t.Fatalf("Stoneskin armor split: %v", cape.Stats)
	}
}
