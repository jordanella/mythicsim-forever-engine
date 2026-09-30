package priest

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Chastise is new in Forever: five ranks (1277331-1277335, trained 20 to 60), an instant Holy nuke
// on a 2 minute cooldown the ranks share (category 2491). Everything comes off the client rows: rank 5
// deals 289 +-12% (272-306) at .143 spell power for 225 mana, 20 yards, and roots for 2 sec, which
// the sim ignores. The row only hits humanoids (TargetCreatureType 64) and cannot be cast in
// Shadowform (StanceExclude). No default rotation casts it; an APL can.
var ChastiseRankMap = spellData.Chastise

func (priest *Priest) registerChastiseSpell(rank *spelldata.Spell) {
	config := spelldata.SpellConfig(&priest.Unit, rank, spelldata.Magic(core.ProcMaskSpellDamage))
	config.ClassSpellMask = PriestSpellChastise
	config.ExtraCastCondition = func(_ *core.Simulation, target *core.Unit) bool {
		return target.MobType == proto.MobType_MobTypeHumanoid
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealDamage(sim, target, rank.DamageEffect().Roll(sim, core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
	}
	priest.RegisterSpell(config)
}
