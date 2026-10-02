package warrior

import "github.com/wowsims/forever/sim/core"

// A Gnome warrior's Eureka! (1259813) in client 1.60.1.70170. Effects 0 and 1 name the same abilities, with
// Intercept, Pummel, Revenge, Shield Bash and Spearing Strike newly among them and Rend off both. Effect 2,
// the periodic bonus Rend had, is a dummy now. Deep Wounds, Sunder Armor, Sweeping Strikes, Concussion
// Blow and Victory Rush are on none of the effects.
func (warrior *Warrior) EurekaSpells() core.EurekaSpells {
	abilities := SpellMaskBloodthirst | SpellMaskCleave | SpellMaskExecute | SpellMaskHamstring |
		SpellMaskHeroicStrike | SpellMaskMockingBlow | SpellMaskMortalStrike | SpellMaskOverpower |
		SpellMaskShieldSlam | SpellMaskSlam | SpellMaskThunderClap | SpellMaskWhirlwind | SpellMaskWhirlwindOh |
		SpellMaskIntercept | SpellMaskPummel | SpellMaskRevenge | SpellMaskShieldBash | SpellMaskSpearingStrike
	return core.EurekaSpells{
		Cost:   abilities,
		Damage: abilities,
	}
}
