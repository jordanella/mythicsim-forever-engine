package priest

import "github.com/wowsims/forever/sim/core"

// A Gnome priest's Eureka! (1259823) in client 1.60.1.70170, which takes Devouring Plague, Shadow Word:
// Pain and Renew off every effect and Holy Fire's dot off the periodic one, and puts Shadow Word: Death on
// the cost and damage lists.
//
//   - Effects 0 and 1 (cost, damage): the direct spells, and the channels Mind Flay, Penance and
//     Starshards, which "do not count as periodics".
//   - Effect 2 (periodic): the three channels only.
//
// Holy Fire is a hit and a dot in one spell here, so the dot takes the bonus back (Eureka! in core).
func (priest *Priest) EurekaSpells() core.EurekaSpells {
	direct := PriestSpellHolyFire | PriestSpellHolyNova | PriestSpellMindBlast | PriestSpellShadowWordDeath |
		PriestSpellSmite
	channels := PriestSpellMindFlay | PriestSpellPenance | PriestSpellStarshards
	return core.EurekaSpells{
		Cost: direct | channels,
		// Penance's first bolt lands with the cast, so it is on the direct list too.
		Damage: direct | PriestSpellPenance,
		Tick:   channels,
	}
}
