package druid

import (
	"github.com/wowsims/forever/sim/core"
)

func init() {
	// Idol of the Moon
	core.NewItemEffect(23197, func(agent core.Agent) {
		character := agent.GetCharacter()
		aura := core.MakePermanent(character.RegisterAura(core.Aura{
			Label: "Improved Moonfire",
		}).AttachSpellMod(core.SpellModConfig{
			ClassMask:  DruidSpellMoonfire,
			Kind:       core.SpellMod_BaseDamage_Flat,
			FloatValue: 33.0,
		}))

		// TODO: this registers the swap proc against 32330, not 23197. That item is not in
		// the Forever database either, so the registration is dead whichever was intended.
		character.ItemSwap.RegisterProc(32330, aura)
	})

	// Idol of Ferocity
	// https://www.wowhead.com/forever/item=22397/idol-of-ferocity
	//
	// Reduces the energy cost of Claw and Rake by 2 (27851); Classic's took 3. The sim has no Claw.
	core.NewItemEffect(22397, func(agent core.Agent) {
		agent.GetCharacter().AddStaticMod(core.SpellModConfig{
			ClassMask: DruidSpellRake,
			Kind:      core.SpellMod_PowerCost_Flat,
			IntValue:  -2,
		})
	})

	// Idol of Brutality
	// https://www.wowhead.com/forever/item=23198/idol-of-brutality
	//
	// Reduces the rage cost of Maul and Swipe by 2 (28855: -20 in the client's tenths of rage, the
	// scale Ferocity's -10 a rank is on); Classic's took 3. The client's mask also names Primal Bite, as
	// Ferocity's does.
	core.NewItemEffect(23198, func(agent core.Agent) {
		agent.GetCharacter().AddStaticMod(core.SpellModConfig{
			ClassMask: DruidSpellMaul | DruidSpellSwipe | DruidSpellPrimalBite,
			Kind:      core.SpellMod_PowerCost_Flat,
			IntValue:  -2,
		})
	})

	// Forever Wolfshead Helm (8345, effect 17768): the bonus belongs to
	// Tiger's Fury and Enrage, not to entering a form.
	core.NewItemEffect(8345, func(agent core.Agent) {
		druid := agent.(DruidAgent).GetDruid()
		core.MakePermanent(druid.RegisterAura(core.Aura{
			Label:    "Wolfshead Helm",
			ActionID: core.ActionID{SpellID: 17768},
			OnGain: func(_ *core.Aura, _ *core.Simulation) {
				druid.WolfsheadTigersFuryEnergy += 20
				druid.WolfsheadEnrageRage += 5
			},
			OnExpire: func(_ *core.Aura, _ *core.Simulation) {
				druid.WolfsheadTigersFuryEnergy -= 20
				druid.WolfsheadEnrageRage -= 5
			},
		}))
	})
}
