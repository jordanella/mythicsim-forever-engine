package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core/dbcenums"
)

// The four talents read their client rows, which carry the tooltips' numbers (assets/confirmed_talents.json).
func TestTalentRowsMatchTooltips(t *testing.T) {
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"Careful Aim rank 1", spellData.CarefulAim.EffectAt(1).FractionAt(1), 0.2},
		{"Careful Aim rank 5", spellData.CarefulAim.EffectAt(1).FractionAt(5), 1},
		{"Bestial Discipline rank 2", spellData.BestialDiscipline.EffectAt(1).MultiplierAt(2), 1.2},
		{"Surefooted rank 1", spellData.Surefooted.Effect(dbcenums.A_MOD_HIT_CHANCE, 0).ValueAt(1), 1},
		{"Surefooted rank 3", spellData.Surefooted.Effect(dbcenums.A_MOD_HIT_CHANCE, 0).ValueAt(3), 3},
		{"Lone Wolf", spellData.LoneWolf.Effect(dbcenums.A_MOD_DAMAGE_PERCENT_DONE, 127).MultiplierAt(1), 1.2},
	} {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}
