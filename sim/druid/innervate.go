package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

var innervateRank = spellData.Innervate.Highest()

func (druid *Druid) registerInnervateCD() {
	innervateTarget := druid.GetUnit(druid.SelfBuffs.InnervateTarget)
	if innervateTarget == nil {
		innervateTarget = &druid.Unit
	}
	innervateTargetChar := druid.Env.Raid.GetPlayerFromUnit(innervateTarget).GetCharacter()

	actionID := core.ActionID{SpellID: innervateRank.ID, Tag: druid.Index}

	innervateAura := druid.innervateAura(innervateTargetChar)

	innervateSpell := druid.RegisterSpell(Humanoid|Moonkin|Tree, core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    innervateRank.SpellSchool(),
		DefenseType:    innervateRank.DefenseTypeCore(),
		ClassSpellMask: DruidSpellInnervate,
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		MaxRange:       float64(innervateRank.MaxRange),

		ManaCost: innervateRank.ManaCost(),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: innervateRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: max(innervateRank.Cooldown(), innervateRank.CategoryCooldown()),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			// If the target already has another Innervate, don't cast.
			return !innervateTarget.HasActiveAuraWithTag(buffs.InnervatesCategory)
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			innervateAura.Activate(sim)
		},
	})

	druid.AddMajorCooldown(core.MajorCooldown{
		Spell: innervateSpell.Spell,
		Type:  core.CooldownTypeMana,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			// Require manual APL usage.
			return false
		},
	})
}

// The generated Innervate aura states no regen (auras 134 and 110 are left out), so the druid's own
// cast attaches it: full spirit regen while casting at 5x rate, as core's raid-config driver does.
// A second druid innervating the same target reuses the first one's aura and its hooks.
// Its mana is regen (no energize effect), so it lands in the regen metrics and makes no threat.
func (druid *Druid) innervateAura(char *core.Character) *core.Aura {
	if aura := char.GetAura("Innervates (Player)"); aura != nil {
		return aura
	}

	return buffs.AttachInnervateRegen(char, buffs.InnervatesAura(&char.Unit, true, 0))
}
