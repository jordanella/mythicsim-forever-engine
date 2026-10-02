package warlock

import "github.com/wowsims/forever/sim/core"

// A Gnome warlock's Eureka! (1259821) in client 1.60.1.70170, which takes Corruption, Bane of Agony, Bane
// of Doom and Unstable Affliction off every effect, and Immolate's periodic damage off the periodic one.
//
//   - Effect 0 (cost): the direct spells, the drains and Wrack, Hellfire and Rain of Fire. Haunt is new on
//     it and is not in the sim.
//   - Effect 1 (damage): the direct spells and Rain of Fire (a hit of its tick spell), Hellfire's hits
//     (Hellfire Effect, new on the list), but no longer Drain Life, Drain Soul or any dot.
//   - Effect 2 (periodic): Drain Life, Drain Soul and Wrack, the channels. Immolate's dot is its own spell
//     here (WarlockSpellImmolateDot), so it takes nothing.
//
// The sim deals Hellfire's hits as ticks of the channel, so it is a Tick spell.
func (warlock *Warlock) EurekaSpells() core.EurekaSpells {
	direct := WarlockSpellConflagrate | WarlockSpellDeathCoil | WarlockSpellImmolate | WarlockSpellRainOfFire |
		WarlockSpellSearingPain | WarlockSpellShadowBolt | WarlockSpellShadowBurn | WarlockSpellSoulFire
	channels := WarlockSpellDrainLife | WarlockSpellDrainSoul | WarlockSpellWrack | WarlockSpellHellfire
	return core.EurekaSpells{
		Cost:   direct | channels,
		Damage: direct,
		Tick:   channels,
	}
}
