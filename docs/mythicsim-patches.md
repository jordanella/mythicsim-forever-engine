# MythicSim downstream patches

MythicSim runs this engine from its fork (`sage3648/mythicsim-forever-engine`, branch
`codex/forever-frostfire-omen`). The branch is ElliotWood/Forever master, which is built on the
official wowsims/forever, plus the patches below. The first base was `442076902` (Merge
wowsims/forever master ea5412873). The current base is `8dc19a4241` (2026-09-27). It includes form-speed and actual spell cast-time Omen of Clarity proc corrections, life-drain weapon effects, Sword of Zeal, Argent Avenger, Fiery Weapon and Lifestealing enchants, Flurry Axe and Electrified Dagger, the 2026-09-27 client hotfix database, Stinging Viper and eight Classic weapon procs, Mage Scroll of Cryoblast, non-engineer explosives and SAF-T / EZ-Thro bombs, Deep Wounds weapon-only damage with outstanding bleed rollover, Raptor pet Savage Rend, Venomstrike procs, Defias Leather set effects, Barbaric Crossbow, Plaguefang and Wolfsbane weapon procs, the Stormshroud and Volcanic Armor proc chances, item effects below item level 50, the refreshed client database, Druid form Faerie Fire cost and timing, Hunter pet Lightning Breath scaling, Inspiration armor bonuses, the client hotfix databases, Hunter ranged scaling, Rogue Hack and Slash cooldown, Shaman Flametongue and Fire Nova fixes, and the merged Penance timing and cost fixes, Demonic Pact pre-pull sacrifice, Mana Tide Totem party restoration, Frost Mage talent fixes, and rank 4 Trueshot Aura. It also carries client 1.60.1.70009 and the earlier lower-rank spell, aura-cap, and consumable fixes.

Keep the set small. Each patch exists because MythicSim needs something upstream does not do
yet. Drop a patch as soon as upstream covers it; do not keep ours alongside an upstream version.

| # | Commit subject | Why MythicSim needs it |
|---|---|---|
| 1 | `cli: sim --strict rejects unknown fields and enum names` | The worker builds requests in code. Without it, a misspelt field or a race the build does not know is dropped silently and the sim runs a different character. |
| 2 | `core: a player option to disable racials` | The race comparison page sims each character with and without its racials to show what they are worth. |
| 3 | `rotation: Destruction casts Conflagrate for Shadow and Flame` | The Destruction rotation never casts Conflagrate, so Shadow and Flame's Shadow buff never applies to the Shadow Bolt filler. |
| 4 | `hunter: Aspect of the Beast` | Forever made Beast the melee aspect. Upstream models only Hawk, so a melee hunter has no aspect. |
| 5 | `rotation: a melee Survival rotation` | Upstream's Survival rotation shoots from range, so Raptor Strike, Mongoose Bite and Strider Kick never fire. MythicSim ranks melee Survival. |
| 6 | `items: Iceblade Hacker and Warblade of Caer Darrow proc from their own hand` | The two hand-written weapon procs fired off both hands, so a main-hand Iceblade Hacker added its Frost damage to every off-hand swing. |
| 7 | `data: inherited stat indices and armor` | Preserve the corrected item stats and armor. |
| 8 | `mage: implement baseline Frostfire Bolt` | Implement the baseline spell and its hybrid rotation. |

## 1. `cli: sim --strict`

- **What it does.** `wowsimcli sim --strict` loads the request with protojson `DiscardUnknown`
  off and exits non-zero with the protojson error. The flag defaults to off, which is
  upstream's behaviour.
- **Files.** `cmd/wowsimcli/cmd/basic_sim.go` (`loadRaidSimRequest`) and `basic_sim_test.go`.
- **Drop it when** upstream's CLI can reject unknown names itself. If upstream's flag has a
  different name, switch the worker to it and drop this patch.
- **Conflicts** can only happen where `simMain` loads the file.

## 2. `core: a player option to disable racials`

- **What it does.** `Player.disable_racials` (field 59, JSON `disableRacials`) skips every racial
  effect and keeps the race's base stats. Besides `applyRaceEffects`, it gates the race-only
  effects that live elsewhere: the night elf priest's Starshards (`sim/priest/priest.go`),
  Bloodthistle (`sim/core/consumes.go`) and the racial multipliers the reforge optimizer models
  (`sim/core/reforge_optimizer/model.go`). The rest of the code checks
  `Character.RacialsDisabled()`.
- **Tests.** `sim/core/racials_test.go` (`TestDisableRacials*`) and
  `sim/priest/racials_test.go`.
- **When rebasing**, grep upstream's new code for race checks outside `applyRaceEffects`
  (`\.Race ==`, `Race_Race`, `GetRace()`) and gate any new ones on `RacialsDisabled()`.
- **Field number.** The worker sends protojson, which reads the field by name, so the number
  only matters for binary protos and saved UI links. If upstream takes 59 for something else,
  move ours to the next free number and leave the name alone.
- **Drop it when** upstream has an equivalent option. Point the worker's `disableRacials`
  (`worker/cmd/refresh-forever-races`) at upstream's name first.

## Dropped: `core: Forever races from client 1.60.1.69977`

Dropped in the rebase onto `6cb2603d`. Upstream now models the Forever races itself: the Skyborne
(`RaceHighOrderSkyborne`, `RaceWindshaperSkyborne`), the Forever race/class pairings, and the
client's racials. Blood Elf, Draenei and Bloodthistle are gone. The patch's client tests were run
against upstream's version:
- Blood Fury, Berserking, Elune's Light, Touch of the Grave, Tauren Endurance, the Human Spirit and
  the weapon specializations match.
- The Skyborne match. Upstream applies Wind Blessed as one attack speed multiplier, and it takes
  the Skyborne base stat offsets (±1) from level 1 character sheets where the patch had zero.
- Eureka! is the 70009 client's 10% cost cut on every class. The patch had 69977's per-class
  figures.
- Upstream also models Gnome Expansive Mind, which the patch left out.
- The pairings are the ones MythicSim's race pages offer.

## 3. `rotation: Destruction casts Conflagrate for Shadow and Flame`

- **What it does.** `ui/specs/warlock/dps/apls/destruction.apl.json` casts Conflagrate (18932)
  after the Immolate refresh, while Immolate is up and either Immolate has under 4 seconds
  left or the warlock knows Shadow and Flame (Shadow, 1293816) and its buff is down. Without
  the talent it only Conflagrates at the end of Immolate. The rule is the one MythicSim's
  previous engine line measured (+5.9% on the 5/5 Destruction reference, neutral for 2/5
  builds). MythicSim copies this rotation into its worker presets.
- **Goldens.** `sim/warlock/TestDestruction.results` (average 410.50 to 420.67 DPS).
- **Drop it when** upstream's Destruction rotation casts Conflagrate.

## 4. `hunter: Aspect of the Beast`

- **What it does.** Registers Aspect of the Beast at its top rank (1299447): 110 melee attack
  power, exclusive with Aspect of the Hawk. With Deadly Aspects, a landed melee auto attack has
  Deadly Aspects' second effect as its chance (2% a rank) to trigger Quick Strikes (1299448), 30%
  melee haste for 12 sec. Beast states no proc chance of its own, so there is no Quick Strikes
  without the talent. Raptor Strike takes a swing's place as a special attack and does not proc it
  (Beast's proc flags are melee auto attacks, 0x4).
- **Files.** `sim/hunter/aspects.go`, the `AspectOfTheBeast` fields and spell mask in
  `sim/hunter/hunter.go`, and `sim/hunter/aspects_test.go`.
- **Tests.** `TestAspectOfTheBeastMeleeAttackPower`, `TestQuickStrikesNeedsDeadlyAspects`. No
  golden moves: the ranged rotations cast Hawk.
- **Drop it when** upstream models Beast. Check that its melee attack power and Quick Strikes
  match before dropping.

## 5. `rotation: a melee Survival rotation`

- **What it does.** `ui/specs/hunter/dps/apls/sv_melee.apl.json` is a melee rotation: Aspect of
  the Beast before the pull, then Raptor Strike queued on cooldown, Mongoose Bite whenever Expose
  Prey opens it, Summon Hawk, Strider Kick, Immolation Trap, and Wing Clip in any global where
  Raptor Strike and Strider Kick are more than half a second away. Wing Clip is a landed melee
  special, so it can proc Expose Prey (1310532's proc flags include melee specials, 0x10), and
  more Wing Clips mean more Mongoose Bites. Spells a build lacks are skipped, so the file serves
  any talents. `presets.ts` publishes it as `SurvivalMeleeRotation`, which MythicSim's
  `scripts/build-forever-gear.mjs` requires before it copies the file into the worker presets as
  `hunter_survival`.
- **Measured** on MythicSim's Beast Mastery reference character moved to 5 yards, with wowtbc.gg's
  5/10/35 build, at 10,000 iterations. Raptor Strike, Mongoose Bite and Strider Kick alone sim 380
  DPS. Adding Wing Clip takes it to 423, and Wing Clip plus Immolation Trap to 466. Explosive Trap
  sims 1 to 2% behind Immolation Trap. Upstream's `sv.apl.json`
  is untouched; it is the ranged Survival rotation.
- **Tests.** `sim/hunter/survival_melee_test.go` (`TestSurvivalMelee`, 5 yards,
  `SurvivalMeleeTalents` = wowtbc.gg's 5/10/35 with the spare point in Focused Fire) and its golden
  `TestSurvivalMelee.results` (average 398.57 DPS on the suite's weapons-only gear).
- **Drop it when** upstream ships a melee Survival rotation. Compare the two on the golden first.

## 6. `items: Iceblade Hacker and Warblade of Caer Darrow proc from their own hand`

- **What it does.** The two weapon procs in `sim/common/classic/items_store_gaps.go` ("Melee
  attacks with this weapon deal 41 / 28 Frost damage") get a
  `NewDynamicLegacyProcForWeapon(item, 0, 1)` proc manager, as every generated weapon proc has.
  Their proc masks alone named both hands' autos and specials. A main-hand Iceblade Hacker also
  procced off every off-hand swing, which was worth 12.5% of a dual-wielding melee hunter's damage.
- **Tests.** `sim/rogue/weapon_proc_hand_test.go` (`TestIcebladeHackerProcsOnlyFromItsHand`). The
  AllItems rows for the two weapons move in `TestFury`, `TestArms`, `TestProtectionWarrior` and
  `TestRetribution`: the harness equips Warblade in the off hand beside a main-hand weapon, where
  it used to proc off main-hand hits too (112 procs a fight on Arms, 31 now). No other row moves.
- **Drop it when** upstream's generator carries these procs (the file says to remove an entry
  then), or upstream scopes them to their hand. A generated `CreateWeaponCoHProcDamage` already
  does.

## Rebasing onto a newer upstream

1. Fetch ElliotWood/Forever master. Rebase the patches onto it:
   `git rebase --onto <new upstream> <old base> <fork branch>`.
2. Resolve `*.results` conflicts by taking upstream's side. During a rebase `--ours` is the
   upstream side and `--theirs` is the patch being replayed, so use
   `git checkout --ours -- <file>`. Never hand-merge a golden.
3. Regenerate the protos (the `-I=/usr/include` is libprotobuf-dev's `descriptor.proto`),
   build, and run the patches' own tests:

   ```
   protoc -I=./proto -I=/usr/include --go_opt=Mgoogle/protobuf/descriptor.proto=google.golang.org/protobuf/types/descriptorpb --go_out=./sim/core ./proto/*.proto
   go build --tags=with_db ./sim/... ./cmd/...
   go test ./cmd/wowsimcli/...
   go test --tags=with_db ./sim/core -run 'DisableRacials|Racial|Skyborne'
   go test --tags=with_db ./sim/priest -run 'Starshards|Arena'
   go test --tags=with_db ./sim/hunter -run 'Aspect|QuickStrikes|SurvivalMelee'
   go test --tags=with_db ./sim/rogue -run IcebladeHacker
   ```

4. After an inherited-stat database correction, verify offensive stat indices below 30 are unchanged, run the importer and armor regression tests, and regenerate affected result snapshots from a completed full test run. The correction of armor, health, mana, resistance and false Physical Damage changes item and mitigation cases across specs.

5. Run the full suite, `go test --tags=with_db $(go list ./sim/... | grep -v sim/web)`. Only
   `sim/warlock/TestDestruction` should fail, from patch 3's Conflagrate, and
   `sim/hunter/TestSurvivalMelee` wherever upstream moved hunter numbers (re-bless it into patch 5).
   If upstream's AllItems rows for Iceblade Hacker or Warblade of Caer Darrow move, re-bless them
   into patch 6. Re-bless it (copy
   `TestDestruction.results.tmp` over `TestDestruction.results`) and fold it into patch 3, so
   the patch carries the golden it moves. Any other failure is upstream's or the rebase's, not a
   golden to re-bless.
6. Check each patch's "drop it when" condition above, and update this file when a patch goes.

## 7. Legacy stat indices and armor

The inherited snapshot was imported before Resilience was removed from the Stat
proto. Its armor and later stats were one index too high. The importer now resolves
stat names against the current enum. The guarded migration corrects inherited
items, enchants and suffixes, preserves distinct client stat maps, restores missing
armor, and takes base/bonus armor from the pinned Forever planner. No conditional
attack power or spell effects are copied from the planner. Full generation uses
the same armor source. JSON and embedded binaries are updated together.

Validation: Python importer regressions and `go test -tags with_db ./sim/core
-run TestForeverReferenceArmor` check all 12 affected reference armor pieces and
Stoneskin Gargoyle Cape's 43 base armor plus 50 bonus armor. The one-time migration
refuses to run against an already migrated or different source snapshot.

## 8. Frostfire Bolt

Registers the three baseline Mage ranks from the client store and adds an experimental Frostfire
hybrid rotation. All ranks have a direct hit and a periodic effect. The spell counts as both Fire
and Frost for talent mods and direct-hit procs. Hot Streak, Fingers of Frost and Missile Barrage
explicitly include it. Improved Fireball reduces its cast time; Improved Frostbolt does not.

Core Frostfire handling takes the lower Fire/Frost resistance and the larger school spell-power,
hit and school multiplier values, avoiding duplicate Curse of the Elements or Elemental Precision.
This is limited to the Frostfire school so other mixed-school implementations are unchanged.

The pinned client store provides the damage, coefficients, mana, speed and periodic-crit flag.
Binary resistance follows Frostbolt and the original WoWSims SoD Frostfire Bolt, and still needs
Forever combat-log confirmation. The existing Fingers of Frost in-flight rule and Missile Barrage
proc chance remain the engine's beta assumptions. See `docs/frostfire-bolt.md` for evidence and tests.

Drop this patch once upstream implements all ranks, the relevant talent hooks and dual-school
calculations. Compare actual casts and damage before switching.

## 9. Light's Vigil refund reporting

The resource serializer merges rows sharing an ActionID. Light's Vigil used the
same ID for its cost and 75% refund, producing one net-negative row and hiding
its refund from resource-source breakdowns. Give only the refund metric tag 1.
The spell's cost, damage, refund calculation and timing remain unchanged.

Evidence: the shared Shockadin report `4161cca1-f878-498b-8aa4-8386c5f85331`.
`TestLightsVigilReportsCostAndRefundSeparately` exercises all three ranks. A
10,000-iteration replay on seed 42 produces identical DPS, threat, damage taken,
healing, time out of mana, actions and auras; the two resource rows sum to the
original net row. Costs and refunds can now be shown independently, including
mana lost to the cap.

Drop this patch when upstream gives the refund a distinct resource action ID
or exposes separate positive and negative resource totals in the result schema.

## 10. Innervate resource attribution

Self-cast Innervate recorded a synthetic 0.2 mana per use. External Innervate
recorded an estimate on top of the real mana ticks, double-counting regeneration.
Both now use one driver that attributes each tick's incremental regeneration to
Innervate and subtracts it from ordinary regeneration. Baseline regeneration gets
mana-cap space first; the bonus records only the remaining actual gain.

The reporting baseline follows other spirit effects, including additive Evocation,
form changes, current Spirit/MP5 and regen speed. Live arithmetic, tick timing and
cast decisions remain unchanged. This does not establish or change the game's
Evocation/Innervate stacking rules. Source-attributed passive regeneration does not
create mana-gain threat; the old synthetic entries incorrectly did. Only TPS values
changed in the Balance and Feral Cat golden fixtures.

Validation: core tests cover casting, noncasting, caps, dynamic stats, regen speed,
overlap, expiry and reset. Class tests exercise self/external Innervate and both
Evocation activation orders. All core, Druid and Mage suites pass with the embedded
database. Paired 10,000-iteration, seed-42, 120/300-second Balance and Arcane replays
have identical damage actions, auras, DPS and OOM time. Resource totals match after
removing the old synthetic row. The 300-second Balance build now attributes
2,965.21236 actual mana per fight to Innervate instead of 0.2.

Drop this patch when upstream reports actual Innervate gains without also counting
them as ordinary regeneration, including cap losses and passive-regen threat rules.

## 11. Holy Nova healing crit defense type

Holy Nova's triggered party heal used `OutcomeHealingCrit` without declaring a
DefenseType. Its first healing crit panicked in `CritDamageMultiplier`, failing
the simulation. Declare `core.DefenseTypeMagic` on the shared heal configuration,
which gives all six ranks the existing 1.5 base healing crit multiplier.

`TestHolyNovaHealingCritAllRanks` casts each rank with a forced healing crit and
checks positive healing, a recorded crit, and the multiplier. A 100-iteration
Docker reproduction failed before the patch and completed with 309 healing crits
afterward. Keep the engine's strict missing-defense-type guard.

Drop this patch when upstream declares the same defense type for every rank of
the triggered Holy Nova heal and the regression passes.

## 12. Unset spell field guards

A field left at its Go zero value can mean "never set" rather than a real value.
Three of them were read without a check, and each fails silently or only by
chance:

- **DefenseType.** `CritDamageMultiplier` panics on `DefenseTypeNone`, but only
  once a crit lands. A spell with a low crit chance passes every test and fails in
  a user's sim, which is how the Holy Nova heal (patch 11) shipped. The three crit
  chance functions (`PhysicalCritChance`, `SpellCritChance`, `HealingCritChance`)
  now call `requireDefenseType`, so a missing DefenseType fails on the spell's
  first crit roll. Every crit path, including the attack-table and expected-damage
  helpers, goes through one of them. The strict panic in `CritDamageMultiplier`
  stays, as patch 11 asks.
- **SnapshotAttackerMultiplier.** An expired dot resets its snapshot to 0. A custom
  `OnSnapshot` that sets `SnapshotBaseDamage` but not the multiplier leaves that 0
  in place, and every tick deals or heals nothing. `TakeSnapshot` marks the
  multiplier with NaN before the callback, panics if a base amount was set without
  a multiplier, and otherwise restores the previous value. A multiplier the
  callback computes as 0 is kept.
- **Cast.CD / Cast.SharedCD.** A Timer without a Duration was already rejected. A
  Duration without a Timer is never read, so the spell silently had no cooldown;
  `RegisterSpell` now rejects it too.

The snapshot guard found one live case: the Bloodfang 8-piece heal set only
`SnapshotBaseDamage = 50` and had no `DamageMultiplier`, so every tick healed for
50 x 0. It now snapshots through `SnapshotHeal` with a multiplier of 1.

Separately, `gen_db` decoded the pinned Forever planner's `stats` as
`map[string]float64`. 10,287 rows carry an `appearances` object and 3 a `skillBuff`
object there, so `make db` panicked in `fillPlannerArmor` on the committed
`wowhead_forever_gearplanner.txt`. The map is now decoded loosely and only `armor`
and `armorbonus` are typed, with a panic naming the item if either is ever not a
number. A full `make db DB2TOOL_FLAGS="--cdn --dbcache ..."` run then completes.
The regenerated database is not part of this patch: its armor for the 14 items
whose values move matches the planner on 13 of them (the committed database on 1),
but the effect generator also swaps Skullflame Shield's active proc from
Flamestrike to Drain Life, which needs its own look.

Validation: `sim/core/unset_field_guards_test.go` covers a damage and a healing crit
roll without a DefenseType (panics at 0% crit), the same spell with one (casts), a
snapshot that sets a base amount but no multiplier (panics), a computed 0
multiplier (allowed), and both cooldown cases. The full suite,
`go test --tags=with_db $(go list ./sim/... | grep -v sim/web)`, has the same
results with and without the patch: every class suite passes, so no registered
spell trips a guard, and only the stale `sim/common/TestRegisteredEffects` baseline
fails, as it does on the base branch.

Drop this patch when upstream checks DefenseType at the crit roll, rejects an
unset snapshot multiplier and a cooldown Duration without a Timer, and decodes the
planner stats without assuming every value is a number.

## 13. Windfury Totem procs in Cat and Bear Form

Classic's Windfury Totem enchanted the held weapon: the totem's periodic aura
(8515) cast 8514, which applied temporary enchant 1783. A shapeshifted druid does
not swing its weapon, so the Classic sim stripped `WindfuryTotem` from the party
buffs of both feral specs.

Forever changed the totem. In client 1.60.1.70009, 8515 is an area party aura
(effect 35, aura 42) with a 20% chance on melee autos and melee specials
(`ProcTypeMask` 0x14) that triggers 8516, the extra attack and its attack power.
Nothing in it names a weapon, so a cat or bear procs it like anyone else.
`feralcat.AddPartyBuffs` and `feralbear.AddPartyBuffs` no longer clear
`WindfuryTotem`. The shared driver already swings the current main-hand weapon,
which in form is the paw.

Validation: `sim/druid/feralcat/windfury_test.go` and
`sim/druid/feralbear/windfury_test.go` run full sims with a 3.5 s two-hander
equipped. With the totem they record Windfury extra attacks (tag 25584) at a
plausible rate per auto, and each extra attack hits for about what the form's own
auto does plus the proc's attack power, not like a swing of the two-hander.
Without the totem they record none. Both fail on the unpatched source. The Cat and
Bear suite goldens are regenerated because the suite's party buffs include the
totem; no other class's results move.

Drop this patch when upstream stops stripping Windfury Totem from the feral specs.

## 14. Forever Shadow crits: Devouring Plague ticks and Shadow Word: Death

Two Shadow crit behaviours on the live Forever server are not in beta client
1.60.1.70009, which the engine otherwise follows. Both were reported by players
running the Shadow benchmark rather than read from client data:

- **Devouring Plague ticks crit.** Every rank's row leaves Periodic Can Crit
  (`ATTR_EX_8_PERIODIC_CAN_CRIT`) off, so `priestTickOutcome(rank.PeriodicCanCrit())`
  never rolled. The benchmark report this fork produced shows 0 crit ticks in 72,232.
  The ticks now always roll for a crit; the heal still equals the damage dealt.
- **Shadowform's +100% critical strike damage bonus covers Shadow Word: Death.**
  The client's mask (41984016) names Mind Blast, Mind Flay, Shadow Word: Pain,
  Devouring Plague and Mana Burn. `PriestSpellShadowWordDeath` is added to the
  modifier's class mask, so Shadow Word: Death crits for 2.0 in Shadowform like the
  rest instead of 1.5.

The spell notes in `ui/sim/spells/priest.json` say the same. Measured on the current
tier Shadow benchmark (Undead, 120 s, 100,000 iterations): the preset rotation goes
from 581.10 to 588.66 DPS, about +2.0 from Devouring Plague and +5.6 from Shadow
Word: Death.

Validation: `sim/priest/shadow_crits_regression_test.go` forces a Devouring Plague
tick at 100% crit for every rank and checks the Shadow Word: Death and Mind Blast
crit multipliers with and without Shadowform. Both tests fail without the patch.
Only `TestShadowPriest` goldens move (re-blessed); `TestSmitePriest` is unchanged,
and the rest of the full suite matches the base branch.

Drop this patch when the client data sets Periodic Can Crit on the Devouring Plague
ranks and Shadowform's crit damage mask names Shadow Word: Death, or if a combat log
shows either behaviour is not live.

## 15. Rage log lines name the real maximum

Boundless Rage raises a Warrior's maximum rage by 10, 20 or 30 (`warrior.go`), and a
Gnome's Expansive Mind raises it 5% more (`racials.go`). Both were applied, but
`rageBar.AddRage` and `SpendRage` logged `of 100 total` whatever the bar held, unlike
energy, which logs its own maximum. MythicSim's timeline reads the maximum from those
lines, so every Warrior's rage row said "max 100". Both lines now log `rb.maxRage`.
Nothing the sim computes changes.

Validation: `sim/warrior/dps/max_rage_log_test.go` checks the log's maximum for a
Human without the talent (100), with Boundless Rage 3/3 (130) and a Gnome with it
(136.5). It fails on the unpatched source.

Drop this patch when upstream logs the rage bar's maximum.

## 16. Faerie Fire and Curse of Recklessness share their armor reduction

In client 1.60.1.70009 both take 505 armor (9907 and 11717), and Wowhead's Forever
class guides state they no longer stack. The client rows carry nothing that says so,
and the buff manifest gave each its own category, so a target with both lost 1010.
Both now sit in `MinorArmorReduction` with the new manifest option `PerStat`: each
stat the aura attaches bids alone in the category, so only the armor competes and
each keeps anything else it does. `gen_buffs.go` renders `PerStat: true` into the
`buffs.Meta`, and `newDebuff` bids with `spelldata.ExclusivePerStat` for it. The two
generated rows in `sim/core/buffs/debuffs_auto_gen.go` were edited to match what the
generator writes, since `gen_spelldata` needs the client database; the synthetic
render fixture covers the new field.

Validation: `sim/core/buffs/armor_reduction_test.go` applies each alone (-505) and
both together (-505, not -1010), and fails on the unpatched source. Every suite that
puts both on the target moves: all physical specs, pets, and one physical weapon
proc (Everlook Pathcarver) in the caster suites. Healer suites do not.

Drop this patch when the client data puts the two in one exclusive category, or if a
combat log shows them stacking on live.

## 17. A hard-cast Shaman spell holds the melee swing

Tested on the Forever beta: a Lightning Bolt cast between swings resets the swing
timer as it completes; a swing that comes due during the cast waits at zero and
lands as it completes; a bolt Maelstrom Weapon makes instant leaves the swing alone.
The engine only acted when the cast would end after the next swing, and then pushed
that swing a full swing past the cast, so a bolt that fit between swings cost no
melee at all. That made weaving 3 to 4 stack bolts between swings look free.

`AutoAttacks.HoldMeleeForCast` models the tested behaviour, and Lightning Bolt, Chain
Lightning and Lava Burst call it for any cast longer than zero. A held swing lands one
nanosecond after the cast completes: at the same instant it would wake the rotation
before the cast's completion runs, and a new cast would replace it, losing the
bolt's damage and cost.

Validation: `sim/shaman/enhancement/swing_hold_test.go` chain-casts an unhasted
2H Orc until it is out of mana, then melees, and checks from the log that no swing
lands inside a cast, that swings due during one land as it completes, that a swing
after a cast that completed first comes at least a full swing later, and that every
bolt started also completes. It fails on the unpatched source. The Enhancement suite
goldens move.

Drop this patch when upstream models a hard cast resetting the swing timer.

## 18. Windfury Totem leaves the main-hand stone or oil alone

Classic's Windfury Totem enchanted the held weapon, so `applyConsumeEffects` skipped
a main-hand stone or oil whenever the party had the totem. Forever's totem is a party
aura that procs extra attacks and names no weapon (see patch 13), so the main-hand
imbue now applies beside it. Rogue poisons were never affected: they register in
`sim/rogue/poisons.go`, not here.

Validation: `sim/warrior/dps/windfury_imbue_test.go` checks that a main-hand
Elemental Sharpening Stone adds 2% crit with and without the totem. It fails on the
unpatched source. No suite golden moves, since none pairs a main-hand stone with the
totem.

Drop this patch when upstream stops displacing the main-hand imbue under Windfury
Totem.
