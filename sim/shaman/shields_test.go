package shaman

import "testing"

// The orb the sim fires must keep the client's cannot-crit attribute; the damage spell picks its
// hit table from it.
func TestLightningShieldOrbCannotCrit(t *testing.T) {
	if !lightningShieldOrb.CannotCrit() {
		t.Fatalf("Lightning Shield orb %d lost its cannot-crit attribute", lightningShieldOrb.ID)
	}
}
