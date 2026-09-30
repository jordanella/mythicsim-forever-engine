package priest

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

// Dark Sacrifice is new in Forever: five ranks (1277324-1277328, trained 20 to 60), a free instant
// self buff on a 10 minute cooldown that turns health into mana over 15 sec. The client row carries the
// base: rank 5 ticks 320 every 3 sec, five ticks. The tooltip's "(1640 + Spirit) Mana" adds Spirit on
// top, which the row does not state; a beta combat log settles how: Papa (report 40, rank 1, Spirit 72)
// took 80 health and gained 94, 94, 95, 94, 95 mana a tick, so each tick pays base + Spirit / 5. The
// health cost is not modelled. The cooldown manager uses it once the whole gain fits in the mana bar.
var DarkSacrificeRank = spellData.DarkSacrifice.Highest()

func (priest *Priest) registerDarkSacrificeSpell() {
	rank := DarkSacrificeRank
	effect := rank.ProcEnergizeEffect()
	metrics := priest.NewManaMetrics(core.ActionID{SpellID: rank.ID})
	tick := func() float64 {
		return effect.Average(priest.Level) + priest.GetStat(stats.Spirit)/5
	}

	config := spelldata.SpellConfig(&priest.Unit, rank, spelldata.Flags(core.SpellFlagAPL))
	config.ProcMask = core.ProcMaskEmpty
	config.Hot = spelldata.DotConfig(rank, effect)
	config.Hot.SelfOnly = true
	config.Hot.OnTick = func(sim *core.Simulation, _ *core.Unit, _ *core.Dot) {
		priest.AddMana(sim, tick(), metrics)
	}
	config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		spell.SelfHot().Apply(sim)
	}
	ticks := float64(config.Hot.NumberOfTicks)

	priest.AddMajorCooldown(core.MajorCooldown{
		Spell: priest.RegisterSpell(config),
		Type:  core.CooldownTypeMana,
		ShouldActivate: func(_ *core.Simulation, _ *core.Character) bool {
			return priest.MaxMana()-priest.CurrentMana() >= tick()*ticks
		},
	})
}
