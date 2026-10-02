package rogue

import "github.com/wowsims/forever/sim/core"

// A Gnome rogue's Eureka! (1259812) in client 1.60.1.70170. Effects 0 and 1 name the same direct
// abilities, Hemorrhage newly among them, and Garrote, Rupture, Cheap Shot and Blade Flurry are off both.
// Effect 2, the periodic bonus Garrote and Rupture had, is a dummy now.
func (rogue *Rogue) EurekaSpells() core.EurekaSpells {
	abilities := RogueSpellAmbush | RogueSpellBackstab | RogueSpellEviscerate | RogueSpellGhostlyStrike |
		RogueSpellGouge | RogueSpellHemorrhage | RogueSpellMutilate | RogueSpellMutilateHit |
		RogueSpellSinisterStrike | RogueSpellRiposte
	return core.EurekaSpells{
		Cost:   abilities,
		Damage: abilities,
	}
}
