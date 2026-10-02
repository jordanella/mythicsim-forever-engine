package arenalib

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// ARENA_ROTATION_AUDIT=1 turns a spec's arena test into a check of its rotation files rather than
// a ranking. For every community talent build on the first gear set it reports two things:
//
//   - what the sim says is wrong with each rotation (its APL validation warnings and errors), which
//     is how a rotation naming a spell the class no longer has - Tiger's Fury after client 70170 -
//     shows up;
//   - every spell the build can cast that none of the spec's rotations ever casts. A new talent
//     ability the rotations were never taught about (Shifting Power) lands here, and so does a
//     button nobody has a use for, which is fine and is for a person to judge.
//
// It changes nothing and fails nothing: it is a list to read.
const rotationAuditEnv = "ARENA_ROTATION_AUDIT"

const rotationAuditIterations = int32(20)

func auditRotations(t *testing.T, spec Spec, talents []TalentBuild, gearSets []string, rotations []string) {
	environment := consumesFor(spec.Role, spec.ClassImbues)
	gear := gearSets[0]
	for _, talent := range talents {
		castable := map[string]string{}
		cast := map[string]bool{}
		for _, rotation := range rotations {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Logf("AUDIT %s | %s | %s: panic: %.200v", spec.Dir, talent.Name, rotation, r)
					}
				}()
				raid := raidFor(spec, talent, gear, rotation, environment)

				statsResult := core.ComputeStats(&proto.ComputeStatsRequest{Raid: raid, Encounter: core.MakeSingleTargetEncounter(0)})
				if statsResult.ErrorResult != "" {
					t.Logf("AUDIT %s | %s | %s: %s", spec.Dir, talent.Name, rotation, statsResult.ErrorResult)
					return
				}
				player := statsResult.RaidStats.Parties[0].Players[0]
				for _, problem := range validations(player.RotationStats) {
					t.Logf("AUDIT %s | %s | %s: rotation %s", spec.Dir, talent.Name, rotation, problem)
				}
				for _, spell := range player.Metadata.GetSpells() {
					if spell.IsCastable && !spell.IsFriendly {
						castable[actionKey(spell.Id)] = spellName(spell.Id)
					}
				}

				simResult := core.RunRaidSim(&proto.RaidSimRequest{
					Raid:       raid,
					Encounter:  core.MakeSingleTargetEncounter(0),
					SimOptions: &proto.SimOptions{Iterations: rotationAuditIterations, RandomSeed: 101},
				})
				if simResult.Error != nil {
					t.Logf("AUDIT %s | %s | %s: %s", spec.Dir, talent.Name, rotation, simResult.Error.Message)
					return
				}
				for _, action := range simResult.RaidMetrics.Parties[0].Players[0].Actions {
					for _, target := range action.Targets {
						if target.Casts > 0 {
							cast[actionKey(action.Id)] = true
						}
					}
				}
			}()
		}
		// By name, so a spell counts as used when any of its ranks was cast.
		usedNames := map[string]bool{}
		for key, name := range castable {
			if cast[key] {
				usedNames[name] = true
			}
		}
		unusedNames := map[string]bool{}
		for _, name := range castable {
			if !usedNames[name] {
				unusedNames[name] = true
			}
		}
		unused := slices.Sorted(maps.Keys(unusedNames))
		if len(unused) > 0 {
			t.Logf("AUDIT %s | %s: never cast by any of %s: %s", spec.Dir, talent.Name, strings.Join(rotations, ", "), strings.Join(unused, "; "))
		}
	}
}

func validations(stats *proto.APLStats) []string {
	var out []string
	add := func(list []*proto.APLValidation) {
		for _, v := range list {
			if v.LogLevel >= proto.LogLevel_Warning {
				out = append(out, fmt.Sprintf("%s: %s", v.LogLevel, v.Validation))
			}
		}
	}
	for _, a := range stats.GetPrepullActions() {
		add(a.Validations)
	}
	for _, a := range stats.GetPriorityList() {
		add(a.Validations)
	}
	for _, u := range stats.GetUuidValidations() {
		add(u.Validations)
	}
	for _, g := range stats.GetGroups() {
		for _, a := range g.GetActions() {
			add(a.Validations)
		}
	}
	return out
}

// A spell's id without its tag: metrics split a cast by tag (Eviscerate by combo points), the spell
// list does not.
func actionKey(id *proto.ActionID) string {
	return fmt.Sprintf("%d/%d", id.GetSpellId(), id.GetOtherId())
}

func spellName(id *proto.ActionID) string {
	if spell := spelldata.Find(id.GetSpellId()); spell != spelldata.Nil && spell.Name != "" {
		return spell.Name
	}
	return id.String()
}

func rotationAuditOn() bool { return os.Getenv(rotationAuditEnv) != "" }
