// Runs every build a spec can be given from what is already in the repository, so the
// leaderboard is a result rather than a guess about which build to show.
//
// The rankings page picks one preset per spec and runs them together in the browser, which
// is fine for one build each and impossible for all of them: the gear sets and rotations
// already sitting in ui/specs/<class>/<spec>/ multiply out to a few hundred runs, and nobody
// holds a tab open for that. So this runs headless, off the same files, and the site reads
// what it produced.
//
// It lives as a test rather than a command for one reason: the talents, spec options and
// consumes that make a build are defined in each spec's own _test.go, and a normal package
// cannot see them. Duplicating twenty struct literals into a cmd/ tool would mean the arena
// could silently drift from what the regression tests actually run, which is the one thing
// that would make its numbers worthless. Each spec contributes ten lines next to the
// definitions themselves instead.
//
// Every build meets the same fixed environment - one buff set, one consumable list for its
// role, one encounter - because that, not a shared raid, is what makes two numbers comparable.
// The environment lives in consumables.go rather than in each spec's own test file; it used
// to come from the fixtures, and fifteen specs brought four different sets from four different
// content phases, which quietly made half the leaderboard incomparable.
//
// Ported from the old Classic engine (branch classic-legacy) to this one when master switched.
// The design is unchanged; the paths, protos and the environment's field names are this
// engine's, and the talent search gained the two-tree split shapes.
package arenalib

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Where each spec drops its results. The merge step in tools/arena reads the directory;
// separate files per spec because `go test` runs packages concurrently and a shared file
// would be a race.
const outDirEnv = "ARENA_OUT"

// Searching the talent trees costs roughly a sim run per talent per step, so it is off by
// default and belongs to the long job on the arena host rather than the CI rebuild.
const optimiseEnv = "ARENA_OPTIMISE"

// Sim runs (at searchIterations) the talent climbs may spend per spec, shared between every
// start and every shape. The wall clock, not the answer, is what this protects: a climb stops
// on its own when no single point move helps, and most do long before their share runs out.
//
// Sized from a timed sample on the 16 thread arena host (2026-09-24), searching one spec at
// a time. At searchIterations the host prices about 7 builds a second for a melee spec
// (warrior, rogue), 5 for the hunter and its pet, and 21 for the mage. A spec has ~22 climbs
// (presets and the exhaustive best as starts, plus 18 shapes); each can overrun its share by
// one step, ~50 runs, so a spec spends about budget + 1,100 runs, plus a minute for the
// exhaustive pass. At 5,000 that is ~15 minutes a melee spec, ~21 for the hunter and ~6 for
// a caster: about three hours for all fifteen, with room for the estimate to be a third out
// before it passes four. Most shapes start from a blindly reshaped build and do not converge
// inside their ~230 runs, so this, not maxSteps, is what bounds them.
const optimiseBudget = 5000

// Overrides optimiseBudget, for a quick search that proves the plumbing rather than finds
// anything.
const budgetEnv = "ARENA_BUDGET"

// Writes each spec's DPS-relevant talents and stops, without running the arena at all.
// A survey rather than a search: one probe per talent, a couple of minutes a spec, and the
// answer to "how much of the tree could an exhaustive pass actually have to visit".
const relevanceEnv = "ARENA_RELEVANCE"

// Enough that two builds a few DPS apart are distinguishable, and the whole arena still
// finishes inside a CI job. A single-player run is far cheaper than the rankings page's
// raid, so this buys more precision than that page does at a fraction of the cost.
const iterations = int32(5000)

// One row of the leaderboard: a build, what it did, and what its damage was made of.
// tools/arena's rawResult mirrors this.
type Result struct {
	Spec     string `json:"spec"`
	Talents  string `json:"talents"`
	Build    string `json:"build"`
	Gear     string `json:"gear"`
	Rotation string `json:"rotation"`
	// Which of the arena's consumable lists this build drank. Published, because equalising
	// the environment and saying so are two different things and the page needs both.
	Consumables string  `json:"consumables"`
	Dps         float64 `json:"dps"`
	// Damage by spell id, for the confidence column. Weighting happens in the merge step,
	// where the manifest is read once rather than once per spec.
	Damage map[string]float64 `json:"damage"`
	// Damage from white swings, which have no spell id and no manifest entry to have.
	WeaponDamage float64 `json:"weaponDamage"`
	// Found by searching the talent trees rather than written down by a person.
	Optimised bool `json:"optimised,omitempty"`
	// Why the sim produced nothing, for the log. Not published: the merge drops a zero row.
	err string
}

// What a spec needs to contribute. Everything here already exists in the spec's test file.
type Spec struct {
	// The key the site looks results up by: the old engine's ui/ directory name.
	Dir string
	// Where the spec's presets live, under ui/specs: "warrior/dps".
	UI    string
	Class proto.Class
	Race  proto.Race

	SpecOptions interface{}
	// Which of the arena's three environments this spec gets. NOT the spec's own test
	// fixture: see consumables.go for why that had to stop.
	Role Role
	// Imbues the class grants itself - a shaman's Windfury Weapon, a rogue's poisons. These
	// override the role list, because they are not things a character buys.
	ClassImbues ClassImbues

	IsTank             bool
	DistanceFromTarget float64

	// Two specs share one preset folder (both priests are ui/specs/priest/dps). These narrow
	// it to the spec's own: talent presets whose name contains Talents, and only the named
	// gear sets and rotations. Empty means everything in the folder.
	Talents   string
	GearSets  []string
	Rotations []string

	// Rotations that differ by class options rather than by APL: a warlock who sacrifices a demon
	// before the pull runs the same priority list with another demon out. Each is ranked as a
	// rotation of its own, under its key, so the arena keeps it only for the builds it wins.
	Variants map[string]Variant
}

// A rotation file run with other spec options.
type Variant struct {
	Rotation    string
	SpecOptions interface{}
}

type TalentBuild struct {
	Name    string
	Talents string
}

// Run enumerates the spec's builds and writes them out when ARENA_OUT is set. Without it an
// ordinary `go test ./...` only runs each build for a few iterations to check the spell
// manifest (checkManifest), which costs about a second a spec.
func Run(t *testing.T, spec Spec) {
	outDir := os.Getenv(outDirEnv)

	uiDir := specDir(spec)
	talents := communityTalents(t, uiDir, spec.Talents)
	// A build the search found on some previous run is carried forward and re-simulated
	// rather than inherited. Without this a rebuild that does not search would quietly drop
	// every optimised build the long one found; with it, the number is always produced by the
	// sim as it stands today even when the search has not run since.
	carried := previouslyOptimised(spec)
	talents = append(talents, carried...)
	talents = append(talents, outsideBuilds(spec)...)
	wasOptimised := map[string]bool{}
	for _, build := range carried {
		wasOptimised[build.Name+"|"+build.Talents] = true
	}
	gearSets := filter(namesIn(filepath.Join(uiDir, "gear_sets"), ".gear.json"), spec.GearSets)
	rotations := filter(namesIn(filepath.Join(uiDir, "apls"), ".apl.json"), spec.Rotations)
	for _, name := range slices.Sorted(maps.Keys(spec.Variants)) {
		rotations = append(rotations, name)
	}
	if rotationAuditOn() && len(gearSets) > 0 && len(rotations) > 0 {
		auditRotations(t, spec, talents, gearSets, rotations)
		return
	}
	if outDir != "" && os.Getenv(relevanceEnv) != "" {
		surveyRelevance(t, spec, outDir, talents, gearSets, rotations)
		return
	}
	if len(talents) == 0 || len(gearSets) == 0 {
		t.Fatalf("%s: %d talent builds and %d gear sets, so there is nothing to rank", spec.Dir, len(talents), len(gearSets))
	}
	// A spec with no APL on disk still runs; the sim falls back to its own default rotation.
	if len(rotations) == 0 {
		rotations = []string{""}
	}

	combos := []run{}
	for _, talent := range talents {
		for _, gear := range gearSets {
			for _, rotation := range rotations {
				combos = append(combos, run{talent, gear, rotation})
			}
		}
	}
	if outDir == "" {
		checkManifest(t, spec, combos)
		return
	}
	sims := &memo{spec: spec, done: map[string]Result{}}
	results := sims.all(combos)
	for i := range results {
		results[i].Optimised = wasOptimised[results[i].Build+"|"+results[i].Talents]
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Dps > results[j].Dps })

	if os.Getenv(optimiseEnv) != "" {
		results = search(t, spec, sims, results)
	}

	for _, result := range results {
		if result.err != "" {
			t.Logf("%s: %s / %s / %s produced nothing: %s", spec.Dir, result.Build, result.Gear, result.Rotation, result.err)
		}
	}
	write(t, filepath.Join(outDir, spec.Dir+".json"), results)
	t.Logf("%s: %d builds, best %.1f dps", spec.Dir, len(results), results[0].Dps)
}

// The talent search. Starts from the best build found by the plain arena rather than an
// arbitrary one: a climb goes to the nearest peak, so where it starts is most of what it finds.
//
// On launch gear specifically, even when the spec has a better set on file. The leaderboard
// compares specs on launch gear because that is the one tier all of them have, and a build
// optimised against later gear would never appear in it - the one view most people read would
// be the one view missing the searched builds.
func search(t *testing.T, spec Spec, sims *memo, results []Result) []Result {
	started := time.Now()
	top := results[0]
	for _, result := range results {
		if strings.Contains(result.Gear, "launch") {
			top = result
			break
		}
	}

	// Climbed from every distinct build the spec has, not only its best one. A climb goes to
	// the nearest peak, so one start reports the nearest peak to one build and calls it the
	// answer; several starts that agree is the cheapest evidence available that the peak is
	// not merely nearby. They share the budget rather than multiplying it.
	anchors, err := anchorsFor(spec.Class)
	if err != nil {
		t.Fatalf("%s: %s", spec.Dir, err)
	}
	shapes := map[string]bool{}
	for _, shape := range anchors {
		shapes[shape.label] = true
	}

	// A carried shape row is not a start: an unconstrained climb from it would come back under
	// the shape's name without the shape. It is where that shape's own climb resumes instead.
	starts := []TalentBuild{}
	resume := map[string]TalentBuild{}
	seen := map[string]bool{}
	for _, result := range results {
		if result.Optimised && shapes[result.Build] && result.Gear == top.Gear && result.Rotation == top.Rotation {
			resume[result.Build] = TalentBuild{Name: result.Build, Talents: result.Talents}
		}
		if result.Gear == top.Gear && result.Rotation == top.Rotation && !seen[result.Talents] && !shapes[result.Build] {
			seen[result.Talents] = true
			starts = append(starts, TalentBuild{Name: result.Build, Talents: result.Talents})
		}
	}

	// The shapes anyone choosing a build actually compares: 31 and 21 points committed to each
	// tree, and every two-tree split. An unconstrained climb answers "what is best" and says
	// nothing about "what if I go deep in the other tree", which is the question a player is
	// usually asking. (anchorsFor, above.)

	// Found builds by name, in the order they were found. Names, not talents, are the key:
	// every shape keeps its own row even when two shapes land on the same build, so every
	// spec shows every combination.
	type find struct{ name, talents, rotation string }
	found := []find{}
	totalRuns := atomic.Int64{}

	// Before climbing from anywhere, look everywhere worth looking. Only 15 to 28 talents per
	// spec can move the number, and restricting to builds that max a talent or leave it alone
	// puts the whole space in reach. What comes back is both a row of its own and the best
	// possible place for a climb to start.
	if best, considered, simulated, err := searchEverything(spec, top, starts[0]); err != nil {
		t.Fatalf("%s: %s", spec.Dir, err)
	} else if best.Talents != "" {
		totalRuns.Add(int64(simulated))
		found = append(found, find{best.Name, best.Talents, top.Rotation})
		starts = append(starts, best)
		t.Logf("%s: considered %d combinations, simulated %d, best %s (%s)", spec.Dir, considered, simulated, best.Talents, time.Since(started).Round(time.Second))
	} else {
		// Said out loud. Seven specs once came back from a four hour run without this row and
		// nothing in the log said why, because only the success path logged anything.
		t.Logf("%s: exhaustive search produced nothing - considered %d combinations, %d runs", spec.Dir, considered, simulated)
	}

	// Every climb is independent, so they run side by side; each prices its own moves in
	// parallel too, and the scheduler sorts out the cores.
	budget := optimiseBudget
	if n, err := strconv.Atoi(os.Getenv(budgetEnv)); err == nil && n > 0 {
		budget = n
	}
	climbs := len(starts) + len(anchors)
	share := max(1, budget/climbs)
	// Every rotation the spec ran on this gear, the top one first, for each climb to choose from.
	rotations := []string{top.Rotation}
	for _, result := range results {
		if result.Gear == top.Gear && !slices.Contains(rotations, result.Rotation) {
			rotations = append(rotations, result.Rotation)
		}
	}
	// A shape plays its deepest tree's rotation: the one that tree's best build on file uses. A
	// shape starts from a blind reshape of some other build, so letting its start pick would hand
	// a 31 Fire shape the Frost rotation whenever the spec's best build is Frost.
	trees, err := loadTrees(spec.Class)
	if err != nil {
		t.Fatalf("%s: %s", spec.Dir, err)
	}
	treeRotation := map[int]string{}
	for _, result := range results {
		if result.Gear != top.Gear {
			continue
		}
		if tree := mainTree(parseTalents(trees, result.Talents)); treeRotation[tree] == "" {
			treeRotation[tree] = result.Rotation
		}
	}
	shapeRotations := func(shape *anchor) []string {
		deepest := shape.holds[0]
		for _, h := range shape.holds {
			if h.points > deepest.points {
				deepest = h
			}
		}
		if rotation, ok := treeRotation[deepest.tree]; ok {
			return []string{rotation}
		}
		return rotations
	}
	type climbed struct {
		build    TalentBuild
		rotation string
		runs     int
		err      error
	}
	out := parallelMap(climbs, func(k int) climbed {
		var build TalentBuild
		var rotation string
		var runs int
		var err error
		if k < len(starts) {
			build, rotation, runs, err = optimise(spec, starts[k], top.Gear, rotations, share)
		} else {
			// From where this shape's climb ended last run when there is one. A sim change moves
			// a peak a point or two, not across the tree, so the climb starts next to the answer
			// and stops after a step instead of spending its share walking back from a blind
			// reshape of the best build.
			shape := &anchors[k-len(starts)]
			from, ok := resume[shape.label]
			if !ok {
				from = starts[0]
			}
			build, rotation, runs, err = optimiseAnchored(spec, from, top.Gear, shapeRotations(shape), share, shape)
		}
		totalRuns.Add(int64(runs))
		return climbed{build, rotation, runs, err}
	})

	taken := map[string]bool{}
	for _, f := range found {
		taken[f.talents] = true
	}
	for k, c := range out {
		if k < len(starts) {
			if c.err != nil {
				t.Fatalf("%s: %s", spec.Dir, c.err)
			}
			t.Logf("%s: climb from %s reached %s (%s) in %d runs", spec.Dir, starts[k].Name, c.build.Talents, c.rotation, c.runs)
			// First name wins among the free climbs. The enumeration labels its result "best of
			// every combination", which is a much stronger statement than "found by climbing" -
			// and when both land on the same build, the stronger label is the true one.
			if !taken[c.build.Talents] {
				taken[c.build.Talents] = true
				found = append(found, find{c.build.Name, c.build.Talents, c.rotation})
			}
			continue
		}
		shape := anchors[k-len(starts)]
		if errors.Is(c.err, errUnreachable) {
			t.Logf("%s: shape %s skipped, %s", spec.Dir, shape.label, c.err)
			continue
		}
		if c.err != nil {
			t.Fatalf("%s: %s: %s", spec.Dir, shape.label, c.err)
		}
		t.Logf("%s: shape %s reached %s (%s) in %d runs", spec.Dir, shape.label, c.build.Talents, c.rotation, c.runs)
		found = append(found, find{shape.label, c.build.Talents, c.rotation})
	}

	// A name found this run supersedes the carried row of the same name: the shape's best is
	// whatever the search says today, not what it said last time.
	fresh := map[string]bool{}
	for _, f := range found {
		fresh[f.name] = true
	}
	results = slices.DeleteFunc(results, func(r Result) bool { return r.Optimised && fresh[r.Build] })

	// A climb that went nowhere, or the enumeration landing on a written build, adds nothing
	// the table does not already say. A shape always gets its row.
	listed := map[string]bool{}
	for _, r := range results {
		listed[r.Talents] = true
	}
	rows := []run{}
	for _, f := range found {
		if !shapes[f.name] && listed[f.talents] {
			continue
		}
		rows = append(rows, run{TalentBuild{Name: f.name, Talents: f.talents}, top.Gear, f.rotation})
	}
	for _, row := range sims.all(rows) {
		row.Optimised = true
		results = append(results, row)
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Dps > results[j].Dps })
	t.Logf("%s: %d runs (budget %d, %d a climb) from %d starts and %d shapes added %d builds in %s, best now %.1f dps",
		spec.Dir, totalRuns.Load(), budget, share, len(starts), len(anchors), len(rows), time.Since(started).Round(time.Second), results[0].Dps)
	return results
}

type run struct {
	talent   TalentBuild
	gear     string
	rotation string
}

// Publication runs by talents, gear and rotation. The same talents turn up under several names
// - two shapes landing on one build, a carried row and its fresh twin - and the second is the
// first's number relabelled: same build, same seed, same result.
type memo struct {
	spec Spec
	mu   sync.Mutex
	done map[string]Result
}

func (m *memo) all(runs []run) []Result {
	key := func(r run) string { return r.talent.Talents + "|" + r.gear + "|" + r.rotation }
	todo := []run{}
	queued := map[string]bool{}
	m.mu.Lock()
	for _, r := range runs {
		if _, ok := m.done[key(r)]; !ok && !queued[key(r)] {
			queued[key(r)] = true
			todo = append(todo, r)
		}
	}
	m.mu.Unlock()

	fresh := parallelMap(len(todo), func(i int) Result {
		return runAt(m.spec, todo[i].talent, todo[i].gear, todo[i].rotation, iterations)
	})

	m.mu.Lock()
	defer m.mu.Unlock()
	for i, r := range todo {
		m.done[key(r)] = fresh[i]
	}
	out := make([]Result, len(runs))
	for i, r := range runs {
		out[i] = m.done[key(r)]
		out[i].Build = r.talent.Name
		// Every row owns its breakdown; the merge only reads it, but nothing should have to
		// know that two rows share a map.
		out[i].Damage = maps.Clone(out[i].Damage)
	}
	return out
}

func runAt(spec Spec, talent TalentBuild, gear string, rotation string, iterations int32) (row Result) {
	environment := consumesFor(spec.Role, spec.ClassImbues)
	row = Result{
		Spec:        spec.Dir,
		Consumables: environment.Label,
		Talents:     talent.Talents,
		Build:       talent.Name,
		Gear:        gear,
		Rotation:    rotation,
		Damage:      map[string]float64{},
	}
	// Not IsTest: that turns on the test suites' labelled random streams, which cost a lot and
	// buy nothing here, and it lets a panic through - one odd build the search probes would
	// take a three hour run down with it. A build that panics is a row with no damage, which
	// the log names and the merge drops.
	defer func() {
		if r := recover(); r != nil {
			row.Dps, row.err = 0, fmt.Sprintf("panic: %.200v", r)
		}
	}()

	raid := raidFor(spec, talent, gear, rotation, environment)

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:      raid,
		Encounter: core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{
			Iterations: iterations,
			RandomSeed: 101,
		},
	})
	if result.Error != nil {
		row.err = fmt.Sprintf("%.200s", result.Error.Message)
		return row
	}
	if result.RaidMetrics == nil || len(result.RaidMetrics.Parties) == 0 {
		row.err = "no metrics"
		return row
	}

	metrics := result.RaidMetrics.Parties[0].Players[0]
	row.Dps = metrics.Dps.Avg
	collect(&row, metrics)
	for _, pet := range metrics.Pets {
		collect(&row, pet)
	}
	return row
}

// The one-player raid a build is simulated in.
func raidFor(spec Spec, talent TalentBuild, gear string, rotation string, environment core.BuffsCombo) *proto.Raid {
	uiDir := specDir(spec)
	specOptions := spec.SpecOptions
	if variant, ok := spec.Variants[rotation]; ok {
		rotation, specOptions = variant.Rotation, variant.SpecOptions
	}
	gearCombo := core.GetGearSet(filepath.Join(uiDir, "gear_sets"), gear)
	rotationProto := &proto.APLRotation{}
	if rotation != "" {
		rotationProto = core.GetAplRotation(filepath.Join(uiDir, "apls"), rotation).Rotation
	}

	distance := spec.DistanceFromTarget
	if distance == 0 {
		distance = 5
	}

	player := core.WithSpec(&proto.Player{
		Class:              spec.Class,
		Race:               spec.Race,
		Equipment:          gearCombo.GearSet,
		Consumables:        environment.Consumables,
		Buffs:              environment.Player,
		TalentsString:      talent.Talents,
		Profession1:        proto.Profession_Engineering,
		Rotation:           rotationProto,
		DistanceFromTarget: distance,
		ReactionTimeMs:     150,
		ChannelClipDelayMs: 50,
	}, specOptions)

	raid := core.SinglePlayerRaidProto(player, environment.Party, environment.Raid, environment.Debuffs)
	if spec.IsTank {
		raid.Tanks = append(raid.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})
	}
	return raid
}

// Without ARENA_OUT the arena still runs every build, briefly, to hold the evidence manifest
// (ui/sim/spells) to what the sim actually does: a spell id dealing damage with no entry there
// counts as unclassified on the leaderboard and the rankings page. This is what
// sim/spell_sources_test.go did before the engine switch, narrowed to the ids that carry damage,
// which the old syntax-tree walk could not follow into the client spell store anyway.
const manifestIterations = int32(3)

func checkManifest(t *testing.T, spec Spec, combos []run) {
	manifest := map[string]bool{}
	files, err := filepath.Glob(filepath.Join(repoRoot(), "ui", "sim", "spells", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no evidence manifest under ui/sim/spells: %v", err)
	}
	for _, file := range files {
		var entries map[string]json.RawMessage
		data, err := os.ReadFile(file)
		if err == nil {
			err = json.Unmarshal(data, &entries)
		}
		if err != nil {
			t.Fatalf("%s: %s", file, err)
		}
		for id := range entries {
			manifest[id] = true
		}
	}

	rows := parallelMap(len(combos), func(i int) Result {
		return runAt(spec, combos[i].talent, combos[i].gear, combos[i].rotation, manifestIterations)
	})
	missing := map[string]string{}
	for i, row := range rows {
		for id := range row.Damage {
			if !manifest[id] {
				missing[id] = combos[i].talent.Name
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(missing)) {
		t.Errorf("%s: spell %s deals damage (build %q) but has no entry in ui/sim/spells", spec.Dir, id, missing[id])
	}
}

// Damage per spell, summed over targets. A pet's damage is the build's damage, and the
// manifest classifies pet abilities the same as any other.
func collect(row *Result, unit *proto.UnitMetrics) {
	for _, action := range unit.Actions {
		damage := 0.0
		for _, target := range action.Targets {
			damage += target.Damage
		}
		if damage <= 0 {
			continue
		}
		if spellId := action.Id.GetSpellId(); spellId != 0 {
			row.Damage[fmt.Sprint(spellId)] += damage
		} else if action.Id.GetOtherId() != proto.OtherAction_OtherActionNone {
			row.WeaponDamage += damage
		} else {
			// An item with no spell behind it. Counted as unclassified rather than as weapon
			// damage, because it is an effect somebody has to have got right.
			row.Damage["0"] += damage
		}
	}
}

// The repository root, found from wherever `go test` put us: spec packages sit two or three
// levels down depending on the class, and guessing wrong should not produce an empty spec.
var repoRoot = sync.OnceValue(func() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("arenalib: no go.mod above " + dir)
		}
		dir = parent
	}
})

func specDir(spec Spec) string {
	return filepath.Join(repoRoot(), "ui", "specs", filepath.FromSlash(spec.UI))
}

// The community builds the site already shows, read straight out of the UI preset that
// defines them, so the arena and the talent picker can never disagree about what a build is.
//
// Read with a regex rather than by running the TypeScript: these are literals of a fixed
// shape, and the alternative is a Node step in the middle of a Go test. A spec that yields
// nothing fails the test rather than quietly ranking one build.
var talentPreset = regexp.MustCompile(`makePresetTalents\(\s*'([^']+)'\s*,\s*SavedTalents\.create\(\{\s*talentsString:\s*'([^']+)'`)

func communityTalents(t *testing.T, uiDir string, named string) []TalentBuild {
	source, err := os.ReadFile(filepath.Join(uiDir, "presets.ts"))
	if err != nil {
		t.Fatalf("no presets to read builds from: %s", err)
	}

	// The same rule the raid page uses to tell a community build from a leftover preset.
	community := regexp.MustCompile(`\d+/\d+/\d+$`)
	all := []TalentBuild{}
	builds := []TalentBuild{}
	for _, match := range talentPreset.FindAllStringSubmatch(string(source), -1) {
		if !strings.Contains(match[1], named) {
			continue
		}
		all = append(all, TalentBuild{Name: match[1], Talents: match[2]})
		if community.MatchString(match[1]) {
			builds = append(builds, all[len(all)-1])
		}
	}
	if len(builds) == 0 && len(all) > 0 {
		// Same fallback as the raid page: a spec whose presets are not named in the x/y/z
		// style still gets its first build ranked rather than disappearing from the table.
		builds = all[:1]
	}
	return builds
}

func namesIn(dir string, suffix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := []string{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), suffix)
		// Several specs ship an empty gear set or rotation for the picker to start from.
		// Simulating a naked character produces a number, which is the problem: it would sit
		// at the bottom of a leaderboard looking like a finding about the build.
		if name == "blank" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func filter(names []string, only []string) []string {
	if len(only) == 0 {
		return names
	}
	return slices.DeleteFunc(names, func(name string) bool { return !slices.Contains(only, name) })
}

func write(t *testing.T, path string, results any) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(results, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

// The optimised builds a previous run committed, read back out of the published file.
//
// Deliberately talents only. Carrying the DPS forward would leave a number describing a sim
// that no longer exists sitting in a table of numbers that do, and there is no way to tell
// them apart by looking. One per name, so every shape's row survives a rebuild that does not
// search, even where two shapes share a build.
// Builds other sites publish for this spec (outside_builds.json, each with its source), run
// beside ours on the same gear, rotations and environment and used as climb starts. They are the
// cheapest check there is on the search: when one beats every row here, the search missed it.
func outsideBuilds(spec Spec) []TalentBuild {
	data, err := os.ReadFile(filepath.Join(repoRoot(), "sim", "arenalib", "outside_builds.json"))
	if err != nil {
		return nil
	}
	var bySpec map[string][]struct {
		Name    string `json:"name"`
		Talents string `json:"talents"`
	}
	if json.Unmarshal(data, &bySpec) != nil {
		return nil
	}
	trees, err := loadTrees(spec.Class)
	if err != nil {
		return nil
	}
	builds := []TalentBuild{}
	for _, build := range bySpec[spec.Dir] {
		if parseTalents(trees, build.Talents).valid(trees) {
			builds = append(builds, TalentBuild{Name: build.Name, Talents: build.Talents})
		}
	}
	return builds
}

func previouslyOptimised(spec Spec) []TalentBuild {
	data, err := os.ReadFile(filepath.Join(repoRoot(), "ui", "app", "arena", "results.json"))
	if err != nil {
		return nil
	}
	var published struct {
		Builds []struct {
			Spec      string `json:"spec"`
			Build     string `json:"build"`
			Talents   string `json:"talents"`
			Optimised bool   `json:"optimised"`
		} `json:"builds"`
	}
	if json.Unmarshal(data, &published) != nil {
		return nil
	}

	// A patch that removes or moves talents turns last run's builds illegal: 70009 took two
	// paladin talents out, and a carried paladin build killed both paladin searches when a
	// climb refused to start from it. An illegal build is not a result, so it is not carried.
	trees, err := loadTrees(spec.Class)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	builds := []TalentBuild{}
	for _, build := range published.Builds {
		key := build.Build + "|" + build.Talents
		if build.Spec == spec.Dir && build.Optimised && !seen[key] && parseTalents(trees, build.Talents).valid(trees) {
			seen[key] = true
			builds = append(builds, TalentBuild{Name: build.Build, Talents: build.Talents})
		}
	}
	return builds
}

// The shapes the search holds, named by what they commit: "31 Fury" and "21 Fury" for each
// tree, and "20 Arms / 31 Fury" for every pair of trees and every split of 51 into a 20ish
// and a 30ish, with the third tree empty.
func anchorsFor(class proto.Class) ([]anchor, error) {
	trees, err := loadTrees(class)
	if err != nil {
		return nil, err
	}
	anchors := []anchor{}
	for i, tree := range trees {
		for _, points := range []int{31, 21} {
			anchors = append(anchors, anchor{holds: []hold{{i, points}}, label: fmt.Sprintf("%d %s", points, tree.Name)})
		}
	}
	for i := range trees {
		for j := i + 1; j < len(trees); j++ {
			for _, split := range [][2]int{{20, 31}, {21, 30}, {30, 21}, {31, 20}} {
				holds := []hold{{i, split[0]}, {j, split[1]}}
				for k := range trees {
					if k != i && k != j {
						holds = append(holds, hold{k, 0})
					}
				}
				anchors = append(anchors, anchor{
					holds: holds,
					label: fmt.Sprintf("%d %s / %d %s", split[0], trees[i].Name, split[1], trees[j].Name),
				})
			}
		}
	}
	return anchors, nil
}

// What the exhaustive question actually needs answering first: which talents can move this
// spec's damage at all.
//
// Points in a talent that changes nothing are interchangeable - every legal way of dumping
// them gives the same number - so they are not choices, they are filler. Only the talents
// that do something multiply the space, and counting them is what says whether visiting all
// of it is a job or a fantasy.
func surveyRelevance(t *testing.T, spec Spec, outDir string, talents []TalentBuild, gearSets []string, rotations []string) {
	if len(talents) == 0 || len(gearSets) == 0 {
		t.Fatalf("%s: nothing to survey", spec.Dir)
	}
	rotation := ""
	if len(rotations) > 0 {
		rotation = rotations[0]
	}
	gear := gearSets[0]
	for _, name := range gearSets {
		if strings.Contains(name, "launch") {
			gear = name
			break
		}
	}

	trees, err := loadTrees(spec.Class)
	if err != nil {
		t.Fatal(err)
	}
	base := parseTalents(trees, talents[0].Talents)

	runs := atomic.Int64{}
	measure := func(points allocation) float64 {
		runs.Add(1)
		return runAt(spec, TalentBuild{Talents: points.String()}, gear, rotation, searchIterations).Dps
	}

	relevant, _ := relevantTalents(trees, base, measure, measure(base))

	type survey struct {
		Spec  string   `json:"spec"`
		Class string   `json:"class"`
		Trees []string `json:"trees"`
		// Relevant talents per tree, by their index in row-major order - the same order the
		// talents string uses, so a counting script can line them up without the sim.
		Relevant map[string][]int `json:"relevant"`
		Total    int              `json:"total"`
		Runs     int              `json:"runs"`
	}

	out := survey{Spec: spec.Dir, Class: spec.Class.String(), Relevant: map[string][]int{}}
	for i, tree := range trees {
		out.Trees = append(out.Trees, tree.Name)
		indices := []int{}
		for j := range tree.Talents {
			if relevant[[2]int{i, j}] {
				indices = append(indices, j)
			}
			out.Total++
		}
		sort.Ints(indices)
		out.Relevant[fmt.Sprint(i)] = indices
	}
	out.Runs = int(runs.Load())

	write(t, filepath.Join(outDir, spec.Dir+".relevance.json"), out)
	counted := 0
	for _, indices := range out.Relevant {
		counted += len(indices)
	}
	t.Logf("%s: %d of %d talents can move the number, found in %d runs", spec.Dir, counted, out.Total, runs.Load())
}

// Enumerates every all-or-nothing build over the talents that can move this spec's damage,
// and returns the best of them.
//
// See exhaustive.go for what that does and does not claim. In short: every one of them is
// considered, the plausible ones are simulated, and the climb still runs afterwards to catch
// what the scoring underrated.
func searchEverything(spec Spec, top Result, start TalentBuild) (TalentBuild, int, int, error) {
	trees, err := loadTrees(spec.Class)
	if err != nil {
		return TalentBuild{}, 0, 0, err
	}
	base := parseTalents(trees, start.Talents)

	runs := atomic.Int64{}
	measure := func(points allocation, iterations int32) float64 {
		runs.Add(1)
		return runAt(spec, TalentBuild{Talents: points.String()}, top.Gear, top.Rotation, iterations).Dps
	}

	relevant, value := relevantTalents(trees, base, func(points allocation) float64 {
		return measure(points, searchIterations)
	}, measure(base, searchIterations))
	if len(relevant) == 0 {
		return TalentBuild{}, 0, int(runs.Load()), nil
	}

	// runs counts what the enumeration simulated too: it measures through the same closure.
	points, considered, _ := exhaustive(trees, relevant, value, measure)
	if points == nil {
		return TalentBuild{}, considered, int(runs.Load()), nil
	}
	return TalentBuild{Name: "best of every combination", Talents: points.String()}, considered, int(runs.Load()), nil
}
