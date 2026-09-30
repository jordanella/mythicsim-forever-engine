package shaman

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
)

// Rank 6's dummy is hundredths of damage per second of weapon speed (16344: 2810 at 60).
func TestFlametongueDamagePerSecond(t *testing.T) {
	if got := flametongueImbue.EffectN(1).Average(core.CharacterLevel) / 100; got != 28.1 {
		t.Fatalf("Flametongue rank 6 deals %v a second of weapon speed, want 28.1", got)
	}
}

// Rank 4's triggered spell (16361): +333 attack power for 1.5 sec, 3 charges.
func TestWindfuryAttackPowerBuff(t *testing.T) {
	if ap, d, c := windfuryImbue.EffectN(1).Average(core.CharacterLevel), windfuryImbue.Duration(), windfuryImbue.ProcCharges; ap != 333 || d != 1500*time.Millisecond || c != 3 {
		t.Fatalf("Windfury Weapon buff is %v AP for %v, %v charges; want 333 for 1.5s, 3", ap, d, c)
	}
}
