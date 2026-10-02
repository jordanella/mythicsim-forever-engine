package feralbear

import (
	"testing"
	"time"
)

// Client 1.60.1.70170: "Faerie Fire no longer resets your swing timer when used." The sim never reset it
// (Faerie Fire is an instant on the global cooldown and nothing in the cast path touches the swing), so
// this pins that: using it moves neither swing, in Bear Form, with the swing close and far.
func TestFaerieFireDoesNotMoveTheNextSwing(t *testing.T) {
	for _, wait := range []time.Duration{200 * time.Millisecond, 900 * time.Millisecond, 1700 * time.Millisecond} {
		sim, d := newBearInForm(t)
		// A swing has just landed, so the next one is a full weapon speed away.
		d.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime)
		sim.CurrentTime += wait
		d.GCD.Set(sim.CurrentTime)

		mhBefore := d.AutoAttacks.MainhandSwingAt()
		if mhBefore <= sim.CurrentTime {
			t.Fatalf("wait %v: the next swing at %v is not ahead of %v", wait, mhBefore, sim.CurrentTime)
		}
		if !d.FaerieFire.Cast(sim, d.CurrentTarget) {
			t.Fatalf("wait %v: Faerie Fire failed to cast", wait)
		}
		if got := d.AutoAttacks.MainhandSwingAt(); got != mhBefore {
			t.Errorf("wait %v: Faerie Fire moved the next main hand swing from %v to %v", wait, mhBefore, got)
		}
	}
}
