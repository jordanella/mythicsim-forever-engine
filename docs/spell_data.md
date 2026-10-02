# Spell Data

The sim reads the client's numbers instead of hand-transcribed literals, out of one store:
`sim/core/spelldata`, a generated file holding every spell the sim can reach, and resolvers that turn
a row into a spell config, an aura, a dot, a talent's modifiers or a proc listener. The warrior reads
it through the resolvers; the other eight classes read its rows by hand instead, one value at a time,
with no resolver in between.

- [The store](#the-store)
- [Building an ability](#building-an-ability)
- [Auras and dots](#auras-and-dots)
- [Talents and auras: ParseStatic and ParseEffects](#talents-and-auras-parsestatic-and-parseeffects)
- [Procs](#procs)
- [Overrides](#overrides)
- [Reading a row by hand](#reading-a-row-by-hand)
- [Regenerating and checking](#regenerating-and-checking)
- [Porting a class onto the resolvers](#porting-a-class-onto-the-resolvers)
- [Traps](#traps)
- [Buffs and debuffs](#buffs-and-debuffs)

The accessor, spell config, aura and ladder snippets below are pinned by
`sim/core/spelldata/example_test.go`, which runs them against the committed store. The ones that need a
character - the parses, the proc triggers - are read off the warrior files they name.

## The store

`sim/core/spelldata/spells_auto_gen.go` holds every spell the sim can reach: the ones the class files
name, the ones items, enchants and set bonuses cast, and everything those reach in turn through a
trigger effect, an actionbar override or a tooltip reference. `sim/core/spelldata/snapshot_test.go`
pins what that comes to - 7048 rows carrying 9651 effects - so a regeneration that moves the universe
says so there.

Every field is the client's column in the client's units: a percentage is the integer 16, rage is on a
0-1000 bar, times are milliseconds. The conversion is in the accessors, so a row always matches what
the DBC says. `sim/core` must not import the package: the store imports core, and the import back
would be a cycle. That is why the raid buffs live in `sim/core/buffs` - see
[Buffs and debuffs](#buffs-and-debuffs).

### Finding a row

```go
frostbolt := spelldata.Find(116)      // Nil where the store does not carry the id
frostbolt = spelldata.MustFind(116)   // panics there instead
```

Every accessor answers on `Nil`, and `Nil` chains: `spelldata.Find(0).EffectN(1).Average(60)` is 0
rather than a crash, so a caller can ask about a spell this build does not have. `MustFind` is the
other bargain, and it is what a package-level variable takes - a regeneration that drops an id then
fails at startup, naming the id, instead of registering a spell with no numbers. `All()` and
`ByName(name)` exist for tests and debugging; a sim names its spells by id.

An id in hand-written code reads without grepping the table: `go run ./tools/spelldata 11574` prints the
row's header and one line per effect, each worded and then followed by the client's own columns. The
header names the school, the times and the cost, and also the stances by name, the weapon or slot the
row requires, the range, the target cap, the stacks and charges, the internal cooldown, the attributes
`attributes.go` exposes, the proc's rate and flags, the spells the tooltip references and the row's
labels. The title line names the ladder a class file reaches the id through - `warrior
spellData.Rend.Highest()` - scanned out of `sim/*/spell_data_auto_gen.go`, and says nothing for a class
the generator has not put on ladders yet. The same scan reads the other way too: `-family
warrior/Execute` prints a ladder's ranks with the call that reaches each, then its highest rank in full,
and `-expr 'spellData.Execute.Rank(3)' -package warrior` prints the row one ladder call names. A pick
followed by the store's own accessors is read too - `-expr
'spellData.Execute.Highest().EffectN(1).Average(core.CharacterLevel)'` answers the value, the doc comment
that accessor carries and the row with the effect it read marked `(read)`.

The worded line states each number in the unit the sim spends it in, which is also how it names the
accessor it was read with: a plain amount is `Average(60)` and a range is `Min`/`Max`, a percentage is
`Percent()`, rage is `Tenths()`, a time is `TimeValue()`, and a share of spell or attack
power is `Coeff()`/`APCoeff()`. A dummy line states the row's number and nothing about what it means,
which is the one shape that always needs the tooltip. An effect whose type or aura the printer has no
wording for says `unrecognised shape`, and the literal beneath it is then the whole answer.

A name instead of an id lists every row carrying it and then the highest rank, and
`-json` is the same answer for a tool to read. `-config 'spelldata.SpellConfig(&warrior.Unit,
executeRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))' -package warrior` prints the config the
resolver builds for that call, each field with the step that filled it - the row or the option - and
the row pick substituted through the package's declarations.

The same reading is an editor hover. `go run ./tools/spelldata -lsp` is a language server on stdio: a
hover on an id Go, APL JSON and TS state - `MustFind(11574)`, `SpellID: 11574`, `"spellId": 11574`,
`fromSpellId(23563)` - shows the row, with the header and the effects as tables; in Go a
`spellData.<Family>` token shows the ladder, a name bound to a ladder chain shows the rank or the value
it reads (substituted through the names it stands on, read from the open buffers first), each accessor
on a declaration line answers for itself, and the cursor on `spelldata.SpellConfig` shows the resolved
config. Each hover logs how it was resolved as a `window/logMessage`. `-hover sim/warrior/execute.go
12:40` prints the markdown a hover at that line and column shows, for scripts and tests. The server
compiles the store in, so a hover matches the checkout it was started in.

`.vscode/extensions/wowsims-spelldata` is its VS Code client, a workspace extension VS Code offers to
install when the repository is opened. It is built, not edited: the source is
`tools/vscode-spelldata`, and `make vscode-spelldata` bundles it with `vscode-languageclient` and writes
the manifest; `npm run check` there fails where the committed output is not a fresh build.
`tools/zed-spelldata` is the Zed extension (`zed: install dev extension`), and
`tools/spelldata/README.md` has the method table and the Neovim and Helix snippets.

### Reaching an effect

```go
damage := frostbolt.EffectN(2)                                 // the second effect the row carries
slow := frostbolt.Effect(dbcenums.A_MOD_DECREASE_SPEED, 0)     // the one effect with this aura and misc value
```

`EffectN` counts from **1, by position**. That is not the client's `EffectIndex`, which has gaps - 46
of the store's rows state an index that is not the position it sits at - and the row keeps the client's
own number in `Effect.Index`. A position the row does not have answers `NilEffect`.

`Effect(aura, misc)` names an effect by what it does rather than by where it sits. It panics when no
effect matches, and when two do: reading the first of several silently is the bug it exists to prevent,
and `EffectN` is the way past it. `FindEffect(type, aura, misc)` is the same question without the
panic, answering `NilEffect` instead.

The role readers each answer the first effect of their kind, or `NilEffect`: `DamageEffect()` (school
damage or any of the weapon-damage effects), `HealEffect()`, `EnergizeEffect()` and `PeriodicEffect()`
(the aura application whose aura ticks).

### Reading a value

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| `BaseValue()`             | the client's own number, unconverted                                  |
| `Percent()`               | over 100, because the client states a percentage as the integer 16    |
| `Tenths()`                | over 10, because rage sits on a 0-1000 bar                            |
| `TimeValue()`             | the number read as milliseconds, which is what a duration modifier is |
| `Period()`                | `EffectAuraPeriod`, the tick interval                                 |
| `Coeff()` / `APCoeff()`   | the spell power and attack power shares of the effect                 |
| `Average(level)`          | the amount at a caster level                                          |
| `Min(level)`/`Max(level)` | the ends of the roll                                                  |
| `Roll(sim, level)`        | the amount for one cast                                               |

`Average(level)` folds the amount the way the tooltip does: the base points plus
`EffectRealPointsPerLevel` for each level between the spell's own `SpellLevel` and the caster's,
stopped at `MaxLevel` where the row states one, and floored. The arithmetic is float32 on purpose -
the client's per-level gain is a float32 widened into the database, and folding it in float64 moves
six rows off the tooltip.

`EffectVariance` is the spread the server rolls the amount over: `Min` and `Max` are the average times
`1 -/+ Variance/2`. `Roll` rolls it where the row states one and answers the average where it does
not, so a call site never has to ask whether this particular spell's amount is a range.

### Attributes, edges and class flags

`Spell.Attr` is the client's 17 attribute words. `HasAttr(word, bit)` reads one - a word past the end
answers unset rather than panicking - and `sim/core/spelldata/attributes.go` names the ones the sim
acts on: `IsPassive`, `IsChanneled`, `RefundsOnMiss` with `MissRefund()` (the 0.8 a
`RageCostOptions.Refund` takes), `PeriodicCanCrit`, `CanProcFromProcs`, `ClassSpellsOnly`,
`CannotCrit`, `IsAProc`, `SuppressesWeaponProcs` and `IsWeaponProcAura`; `IsBleed` reads
`SpellCategories.Mechanic` instead.

|                     |                                                                                                        |
| ------------------- | ------------------------------------------------------------------------------------------------------ |
| `effect.Trigger()`  | the spell `EffectTriggerSpell` names                                                                   |
| `spell.Triggered()` | every spell the row's effects fire, deduped, in effect order                                           |
| `spell.Drivers()`   | the spells whose effects fire this one, and the ones whose actionbar override replaces a spell with it |
| `spell.Refs()`      | the spells the tooltip names (`$12880d`, `$12966n`), in the order it names them                        |

An id the store does not carry is left out of those lists rather than answered as `Nil`. Lightning
Shield rank 1 shows why the two kinds of edge are separate: its aura effect triggers 26545, the
dispatcher every rank shares, while the rank's own damage spell 26364 is reachable only as a tooltip
reference.

`Spell.ClassFlags` is `SpellClassSet` with `SpellClassMask_0..3`: the family a spell files under and
its bit in it. `Effect.ClassFlags` is the same for `EffectSpellClassMask`, which is the set of spells a
modifier effect reaches, and `spell.AffectedBy(effect)` asks whether this spell is one of them.
`core.SpellConfig.ClassFlags` carries the row's own, which is what puts a registered spell inside the
talents that name it.

### A class file is ladders

Every class file is a generated `spellData` global with a `spelldata.Ladder` per family and no numbers
of its own (`sim/warrior/spell_data_auto_gen.go`):

```go
var spellData = generatedSpellData{
	AngerManagement: spelldata.Ranked(12296),
	Anticipation:    spelldata.Talent(12297, 5),
	BattleShout:     spelldata.Ranked(6673, 5242, 6192, 11549, 11550, 11551, 25289),
}
```

`Ranked` is one spell per rank, lowest first. `Talent` is a trait-tree talent, which the client states
as one spell whose per-rank numbers live in a curve: each rank is that spell with the curve's value on
the effects the curve covers, and an effect it has no row for keeps the spell's own base value. Both
`MustFind` every id, so a family that loses a spell fails at startup rather than registering nothing.

|                                          |                                                                                       |
| ---------------------------------------- | ------------------------------------------------------------------------------------- |
| `Rank(n)`                                | the spell at rank n. Rank 0 is untaken and answers `Nil`, so no `if rank > 0` guard   |
| `Highest()`                              | the top rank                                                                          |
| `ByID(id)`                               | the rank with this spell id; panics on one the ladder does not carry                  |
| `Len()`, `Each(fn)`                      | how many ranks, and each of them with its number                                      |
| `ValueAt(rank)`                          | the rank's only effect in the client's units; panics where the rank has more than one |
| `FractionAt`, `MultiplierAt`, `TenthsAt` | the same over 100, as `1 +` that, and over 10                                         |
| `EffectAt(n)`, `Effect(aura, misc)`      | one named effect across the ranks, carrying the same four readers                     |

`MultiplierAt` takes its sign from the data: a talent the client states as -2/-4/-6 gives 0.94 at rank
3 and nobody writes the minus.

## Building an ability

`spelldata.SpellConfig(unit, row, opts...)` fills the `core.SpellConfig` fields the row states, and
leaves everything else to the caller:

```go
var pummelRank = spellData.Pummel.ByID(6554)

config := spelldata.SpellConfig(&warrior.Unit, pummelRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) { /* ... */ }
warrior.RegisterSpell(config)
```

What the row fills: the `ActionID`, `Rank` (from "Rank 4", which is what flips `HasRanks` in the APL
UI), `SpellSchool`, `DefenseType`, `ClassFlags`, `MissileSpeed`, `MinRange`/`MaxRange`, the cast - cast
time, the GCD where the row sits in the global cooldown category, its own cooldown and the category one
it shares - and the cost out of the first bar it states, with rage divided off the 0-1000 bar and the
refund the Discount Power On Miss attribute states. `CastRequirement` gets the forms and caster auras
the client requires (see below), which is why Pummel needs no Berserker Stance condition. `Flags` gets
what the attributes and targets say:
`SpellFlagPassiveSpell`, `SpellFlagChanneled`, `SpellFlagSuppressWeaponProcs` and `SpellFlagHelpful`.

A bleed row also fills the damage and threat multipliers with 1.

What it does not: `ApplyEffects`, `ProcMask`, the multipliers on any other row, `ClassSpellMask`, `ExtraCastCondition`,
`Dot`, `RelatedSelfBuff`, the threat numbers the client does not carry - and `MaxTargets`, which has no
`SpellConfig` field at all, so a caller that caps an area effect reads `row.MaxTargets` itself. The same
goes for `RequiredAreas`, the area group a spell only works in: `row.AreaType()` names the kind of
terrain it stands for (Forest and Grassland, Mountainous, ...) and a caller gates on
`sim.Encounter.InArea(row.AreaType())`; a group that is a single zone reads as `AreaTypeUnknown`. A unit
is needed for the cooldown timers, so a config is built where the sim has a character rather than at
package init.

|               |                                                                                                                                                                        |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Melee(mask)` | the proc mask, `SpellFlagMeleeMetrics` and `SpellFlagAPL`, damage and threat multipliers of 1, and `IgnoreHaste`                                                       |
| `Magic(mask)` | the proc mask, `SpellFlagAPL`, the multipliers, and `BonusCoefficient` from the damage effect's spell power share - the heal's where the spell damages nothing         |
| `Proc()`      | `SpellFlagPassiveSpell` and `SpellFlagNoOnCastComplete`, clears `SpellFlagAPL`, and empties the cast - cast time, GCD, cooldowns - every cost and the cast requirement |
| `Flags(f)`    | ors in flags the client does not state                                                                                                                                 |
| `Tag(n)`      | splits one spell id into several actions                                                                                                                               |

The row is filled first and the options run on top in the order given, so an option sees what the row
put there.

### Where a spell can be cast

`core.CastRequirement` is what the client requires of the caster's form and auras, and core refuses a
cast, `CanCast` and `CanQueue` that break it ("wrong form", "missing caster aura", "excluded caster
aura"). It reads four client sources: `SpellShapeshift`'s mask and exclude mask (`StanceMask`,
`StanceExclude`), `SpellAuraRestrictions` (`CasterAura`, `ExcludeCasterAura` - Shifting Power names
Cat Form there as well as in its mask), the not-shapeshifted and castable-in-caster-form attribute
bits, and `SpellShapeshiftForm`'s stance flag (`dbcenums.ShapeshiftForm.IsStance`, generated into
`sim/core/dbcenums/forms_auto_gen.go`).

For the caster's form `f`: an excluded form refuses; a form in the mask allows; in a shapeshift that is
not a stance (Cat, Bear) a spell with a mask or the not-shapeshifted bit is refused; with no form or in
a stance (the warrior's, Moonkin, Tree) a spell with a mask is refused unless it is castable in caster
form. A required caster aura must then be active on the caster, matched by `ActionID.SpellID`.

The class reports its form in `Unit.ShapeshiftForm` - the warrior's `setStance`, the druid's `setForm` -
and nothing else. A unit that sets `Unit.AutoUnshift` (the druid, to `ClearForm`) casts a spell its form
refuses but no form allows by leaving the form first, once, after every other check passed. That
holds in a stance-type form too: Healing Touch in Moonkin Form and Wrath in Tree of Life leave the form.

- A class that reads the store by hand takes `row.CastRequirement()`.
- A hand-built spell with no row states it by hand: `core.InForms(dbcenums.FORM_BATTLE_STANCE)`, with
  `.Excluding(...)` and `.OrCasterForm()`, or the struct literal for the attribute bits.
- A talent that swaps a spell on the action bar (`A_OVERRIDE_ACTIONBAR_SPELLS`) swaps its row too:
  Vanguard's Charge takes `chargeRank.OverriddenBy(spellData.Vanguard.Highest()).CastRequirement()`,
  whose row allows Defensive Stance.
- A form spell (subtext "Shapeshift") is a single-rank ladder like a warrior stance, so
  `spellData.CatForm` exists.

### The three shapes a registration takes

1. **The row, plus what no row states.** `sim/warrior/pummel.go` and `sim/warrior/slam.go`: the
   resolver, then the cast condition and `ApplyEffects`.
2. **The row, then an override of one of its fields.** `sim/warrior/hamstring.go` adds
   `ClassSpellMask` and pins the threat numbers with their review marker, because the client states no
   threat coefficient. Write the override after the call, on its own line, with the reason beside it.
3. **A hand-written `core.SpellConfig` reading the row's values.** The Sweeping Strikes hit spell in
   `sim/warrior/talents_arms.go` is a spell the sim registers and the store does not carry, so it is
   built by hand out of the parent's numbers.

### When not to take a default

- **A dot the client keeps on an `A_PERIODIC_DUMMY` stays hand-written.** `PeriodicEffect()` does not
  answer a dummy, and Deep Wounds' dummy states a spell power coefficient of 1 that `DotConfig` would
  put on a weapon-damage tick. Its spell config still resolves; only the dot is by hand
  (`sim/warrior/talents_arms.go`).
- **A sub-spell whose casts the sim reports does not take `Proc()`.** The option marks the spell
  passive and empties its cast, cooldown and cost, and the metrics aggregator counts no cast for a
  passive spell. Blood Craze's heal takes it on its own row, `BloodCrazeTriggered`
  (`sim/warrior/talents_fury.go`); Whirlwind's off-hand strike takes it on the Whirlwind row itself -
  `Melee(ProcMaskMeleeOHSpecial)`, `Proc()`, `Tag(2)` - which is what lets a spell with a cast time and
  a cooldown of its own lend its row to a sub-spell that has neither, writing only `ClassSpellMask`
  and `ApplyEffects` by hand (`sim/warrior/whirlwind.go`). Deep Wounds (`sim/warrior/talents_arms.go`)
  and Retaliation's counterattack (`sim/warrior/retaliation.go`) name the flags they need by hand
  instead - the bleed takes `SpellFlagNoOnCastComplete` with `SpellFlagIgnoreResists` and
  `SpellFlagProc`, the counterattack `SpellFlagMeleeMetrics`.
- **A sim-only copy of a spell carries the parent's `ClassFlags`.** Sweeping Strikes' hit spell and its
  normalized-attack copy (`sim/warrior/talents_arms.go`) have no row of their own, and without the
  parent's family mask the talents that name Sweeping Strikes would not reach them.

## Auras and dots

`spelldata.AuraConfig(row, opts...)` answers a `core.Aura` with the row's name as the label, its
`ActionID`, its duration - the client's -1 permanent aura becomes `core.NeverExpires` - and its stacks:
`CumulativeAura` where the row states one, `ProcCharges` otherwise, since the sim keeps stacks and
charges in one field. A row that states no duration at all resolves to 0, which core refuses to
activate: such an aura needs `Permanent()` or a duration of the caller's.

`Label(name)` renames one, which two auras of the same spell on one unit need. `Permanent()` makes the
aura last the iteration whatever the row says.

```go
aura := warrior.RegisterAura(spelldata.AuraConfig(shieldBlockRank))
spelldata.ParseEffects(&warrior.Character, aura, shieldBlockRank)
```

`spelldata.DotConfig(row, effect, opts...)` answers a `core.DotConfig`: that aura, `TickLength` from
the effect's period, `NumberOfTicks` from the row's duration over that period, `BonusCoefficient` from
the effect's spell power share, and an `OnTick` that deals the effect's own `Average` at the caster's
level, on the stats and multipliers in force at the tick - `OnSnapshot` is nil. It panics where the
effect states no period or the row no duration, rather than resolving a permanent aura's -1 into an
absurd number of ticks. A caller replaces `OnTick` for a heal or a value the client does not state.

`row.TickOutcome(dot)` picks the tick's outcome from the row: a tick that can crit where the client
marks Periodic Can Crit, on the magic hit table where the row's defense type is magic. A dot's `OnTick`
therefore never names an outcome itself.

## Talents and auras: ParseStatic and ParseEffects

A talent, a passive and a buff are all the same thing to the client: an aura whose `E_APPLY_AURA`
effects say what it does. The parse walks those effects and attaches each to the sim - a `SpellMod`
for the two modifier auras, a stat buff, a stat multiplier, a pseudo-stat multiplier or one of the
speed multipliers - so a port writes which row it is reading rather than what the row contains.

```go
spelldata.ParseStatic(&warrior.Character, spellData.Cruelty.Rank(warrior.Talents.Cruelty))
```

`ParseStatic` applies now and never comes off, which is what a talent or a class passive is.
`ParseEffects(character, aura, row, opts...)` makes the same attachments follow an aura: they turn on
when it is gained, off when it expires, and scale with its stacks where the row states
`CumulativeAura`.

A modifier effect is gated by the client's own class flags: `EffectSpellClassMask` decides which
registered spells the mod reaches, so a talent touches exactly the spells the client names it for. An
effect that names no spells at all, on a row of a class family, reaches that whole family - which is
what the client means by an unmasked class modifier.

**Call `ParseEffects` before the aura activates.** It attaches through `ApplyOnGain` and
`ApplyOnExpire`. An aura already up when the parse runs is caught up the way core's `Attach` helpers
catch one up, with one exception: the rows that need a `Simulation` to act - the cast, melee and attack
speed multipliers - stay off until the aura is applied again.

**An aura several ranks share takes a modifier once**, however many of those ranks the modifier's mask
names, because the mod is attached to the aura and not to each spell pointing at it.

|                   |                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------- |
| `Effects(1, 2)`   | only these effects, counted from 1 by position the way `EffectN` counts                                             |
| `SkipEffects(3)`  | everything but these                                                                                                |
| `Conditional(fn)` | a condition every attachment is gated on, read on gain and whenever the caller calls `Refresh(sim)`                 |
| `IgnoreStacks()`  | the values do not follow the aura's stacks, which is what a row stating charges rather than cumulative stacks means |

Defensive Stance is both shapes at once (`sim/warrior/stances.go`): the stance passive is parsed onto
the stance aura, and Defiance is parsed onto the same aura with a `Conditional` for the shield its
tooltip asks for and the row states nowhere, re-read through `Refresh` on an off-hand swap.

The call answers a `*Parsed`. `Applied` names each attachment - the effect, the sim kind it became and
the value in the sim's own units - and `Skipped` holds the aura effects the table has no row for, which
are exactly what the port still has to wire by hand. `SPELLDATA_REPORT=1` prints each of those on the
console with its position, the client's name for its aura, its misc value and its amount. Improved Slam
(`sim/warrior/talents_arms.go`) is the ordinary case: the cast time and global cooldown mods attach,
the five rank-swap effects behind them have no sim kind and are reported.

### What a value becomes

The table writes the sim's own units, and which unit that is differs per stat. The conversions:

|                                                             | the sim stores        | the parse writes                              |
| ----------------------------------------------------------- | --------------------- | --------------------------------------------- |
| Strength, Agility, Stamina, Intellect, Spirit               | flat points           | the value                                     |
| Health, Armor, the five resistances                         | flat points           | the value                                     |
| PhysicalDamage, SpellDamage and the per-school damage stats | flat power            | the value                                     |
| HealingPower, AttackPower, RangedAttackPower                | flat power            | the value                                     |
| MP5                                                         | mana per five seconds | the value                                     |
| PhysicalHitPercent, SpellHitPercent                         | percentage points     | the value                                     |
| PhysicalCritPercent, SpellCritPercent                       | percentage points     | the value                                     |
| **BlockPercent**                                            | **a fraction**        | **the value over 100**                        |
| DodgeRating                                                 | rating                | value x `DodgeRatingPerDodgePercent`          |
| ParryRating                                                 | rating                | value x `ParryRatingPerParryPercent`          |
| ExpertiseRating                                             | rating                | value x `ExpertisePerQuarterPercentReduction` |
| `PseudoStats.BonusHealingTaken`                             | flat healing          | the value                                     |
| every pseudo-stat multiplier and stat dependency            | a multiplier near 1   | `1 + value/100`                               |

Block is the one that reads differently from its name: core sums `stats.BlockPercent` with the rating
share already divided by 100, so the client's integer 5 is 0.05 there and a row handed over unconverted
is five hundred percent block. `TestParseStaticStatConventions` pins one row of each family against
what core reads back.

A talent that states the same percentage twice - once for the hit and once for the dot - has its dot
effect folded into the hit one, because a single `SpellMod_DamageDone_Flat` already reaches the ticks
and attaching both would double the bonus. The `Applied` entry says `folded-into`. A parse narrowed to
the dot effect alone attaches it as a dot mod instead.

### What the table does not take

Anything the table has no row for is skipped and reported rather than guessed at: the dummies the
client uses for "a script does this", the proc-driver auras (a proc is a `ProcTrigger`, not a
modifier), the crowd-control auras - stun, fear, root, silence - and `A_MOD_SKILL`. Those are named in
the report rather than numbered, so a port can see at a glance which ones are its own work.

Rows the table does know are skipped by the shape of the parse. A value the sim keeps as one number
per unit cannot be applied once per stack, and the static path has no aura to follow and no
`Simulation` to answer a later `Refresh` with:

|                                     | skips                                                                                                                       |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| a stacking aura (`MaxStack` > 0)    | the stat multipliers, the equipment scaling, the pseudo-stat multipliers, the speed multipliers and the cooldown multiplier |
| `ParseStatic`                       | the speed multipliers, which need the `Simulation` an aura's gain hands over                                                |
| `ParseStatic` with `Conditional`    | the stat multipliers and the equipment scaling as well                                                                      |
| an aura on a unit with no character | the equipment scaling, whose helper is a character's                                                                        |

`IgnoreStacks()` clears the first row of that table for a spell whose column counts charges rather
than cumulative stacks.

## Procs

The client's proc chance column is not always a chance, so the generator bakes what the tooltip says
about it into the row as a `ProcChanceSource`. The four shapes:

|                     |                                                                                                                                                                                                                               |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ProcChanceColumn`  | the tooltip renders `$h%`, so `SpellAuraOptions.ProcChance` is the roll                                                                                                                                                       |
| `ProcChanceEffectN` | the tooltip renders `$mN%`: the value of the effect at position `ProcChanceEffect` is the roll                                                                                                                                |
| `ProcChanceAlways`  | the column reads 100 or 101 and the trigger clause states no chance at all, so the aura fires whenever its own condition is met                                                                                               |
| `ProcChancePPM`     | the client states no chance anywhere, or a 100/101 column sits beside a trigger clause saying the effect only sometimes happens ("Chance to strike your ranged target"), so the rate has to come from an override into `RPPM` |

Reading `ProcChance` directly is the bug `ProcChanceSource` exists to prevent: 100 and 101 are the
client's "fires on its own condition" sentinel at least as often as they are a certainty.

```go
trigger := spelldata.ProcTrigger(&warrior.Character, spellData.Enrage.Rank(rank),
	func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
		warrior.EnrageAura.Activate(sim)
	})
trigger.Name = "Enrage - Trigger"
warrior.MakeProcTriggerAura(trigger)
```

`spelldata.ProcTrigger(character, row, handler, opts...)` fills the listener the row describes: the
name and action id, the `Callback`, `ProcMask`, `Outcome` and `RequireDamageDealt` the decoder reads
out of `ProcTypeMask`, the internal cooldown, `CanProcFromProcs` and `ClassSpellsOnly` from the
attributes, the class mask the proc effect names, and the rate the source above points at. A row that
is a weapon proc aura also excludes the hits of spells flagged Suppress Weapon Procs.

|                 |                                                                                                                                                             |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PPM(n)`        | a rate the client does not carry. It clears the chance and binds a manager to the trigger's proc mask, so an option that narrows the mask has to come first |
| `Chance(f)`     | a rate the caller states outright, manager included                                                                                                         |
| `ChanceFrom(e)` | the chance an effect states, for a `$mN` the generator could not resolve                                                                                    |

A trigger that ends with neither a chance nor a manager panics naming the spell. A listener that fires
on every qualifying hit is never what a row with no stated rate means, and a procs-per-minute rate with
no proc mask to measure it on panics for the same reason - on a weapon proc, which hits count is the
weapon's to say.

### The decoder and the tooltip hints

`core.DecodeProcTypeMask(row.ProcFlags, row.ProcHint)` turns the client's two proc words into a
`Callback`, a `ProcMask`, an `Outcome` and `RequireDamageDealt`, and names the bits it does not model
in `Unsupported`. `RequireDamageDealt` defaults to **true**, and a listener that fires on a dodge, a
parry, a miss or a block needs it false. Where the tooltip named that outcome the row carries
`ProcHintOutcomeTaken` and the decoder clears it already; a row whose wording yielded no hint - the
Battlegear of Wrath consume 23547 is one - has to be cleared by hand. Which outcome it is stays the
caller's either way, since no `ProcTypeMask` has a bit for any of them.

The mask states which hits reach the listener; it cannot state the condition around them, so the
generator reads that off the tooltip into `core.ProcHint`:

|                        |                                                                                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ProcHintCastTrigger`  | the tooltip names the cast itself, which turns a mask of the spell bits alone into `OnCastComplete` with no outcome                                     |
| `ProcHintCrit`         | the tooltip names a critical strike, which sets `Outcome = OutcomeCrit` and keeps the listener on the hit rather than the cast                          |
| `ProcHintHeals`        | the trigger clause names healing or an unrestricted spell, which is the evidence that a helpful-spell bit carries a real trigger rather than a leftover |
| `ProcHintPureHeal`     | the trigger is healing only, so the damage callbacks come off                                                                                           |
| `ProcHintNamedAbility` | the clause names one ability ("your Shock spells"), which no proc mask can state                                                                        |
| `ProcHintOutcomeTaken` | the clause names an outcome the mask has no bit for, which clears `RequireDamageDealt`                                                                  |

The last two are shapes the decode reads past rather than models, and they are what
`ProcTriggerUnsupported(character, row)` reports alongside the decoder's own unsupported bits. A
trigger is still built for all of them: a listener that hears fewer hits than the client's is
deliberately the narrower one.

A class mask naming **another class's** family is dropped rather than obeyed. An item every class can
wear states the filter of the one class it was written for, and on anyone else that mask matches
nothing and would silence the listener instead of narrowing it. A family the client states for no class
and the empty mask are evidence of nothing and are left alone.

### The shapes, as the warrior reads them

- **Enrage** (`sim/warrior/talents_fury.go`) is the column: its tooltip renders `$h%`, so the 30 in
  the column is a real roll on damage taken. Only the trigger's name is the caller's, because the
  driver and the buff share one.
- **Shield Specialization** (`sim/warrior/talents_protection.go`) is the effect ladder: the column
  reads 100 and the roll is effect 2. Its tooltip names an outcome, which the row carries as
  `ProcHintOutcomeTaken`, so the decoder already clears `RequireDamageDealt` - a block deals no damage.
  Which outcome it is stays the caller's: `Outcome = OutcomeBlock`.
- **Flurry** (`sim/warrior/talents_fury.go`) is the no-roll shape: the row states "always" and the
  crit is the condition. The buff row supplies the duration and the three charges through
  `AuraConfig`, the haste comes off the talent's ladder, and the mask stays the caller's because the
  row's reaches ranged and spell hits while the tooltip says melee. Where the row and the tooltip
  disagree on a number - 12966's flat 30 against the ladder's 25 at five points - the decision is
  written at the call site.
- **Lightning Shield** 324 is the same no-roll shape on a row nothing ports yet: 100 in the column,
  three charges, a 3.5 second internal cooldown, and its damage spell reachable only as a tooltip
  reference while its effect triggers the dispatcher every rank shares.

### Item and enchant procs

An item, enchant or set proc is registered from two spell ids and a shape:

```go
shared.NewSpellDataDamageProc(shared.SpellDataProc{TriggerSpellID: 7711, BuffSpellID: 7712},
	[]shared.ItemVariant{
		{ItemID: 17111, ItemName: "Blazefury Medallion"},
	})
```

`NewSpellDataProc` is the same for a proc whose buff is an aura. What the listener
hears, how often it fires and its internal cooldown come from the trigger's row; how long the buff
lasts and how it stacks come from the buff's; only the stats come from the item's effect entry, where
the sim's item level scaling lives. `IsWeaponProc` states the one shape no row describes: the game
casts a chance-on-hit effect off the weapon's hit without consulting a proc mask, and there the 100/101
sentinel is never a rate.

A buff the client states as a percentage of a stat (`A_MOD_PERCENT_STAT`, misc the stat index, -1 all
five) has no effect entry, which carries flat stats only, so the generator routes it as auras:
`NewSpellDataAuraProc` parses the buff's row, whose multiplier works through a dynamic stat dependency
and reports what it adds and removes to the temporary stats listeners. Insight 8216's 1299796 doubles
Spirit for 10 s. The aura proc registers the wearer's buff with the item swap, so an enchant's buff
drops with its weapon or shield, and, where the buff moves stats, with the stat-proc APL values. The
client has no item proc of this shape. `ParseStatic` reads the same aura as a multiplier applied once,
which reports nothing.

An enchant's procs are read one per slot of its `SpellItemEnchantment` row that casts a combat spell
(Effect 1) or hangs an equip aura off a hit (Effect 3), and at most one of them registers. A combat spell's chance, where the client states
one, is on the enchantment rather than on the spell - `EffectPointsMin`, Fiery Blaze's 15 - and the
store writes it onto the spell's row as its `ProcChance` column, noted beside the row. Every
enchantment casting one spell states the same chance for it; the generator stops on one that does
not. The chance is rolled on the hits of the enchanted weapon only
(`shared.statedWeaponProcChance`). An equip aura's own description is empty, so the store reads
the description of the spell granting the enchant in its place, for what no proc mask can state: the
named ability, outcome-taken, attack-dodged and attack-parried hints, and whether its 100 is a
sentinel ("often", "sometimes", "a chance to"), which makes it `ProcChancePPM`. The grant's cast,
crit and heal wording is left alone: which hits feed the aura is its own mask's to say.
A combat spell and an aura applying the same spell (Crusader) register once, as the combat spell.

`spelldata.ItemProcUnsupported(trigger, isWeaponProc)` is the single decision about whether the rows
say enough. The generator calls it when it writes the item files, the audit calls it, and a test pins
it, so a proc the generator emits is one the sim can build and a proc it comments out carries the
reason in the comment. `tools/database/unsupported_procs.txt` is the audit's census of every proc an
item, enchant or set can reach and what the rows refuse; `sim/common/forever/registered_effects.txt`
lists the item and enchant effects that do register. Neither is tracked: each is a local baseline its
test writes on the first run and then compares against, naming every line that moved.

## Overrides

`tools/database/overrides/spell_overrides.go` holds the numbers the store carries that the client
database does not state. One row per number:

```go
{16928, PPM, 1, "Annihilator: ProcChance 101 sentinel; TBC 1 PPM, unverified on Forever", "tbc-carryover"},
```

`Reason` and `Source` are not decoration: they are what the next reader weighs the number against, and
the generator refuses a row without a reason. `Source` names where it came from - `tbc-carryover`,
`tooltip`, `wcl:<report>/<fight>`, `issue #N`.

Every field names a rule the generator checks before it writes the value, so an override outlives its
reason no longer than the next regeneration:

|                                   |                                                         | stale when                               |
| --------------------------------- | ------------------------------------------------------- | ---------------------------------------- |
| `PPM`                             | procs per minute, onto `Spell.RPPM`                     | the row states a `SpellProcsPerMinuteID` |
| `FlatThreat`                      | threat the tooltip states without a number              | the spell gains a threat effect          |
| `APCoefDirect` / `APCoefPeriodic` | an attack power coefficient the client keeps in script  | the client states one on that effect     |
| `ProcChancePct`                   | a whole percentage onto `Spell.ProcChance`              | the tooltip states a chance of its own   |
| `DurationMs`                      | an aura duration                                        | the client states a `SpellDuration`      |
| `Hint`                            | the `ProcHint` bits the tooltip's wording did not yield | -                                        |

Two more refusals have nothing to do with the field: an override naming a spell the store does not
carry, and two overrides of the same field on one spell. Every one that is applied leaves an
`// override: <field> <value> -- <reason>` comment on the row it wrote to, so reading the generated
store says which numbers are not the client's.

An item or enchant proc refused as `states no rate` loses that refusal once a `PPM` override sits on
the spell it is routed through - the `TriggerSpellID` of its commented-out registration, the id on its
`// trigger N` line - and registers where it was the only one:

- A combat enchant (Effect 1) or a chance-on-hit item effect: the combat spell itself - Fiery Weapon's
  13897, Unholy Weapon's 20006 - not the spell that grants the enchant. The rate is measured on the
  hits of the weapon carrying it (`NewDynamicLegacyProcForEnchant(id, ppm, 0)`, or `...ForWeapon` for an
  item).
- An equip aura (Effect 3): the aura carrying the proc trigger - Revelation's 1248806 - not the spell it
  triggers (1248808) nor the grant (1248805). The rate is measured on the aura's own proc mask, so an
  aura whose flags decode to no mask stays refused as `no proc mask to measure its rate on`.

A procs-per-minute enchant proc rolls on weapon hits only: its mask keeps its melee and ranged bits,
and a spell or heal hit never procs it. `dpmForMask` strips the rest for every enchant, and
`spelldata.EnchantAuraUnsupported` refuses a rate on an equip aura that hears only spells as
`a procs-per-minute rate hears no weapon hits in this mask`, so it stays listed rather than
registering to never fire. An enchant the game does let spells proc is an exception to state there,
by enchant; there is none today. A stated chance (Fiery Blaze 36's 15%, Insight 8216's 35%) is not a
rate and hears what its row says.

Revelation 8217 stays listed. Its rate is scripted and the client does not state it: trigger 1248806's
`ProcChance` 100 is the sentinel beside the tooltip's "a chance", and its effect entry resolves no stats
from 1248808.

A rate follows the weapon the enchant sits on. A weapon enchant, ranged ones included
(`NewDynamicLegacyProcForEnchantWithMask`), rolls only on the hand carrying it, at that weapon's speed;
an item swap moves it with the weapon. An enchant on no weapon - armor, a cloak, a shield or a
held-in-off-hand item - is priced off the main hand (`NewLegacyPPMManager`, which prices every mask but
the off hand's and the ranged one's at the main hand's speed).

Where a combat spell and an aura apply the same spell (Crusader), the slot whose row states a rate is
the one kept, so the override goes on the combat spell to keep it the combat spell. After adding a row,
run `gen_spelldata` and then `gen_db`, which classifies the procs out of the store compiled into it.
`TestEnchantProcRoutingTakesAPPMOverride` in `tools/database` and `TestSpellDataProcTakesAPPMOverride`
in `sim/common/shared` pin both halves on those three rows with a rate that exists only in the test.
`TestEnchantAuraPPMFollowsItsWeapon` beside the latter pins the routing by slot, and
`TestEnchantPPMFollowsTheEnchantedWeapon` in `sim/core` the move across an item swap.

`overrides.AreaBonuses` is the second table in the same file, for an effect whose tooltip says it is
doubled in some kind of area while the client states no companion row for it ("This effect is doubled
in Volcanic areas" on Molten Fury). A row names the AreaGroup ids any of which counts, the factor on
the effect's amounts and the factor on its duration, onto `Spell.AreaBonusGroups`, `AreaMultiplier`
and `AreaDurationMultiplier`; `row.AreaBonus(&character.Env.Encounter)` reads them against the
encounter's area set, and is stale once the row states a `RequiredAreasID` of its own. The on-use stat
actives `shared.NewSimpleStatActive` registers read it; a proc chance the tooltip says is doubled has
no reader yet.

`tools/database/overrides/extra_spells.go` is the other hand-kept list: spells the generator
force-includes although no class, item, enchant or set reaches them. It is empty today. Each entry
states a reason and a source like an override, and the generator refuses one without a reason. The
extras are added while the store renders, so adding one without regenerating fails the check rather
than shipping a store that does not match its source.

## Reading a row by hand

Rogue, warlock, mage, druid, priest, shaman, hunter and paladin read the store by hand: a class file
names the ladder it needs, picks a rank, and reads every field the registration wants straight into a
plain expression, with no resolver in between. **A row is a source of numbers only** in these eight
classes' files - no `spelldata.SpellConfig`, `AuraConfig`,
`DotConfig`, `ProcTrigger`, `ParseEffects` or `ParseStatic`. `core.SpellConfig` and `core.DotConfig`
are built by hand, field by field, the same shapes [Building an ability](#building-an-ability) and
[Auras and dots](#auras-and-dots) describe for the resolvers, just filled without one.

The generated file is still ladders (see [A class file is ladders](#a-class-file-is-ladders)):
`spellData.ShadowWordPain` is a `spelldata.Ladder`, `.Highest()`/`.ByID(id)`/`.Rank(n)` answer a
`*spelldata.Spell`, and `.Each(fn)` walks every rank in declaration order:

```go
MindBlastRankMap.Each(func(_ int32, rank *spelldata.Spell) {
	priest.registerMindBlastSpell(rank, mindblastCDTimer)
})
```

What a registration reads off the row, since there is no resolver to read it for the caller:

|                                                    |                                                                                                                                                                                         |
| -------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `rank.ID`, `rank.RankNumber()`                     | the `ActionID` and the `Rank` field a `core.SpellConfig` wants; `RankNumber()` is `NameSubtext_lang`'s "Rank N", 0 where the client states none                                         |
| `rank.Cost()`                                      | the cost off the first power the row states, in the sim's units - rage already divided by ten; it answers a `float64`, so an `int32` field like `ManaCostOptions.FlatCost` needs a cast |
| `rank.GCD()`, `rank.CastTime()`                    | the global cooldown and the cast time                                                                                                                                                   |
| `max(rank.Cooldown(), rank.CategoryCooldown())`    | the row's own cooldown, or the one it shares with a category                                                                                                                            |
| `rank.Duration()`                                  | an aura or dot's length; the client's -1 becomes `core.NeverExpires`                                                                                                                    |
| `rank.SpellSchool()`, `rank.DefenseTypeCore()`     | already in core's enums - the accessor converts the client's byte                                                                                                                       |
| `float64(rank.MaxRange)`, `float64(rank.MinRange)` | the plain fields, in yards                                                                                                                                                              |

Straight off `sim/shaman/shocks.go`:

```go
func (shaman *Shaman) newShockSpellConfig(rank *spelldata.Spell, spellSchool core.SpellSchool, shockTimer *core.Timer) core.SpellConfig {
	return core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.ID},
		SpellSchool: spellSchool,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       SpellFlagShamanSpell | SpellFlagShock | core.SpellFlagAPL | SpellFlagInstant,
		MaxRange:    float64(rank.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: rank.GCD()},
			CD: core.Cooldown{
				Timer:    shockTimer,
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: rank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,
	}
}
```

**An amount is `Average(core.CharacterLevel)`, or `Roll(sim, level)` where the class rolls it.** The
seven classes ported first read every amount as `Average`, both ends of a `(low, high)` alike, because
the literals they replaced never rolled. Paladin's did, so its damage and heals take `Roll` and its
per-hit and aura numbers `Average`. A row can still carry a spread (Frostbolt's `Variance` is 0.105)
and `Roll(sim, level)`/`Min`/`Max` read it where a class asks for one; `Average` is what keeps a
ported number equal to the one it replaces, not an absence of spread in the data.

**Name the effect by role where one applies; by position where the row hides it behind another
effect.** `DamageEffect()`, `HealEffect()`, `EnergizeEffect()` and `PeriodicEffect()` answer the first
effect of their kind, which is right wherever a spell has only one. Hellfire's tick is not:
`PeriodicEffect()` would find the `A_PERIODIC_TRIGGER_SPELL` at position 1, the trigger that fires
the Hellfire Effect spell each tick, rather than the damage tick itself at position 2:

```go
var hellfireRank = spellData.Hellfire.Highest()

// The tick sits at effect position 2: position 1 is the A_PERIODIC_TRIGGER_SPELL that fires the
// Hellfire Effect spell each tick, not the damage tick.
var hellfireTick = hellfireRank.EffectN(2)
var hellFireCoeff = hellfireTick.Coeff()
```

**A tick the description keeps on another spell is read off that spell, not the row that names it.**
Where the family has its own `XTriggered` ladder for the spell the trigger fires, that ladder is the
edge; failing that, `row.Triggered()[0]` (an actual `TriggerID` edge) or `row.Refs()[0]` (a tooltip
reference) reach the same spell, and a bare `spelldata.MustFind(id)` is the last resort, with a
comment naming why no other edge is unique. Hurricane's periodic damage sits on the spell its aura
triggers each tick, which the family's own `HurricaneTriggered` ladder names; the tick length still
comes off Hurricane's own row - the periodic dummy that carries no damage of its own:

```go
tickLength := hurricaneRank.Effect(dbcenums.A_PERIODIC_DUMMY, 0).Period()

// Hurricane's periodic damage is the spell HurricaneTriggered casts each tick.
hurricaneTickSpell := spellData.HurricaneTriggered.Highest()
hurricaneTick := hurricaneTickSpell.DamageEffect()
```

**`EffectAt(n)` counts from 1 by position, like `EffectN` - not the client's `EffectIndex`.**
`EffectAt(n+1)` lines up with client index `n` only where the row's indices run contiguously from 0;
where they do not, `Effect(aura, misc)` names the effect instead. Rogue's `PuncturingWounds` talent
needs it: its two crit modifiers share an aura and misc, so `Effect(aura, misc)` cannot tell them
apart, and sit at positions 1 and 3 around the proc trigger at position 2 (client index 1);
`EffectAt(3)` is the one Mutilate reads:

```go
// The proc trigger sits between the two crit modifiers, at effect position 2; the Mutilate
// crit bonus is the second of the two, at position 3, and they share an aura and misc so have
// to be indexed rather than named.
FloatValue: spellData.PuncturingWounds.EffectAt(3).ValueAt(rogue.Talents.PuncturingWounds),
```

**A `Ranked` family's rank can carry a per-level gain a `Ladder`'s per-rank readers do not add in.**
`EffectAt(n)` and `Effect(aura, misc)` answer base points across the ranks, which is right for a
`Talent`, whose curve states base points outright. A `Ranked` family - one spell per rank - can still
carry `EffectRealPointsPerLevel` on that effect, so `ValueAt`/`FractionAt`/`MultiplierAt`/`TenthsAt`
read low wherever the client scales it past the rank's own `SpellLevel`. Read the rank's own effect
through `Average` instead of the ladder's per-rank reader:

```go
X.Rank(rank).EffectN(1).Average(core.CharacterLevel)   // not X.EffectAt(1).ValueAt(rank)
```

**A dot ticks on current stats, so there is no `OnSnapshot`.** `OnTick` reads the effect's own
`Average` straight into `CalcAndDealPeriodicDamage` (or `CalcAndDealPeriodicHealing`) every time it
fires, on whatever stats and multipliers are in force then. An input the cast fixes - combo points
about to be spent, a stack about to be consumed - is read where the cast reads it and kept in a
variable the `OnTick` closure captures, never re-read at the tick. Rip's combo points are read in
`ApplyEffects`, before `SpendComboPoints` runs them to zero - not in `OnTick` after, where the count
would already be gone:

```go
var cp int32

// ...
	ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
		if result.Landed() {
			cp = druid.ComboPoints()
			spell.Dot(target).Apply(sim)
			druid.SpendComboPoints(sim, spell.ComboPointMetrics())
		}
		spell.DealOutcome(sim, result)
	},
	OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		ap := dot.Spell.MeleeAttackPower(target)
		var tickDamage float64
		switch {
		case cp <= 3:
			tickDamage = 990 + 0.18*ap
		case cp == 4:
			tickDamage = 1272 + 0.24*ap
		default: // 5
			tickDamage = 1554 + 0.24*ap
		}
		dot.Spell.CalcAndDealPeriodicDamage(sim, target, tickDamage/6, dot.OutcomeTick)
	},
```

**Hand numbers stay hand numbers - Go literals with a review comment.** A hand-supplied threat
number, PPM or coefficient keeps the same marker a resolver-built config carries. `sim/warrior/hamstring.go` writes it as an assignment on the config the resolver already
built:

```go
// TODO: Manual review needed -- the client states no threat coefficient; 1 until measured in game.
config.ThreatMultiplier = 1
```

A hand-built `core.SpellConfig`, with no resolver to build it first, carries the identical comment on
the identical field, as a struct-literal line instead:

```go
// TODO: Manual review needed -- the client states no threat coefficient; 1 until measured in game.
ThreatMultiplier: 1,
```

### Worked examples

#### Shadow Word: Pain's tick

`sim/priest/shadow_word_pain.go`:

```go
tick := rank.PeriodicEffect()
tickLength := tick.Period()

priest.RegisterSpell(core.SpellConfig{
	ActionID:       core.ActionID{SpellID: rank.ID},
	SpellSchool:    core.SpellSchoolShadow,
	DefenseType:    core.DefenseTypeMagic,
	ProcMask:       core.ProcMaskSpellDamage,
	Flags:          core.SpellFlagAPL,
	ClassSpellMask: PriestSpellShadowWordPain,
	Rank:           rank.RankNumber(),
	MaxRange:       float64(rank.MaxRange),

	ManaCost: core.ManaCostOptions{
		FlatCost: int32(rank.Cost()),
	},

	Cast: core.CastConfig{
		DefaultCast: core.Cast{GCD: rank.GCD()},
	},

	Dot: core.DotConfig{
		Aura: core.Aura{
			Label: fmt.Sprintf("ShadowWordPain-%d", rank.RankNumber()),
		},
		NumberOfTicks:       int32(rank.Duration() / tickLength),
		TickLength:          tickLength,
		AffectedByCastSpeed: false,
		BonusCoefficient:    tick.Coeff(),

		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.Spell.CalcAndDealPeriodicDamage(sim, target, tick.Average(core.CharacterLevel), dot.OutcomeTick)
		},
	},

	ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
		return spell.CalcPeriodicDamage(sim, target, tick.Average(core.CharacterLevel), spell.OutcomeExpectedMagicHit)
	},
})
```

No `OnSnapshot`, no `useSnapshot` branch: the tick is `tick.Average(core.CharacterLevel)` wherever it
is asked for, cast or projected.

#### Consecration's borrowed tick

Each rank's tooltip names one spell for both ticks, and the family's own `ConsecrationTriggered`
ladder carries those spells rank for rank (`Refs()[0]` reaches the same row). The two ticks sit on its
first two effects by position, since both are school damage and `DamageEffect()` cannot tell them
apart: the base tick on effect 1, the bonus its first four targets take, with its own spell power
share, on effect 2. The periodic dummy on Consecration's own row states no tick: its period times the
ticks, and its points are how many targets take the bonus, `sim/paladin/consecration.go`:

```go
tickSpell := spellData.ConsecrationTriggered.Rank(n)
tick := tickSpell.EffectN(1)
bonus := tickSpell.EffectN(2)

dummy := rank.Effect(dbcenums.A_PERIODIC_DUMMY, 0)
bonusTargets := int(dummy.Average(core.CharacterLevel))
tickLength := dummy.Period()
numberOfTicks := int32(rank.Duration() / tickLength)

dealTick := func(sim *core.Simulation, dot *core.Dot) {
	for i, target := range sim.Encounter.ActiveTargetUnits {
		damage := tick.Average(core.CharacterLevel)
		if i < bonusTargets {
			damage += bonus.Average(core.CharacterLevel) + bonus.Coeff()*dot.Spell.BonusDamage(dot.Spell.Unit.AttackTables[target.UnitIndex])
		}
		dot.Spell.CalcAndDealPeriodicDamage(sim, target, damage, dot.OutcomeTickMagicHit)
	}
}
```

#### A heal and a mana restore

```go
heal := rank.HealEffect().Average(core.CharacterLevel)      // both ends, where the class does not roll
mana := rank.EnergizeEffect().Average(core.CharacterLevel)  // 0 on a rank with no Energize effect at all
```

#### Registering several ranks

Forever's Starfire tops out at rank 7, so the spec's own subset ladder names ranks 6 and 7 by their
ids rather than a rank the table does not carry, `sim/druid/starfire.go`:

```go
var StarfireRankMap = spelldata.Ranked(spellData.Starfire.Rank(6).ID, spellData.Starfire.Rank(7).ID)
```

#### An attack power coefficient

Nothing in the client states an attack power coefficient per combo point, so Rupture's sits as a
literal table indexed by combo points, right beside the number it scales, `sim/rogue/rupture.go`:

```go
func (rogue *Rogue) ruptureDamage(target *core.Unit, comboPoints int32, baseDamage float64, damagePerComboPoint float64) float64 {
	return baseDamage +
		damagePerComboPoint*float64(comboPoints) +
		[]float64{0, 0.01, 0.02, 0.03, 0.03, 0.03}[comboPoints]*rogue.Rupture.MeleeAttackPower(target)
}
```

## Regenerating and checking

```
go run ./tools/database/gen_spelldata              # rewrite every generated spell data file
go run ./tools/database/gen_spelldata -check       # name the ones that are stale, write nothing
go run ./tools/database/gen_spelldata -unchecked   # write without the type-check below
make spelldata-check                               # the same check
```

The generator reads `tools/database/wowsims.db` and writes all of it in one pass:
`sim/core/spelldata/spells_auto_gen.go`, `sim/core/dbcenums/forms_auto_gen.go`, a
`sim/<class>/spell_data_auto_gen.go` of ladders into the store for every class, and the raid buffs,
which [Buffs and debuffs](#buffs-and-debuffs) describes. It is its own binary rather than a mode of `gen_db` on purpose:
`gen_db` imports the sim and the sim reads these files, so a stale one would stop the generator that
fixes it from compiling. For the same reason nothing is written until all of it type-checks: the
rendered bytes go to a staging directory first and are compiled through `go build -overlay` in the
place of the committed files, and a failure leaves the tree exactly as it was and prints what the
compiler said. `-unchecked` skips that build, for a class file whose call sites cannot type-check
against what it will state until they move - the file is written first and the call sites follow;
`-check` closes the loop once the package builds again. `make db` runs the store before `gen_db` for the same ordering reason - `gen_db`
classifies every item and enchant proc out of the store compiled into it.

Nothing lists which spells to generate. The class files walk `dbc.Classes`, take each class's own skill
lines and emit a family for every spell whose subtext reads `Rank N`; the store starts from what those
families name, from the item, enchant and set-bonus tables, and from the talent trees, and closes over
everything those reach. A family that could not be resolved is named in the `// Not generated:` comment
at the head of the class file. The one hand-kept list is `overrides.TalentGrantedAbilities`: a one-rank talent
node on an ability nothing trains (Shifting Power) is filed on a single-rank ladder through the skill line row that
grants it, which the generator would otherwise skip as an ability the game teaches under its own ranks.

`assets/db_inputs/spell_store_inputs.json` is the client rows the store was built from, committed
beside it, and every `SpellShapeshiftForm` row whole alongside them, since a form names no spell for
the store's closure to reach it by. It is gzip-compressed JSON despite the name, like everything under
`assets/db_inputs/dbc` - `zcat` it to read it. It exists so the store, the buffs and the shapeshift
forms can all be rebuilt without the client database, which is gitignored and comes from a local WoW
install, and `-check` compares it too: a capture that no longer matches the database would render the
committed files all the same, so nothing else would notice it going stale.

What guards the outputs:

- `TestStoreRegeneratesFromTheCommittedInputs` in `tools/database` re-derives the closure, the hand
  links, the curves, the tooltip hints, the overrides and the emitter from that capture and asserts the
  committed store is what comes out. No build tag and no database, so it runs everywhere.
  `TestBuffFilesRegenerateFromTheCommittedInputs` does the same for the buff files, and
  `TestFormsRegenerateFromTheCommittedInputs` for `sim/core/dbcenums/forms_auto_gen.go`.
- `sim/core/spelldata/snapshot_test.go` pins the shape and the counts of the committed store, and a
  handful of rows read out of the client by hand, so a regeneration that moves a number says so there
  instead of in a sim result.
- The repository's `pre-commit` hook runs `-check` when the database is present and the commit touches
  the generator or one of its outputs.

Goldens are the last check and the one that costs time: run the suites of the class that changed, and
diff the `.results` against the `.results.tmp` with a local goldendiff helper or by hand. A port that
was meant to be mechanical and moves a golden is a wrong port, not a new baseline.

## Porting a class onto the resolvers

1. **Start from the ladders.** Every class file is already one ladder per family, and the eight
   classes other than the warrior read them by hand, the way [Reading a row by hand](#reading-a-row-by-hand)
   describes. From hand reads onto the resolvers - `SpellConfig`, `AuraConfig`, `DotConfig`,
   `ParseEffects`, `ProcTrigger` - is the warrior's move, and this checklist is written for it.
2. **Dump the rows before touching a call site**, three ways: the effects at the rank taken, which
   registered spells each modifier's `EffectSpellClassMask` names, and the whole `ProcTrigger` the row
   decodes to. The mask dump is what turns a golden move into something you knew before you ran it.
3. **Pass `Ladder.Rank(n)`, never `MustFind(id)`,** for anything the talent tree prices: a trait
   talent's per-rank numbers live in the curve, and the base row's effect can read 0.
4. **Check the cooldown categories.** A category cooldown resolves to `Unit.CategoryTimer`, which is
   one map per unit and the same one consumables and racials use for categories 4, 30, 1141, 1153 and 1190. A class row stating one of those would share a cooldown with a potion. Two of the class's own
   abilities sharing a category share the timer, which is what the client does: the warrior's Revenge
   and Overpower run off 65, Shield Bash and Pummel off 88, and Mortal Strike, Bloodthirst and Shield
   Slam off 971.
5. **Read what the resolver picked up.** `SpellFlagHelpful` follows the first effect's target and flips
   `IsFriendly` in the APL editor, so an attack whose first effect is a self side-effect needs it
   cleared. `Rank` comes from "Rank N" and flips `HasRanks` there. `IgnoreHaste` follows the physical
   school, not the defense type.
6. **A row with a `Variance` rolls.** Use `Roll(sim, level)` where the client states a spread; the
   extra draw reshuffles every later roll, which is a golden move with a known cause.
7. **Parse before you activate.** `ParseEffects` attaches on gain, so an aura already up when the parse
   runs loses the rows that need a `Simulation`.
8. **Compare value by value before deleting a hand modifier.** A mod that lands in a different bucket -
   a school pseudo-stat where the sim had a per-spell mod - is the same number in a different place,
   and a DPS diff will not show it. Diff the `.results` as well as the DPS numbers.
9. **`RequireDamageDealt` defaults true.** A listener that fires on a dodge, a parry, a miss or a block
   needs it false, which the decoder has already done where the tooltip named the outcome and the row
   carries `ProcHintOutcomeTaken`; clear it by hand only where the wording yielded no hint. The outcome
   itself is always the caller's.
10. **Rename a trigger whose driver and buff share a name**, and split a row that decodes to hits dealt
    _and_ taken into two listeners with a condition each.
11. **Say which source won.** Where the row and the tooltip disagree, the decision goes in a comment
    that names the tooltip's wording. Where a number stays by hand, its review marker stays with it.
12. **Isolate every golden move** by reverting exactly one change and re-running. Two causes in one
    commit look like one inexplicable cause.
13. **Delete the class masks last**, and only where the constant's sole reader is the `ClassSpellMask`
    write. A handler calling `spell.Matches` is a reader too.
14. **A gear proc is three calls, not one.** `ProcTrigger` for the listener, `AuraConfig` for the buff
    it applies and `ParseEffects` for what that buff does - the idiom `sim/warrior/items.go` uses for
    every set bonus and item buff the class owns.
15. **`Melee` and `Magic` put the spell in the rotation; `Flags` does not.** A stance-gated
    ability takes `Flags` so the APL editor does not offer it, and a config that writes
    `ThreatMultiplier = 1` or `FlatThreatBonus = 0` after the resolver is writing what the resolver
    already wrote - keep it only where its review marker says the number is still to be measured.

## Traps

**The data contains ranks the game never grants.** Fireball 38692 and Frostbolt 38697 are rank 14
entries at level 70 whose `SkillLineAbility` rows and spell attributes are byte-for-byte
indistinguishable from the real rank 13s, so the generator cannot filter them. Reaching for
`Highest()` on those families silently casts a spell that does not exist - it cost 1.2% DPS when it
happened during the mage port. Use `ByID`.

**A rank can carry nothing in a role.** Lay on Hands rank 1 restores no mana where ranks 2-4 do, so
`EnergizeEffect()` answers `NilEffect` there, which every accessor reads as zero.

**Never delete a generated file before regenerating it.** The class package stops compiling, and
`gen_db` - which imports the sim - then cannot build either. Regenerate over the top, or
`git checkout HEAD -- <path>` to get back.

**A melee ability states its bonus as weapon damage, not school damage.** Sinister Strike's +98 is
`E_NORMALIZED_WEAPON_DMG`, and Holy Strike's flat part sits there too; `DamageEffect()` answers the
first weapon or school damage effect alike, so the read is the same either way.

**Rage is stored on a 0-1000 bar.** Heroic Strike's cost reads 150 where the player sees 15.
`PowerCost` and `Cost()` divide it through; mana, energy and focus need no conversion, and an
`Energize` effect that restores rage is still in tenths, which `Tenths()` reads.

**A new client table needs a settings line.** `SpellCastTimes` was missing from
`generator-settings.json` and cast times read zero until it was added and `make db` re-run. Adding a
table is one line; the extractor needs no code.

**`EffectN` counts positions, `Effect.Index` is the client's number.** `EffectN(1)` is the first effect
the row carries, whatever index the client gave it; 46 rows state an index that is not the position it
sits at. A store row read with the client's number in hand is off by one wherever the two agree, and
wrong in a different way where they do not.

**`Average` floors the whole amount, in float32.** The client resolves an amount to a whole number,
so an effect whose `EffectBasePointsF` is fractional - about one in 55 - keeps its fraction under the
per-level gain and is floored once at the end. Reading `BaseValue()` and doing the arithmetic in
float64 gives a different answer on those rows.

**A row with more than one effect has to name the one it means.** `Ladder.ValueAt` panics rather than
read the first of several, and `Effect(aura, misc)` panics when two effects match. Both are the same
rule: `EffectAt(n)` or `EffectN(n)` is how a caller says which.

**A hand-built `Effect` scales from its own `SpellLevel` and `MaxLevel`.** The generator stamps both onto
every effect so `Average(level)` needs no lookup; an `Effect` literal in a test or a hand-written config
that leaves them zero scales `PPL` from level 0 up to the caster's level. Copy them from the spell's row,
or set `PPL` to 0.

**`MustFind` at package init fails at startup, not at the call site.** That is the point - a
regeneration that drops or renumbers an id stops the sim with the id in the message - but it means a
package-level `var` reaching for a spell this build does not carry takes the whole package down,
including tests that never touch that spell.

**`SPCoef` of exactly 1 is the column's filler.** 2,113 of the store's 9,651 effects carry it, on
weapon-damage effects, speed auras and shapeshifts among them, and no physical-school row in this
client states a fractional coefficient at all. `Magic()` and `DotConfig` hand the row's coefficient
to core as it stands, so a physical row read through either would take spell power per hit or per
tick: read the tooltip before you believe a coefficient of 1.

**`Spell.ProcChance` is not always a chance.** 100 and 101 are the client's "fires whenever its own
condition is met", and the tooltip is what says which the column is: `ProcChanceSource`, baked in at
generation, is the answer, and `spelldata.ProcTrigger` reads it rather than the column.

**A proc row with no stated rate panics when the trigger is built.** `ProcChancePPM` with no override
behind it means the client states nothing anywhere, so `spelldata.ProcTrigger` demands a `PPM()` rather
than build a listener that fires on every hit.

## Buffs and debuffs

Every raid, party, individual and enemy-debuff proto field has one row in
`tools/database/buffmanifest`. The manifest owns the field numbers and names the spell each row reads;
the store owns the values. The same `gen_spelldata` pass that renders the store renders, from the same
captured inputs:

- `sim/core/buffs/buffs_auto_gen.go` and `sim/core/buffs/debuffs_auto_gen.go`, a constructor per row;
- `ui/features/settings/model/buffs_debuffs_auto_gen.ts`, the settings inputs;
- `proto/buffs.proto`, the four messages.

They go through the same staging type-check and the same `-check` as the store, and regenerate without
the client database.

### How a buff reads the store

The generated constructors live in `sim/core/buffs`, above core, because the store imports core. Each
supported row holds a package-level `*Meta` and reads every number off it at runtime, through the same
`spelldata.ParseEffects` a class's own aura reads its stats through - a generated buff carries no effect
routing of its own:

```go
var battleShoutSpell = spelldata.MustFind(25289)
var battleShoutMeta = &Meta{
	Label:      "Battle Shout",
	Spell:      battleShoutSpell,
	Category:   BattleShoutCategory,
	SingleAura: true,
}

func BattleShoutValue(talentPoints int32) float64 {
	return battleShoutMeta.Value(talentPoints)
}
func BattleShoutDuration(talentPoints int32) time.Duration {
	return battleShoutMeta.Duration(talentPoints)
}
func BattleShoutAura(unit *core.Unit, isPlayer bool, talentPoints int32) *core.Aura {
	return newBuff(unit, battleShoutMeta, isPlayer, talentPoints)
}
```

`Meta.Options(talentPoints)` states the `spelldata.ParseOpt`s every row needs:
`spelldata.RaidBuffOptions` - `Level(core.CharacterLevel)`, `BuffAuras`, `SkipAuras` where the
generator found any and `FullComboPoints` for a finisher, which the generator parses with too - and `ScaledBy`
the improving talent (left out where the talent scales the duration instead, since `Duration` reads
that separately). `newBuff` adds `SchoolResistances` and, where
`Category` is set, `Exclusive(Category, true)` for a `SingleAura` row or `ExclusivePerStat(Category)`
for any other, so a generated buff and a hand-written scroll of the same stat or resistance bid
against each other under the categories core's own exclusive stat buffs use. `newDebuff` adds only
`Exclusive(Category, SingleAura)`, since a debuff never carries a resistance of its own;
`newItemCountBuff` adds `Count` so that a party with several of the same item is worth that many
copies of the amount. `Value` is a damage shield's own effect where the spell states one, and otherwise
the first thing `spelldata.DryRun` attaches for those options. `Duration` reads the spell, or `Cast` where the
spell states no duration of its own; `Cooldown` reads `Cast` where the `Meta` carries one - an
external cooldown whose `CastID` is not its `SpellID` - else the spell itself; both go through the helpers in `sim/core/buffs/amounts.go`, and a talent that scales the
duration truncates it the way `talentScaled` truncates an amount everywhere else. A row whose aura is
a damage shield skips the parse outright: `newDamageShield` calls `core.NewDamageShield` with the
spell's school and `Value`.

`sim/core/spelldata/buff.go` holds a second table, read only when a parse states `BuffAuras()`:
`A_PERIODIC_ENERGIZE` on a mana row becomes MP5 there, and `A_MOD_ATTACK_POWER_PCT` a percentage on
attack power. Neither sits in the parse's shared table, because warrior rows carry them for a value the
class wires itself - Bloodrage's rage tick is an `A_PERIODIC_ENERGIZE`, Berserker Stance Passive an
`A_MOD_ATTACK_POWER_PCT` of 0 - and reading either off the shared table would change what those rows
mean there.

The generator asks the same parse for every manifest row: `spelldata.DryRun`, given the row's options
and no character, answers what it attaches, and `SkippedNotes` on that answer what it leaves out, so a
row the generator writes is one the sim builds the same way. An effect the parse attaches at 0 beside
the buff - the healing-taken row every paladin aura states - goes into the row's `SkipAuras`, so the
built aura leaves it out. A row a driver decides the meaning of - `KindExternalCD`, `KindProc`,
`KindManual` - is written whatever the parse attaches; a damage shield needs its shield effect, and
every other row an amount. A row the parse attaches nothing of renders as a commented shell naming the reason, and a row
it does write states what it could not read as a `// Left out:` note above it in the generated file,
one per aura effect the parse has no row for.

Core applies the buffs through `core.BuffHooks`, which `sim/core/buffs` registers from its `init`:
`ApplyBuffs`, `ApplyDebuffs` and the Gift of Arthas elixir's debuff. A sim that
never imported the package panics naming the import. `sim/common` imports it, which links it into every
sim and every class test; core's own tests reach it through the generated-buff tests, which sit in
`package core_test` beside `sim/core/export_test.go`.

### Adding a buff

1. Append the field's row to its message's slice in `tools/database/buffmanifest/buffs.go`. State the
   `Field` and the `SpellID` its numbers are read from; a `CastID` where the player learns another
   spell than the aura, as for a totem; a `Talent` or an `ImpAction` where something improves it; a
   `Category` where other sources of the same effect bid against it; and the `Stats` it matters to. A
   row that is not simply its aura states a `Kind`.
2. `go run ./tools/gen_buffs_proto` and `make proto`, so the compiled protos carry the field. The pass
   below type-checks against them and writes nothing while the field is missing.
3. `go run ./tools/database/gen_spelldata`. The spell becomes a root of the store, and the constructor,
   the apply block and the settings input are written. Read the `buffs:` lines it prints.
4. A row the kind cannot express outright states `Driver: true`, and `sim/core/buffs/drivers.go`
   declares `drive<Go>`.
5. With a database, `TestManifestAnchorsMatchTheClient` checks the pin is the top rank the owner's skill
   lines grant.

### The manifest row

`BuffSpec` in `tools/database/buffmanifest/manifest.go` holds what the client cannot state about a
field. `buffs.go` holds the rows in four slices, `Raid`, `Party`, `Individual` and `Debuffs`, one per
proto message and in the order the settings tab lists them. A row's scope is the slice it sits in and
its proto number its position there, so each message's numbers are dense from 1, and a row placed
anywhere but the end renumbers the rows after it. `buffmanifest.All()` pairs every row with both as a
`Row`.

| Field            | What it is                                                                                                                                                                                                                                                                                                                         |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Field`          | the proto field, snake_case                                                                                                                                                                                                                                                                                                        |
| `SpellID`        | the spell the numbers are read from, and a root of the store: the top rank of the castable family, or the aura that family's cast applies when the cast is a summon or a dummy                                                                                                                                                     |
| `CastID`         | the cast the player learns, where it is not `SpellID`. It names the buff and times an external cooldown: Mana Tide's aura 17360 carries the mana, the cast 17359 the 13 seconds and the 5 minutes                                                                                                                                  |
| `Talent`         | the spell of the trait node that prices the improved state. Its one spell modifier whose class mask reaches `SpellID` is the improvement - none or two fails the pass - and a `SPELLMOD_DURATION` modifier scales the duration, any other the value. The generated `Meta.TalentEffect` names that effect by its `EffectN` position |
| `Category`       | the exclusive-effect category the aura bids in, `""` for none                                                                                                                                                                                                                                                                      |
| `SharedCategory` | a second category the aura joins without an effect of its own, which is how the paladin auras exclude each other across schools. Applied to the player's copy only, and declared once in the generated file as `<Name>Category`                                                                                                    |
| `SingleAura`     | the category holds one aura at a time, so the loser is deactivated rather than outbid                                                                                                                                                                                                                                              |
| `Driver`         | the apply block hands the row to `drive<Go>` instead of activating the aura outright                                                                                                                                                                                                                                               |
| `Stats`          | the UI relevance tags a spec's `epStats` and `displayStats` are matched against                                                                                                                                                                                                                                                    |
| `ImpAction`      | the improved state's source when it is not a talent - an item, or the spell an item set grants at a piece threshold - and the icon that state shows                                                                                                                                                                                |
| `Label`          | a UI label override                                                                                                                                                                                                                                                                                                                |
| `Kind`           | what the row is when it is not simply the aura its spell states, below                                                                                                                                                                                                                                                             |
| `Proto`          | an override of the derived proto type, for a flag that is a number: `retribution_aura_spell_power` is `ProtoDouble`                                                                                                                                                                                                                |
| `Owner`          | the providing class, only for a spell no class family files: the four Atiesh rows. A row whose spell a family files and that names an owner fails the pass                                                                                                                                                                         |
| `Reason`         | what a `KindFlag` row is, which the shell it renders as carries. Required on a flag and refused on any other row                                                                                                                                                                                                                   |

The last four are exceptions; most rows state none of them. The rest of a row is derived:

- **Go stem.** `GoStem()` is `GoField()`, protoc-gen-go's name for the field, without underscores:
  `battle_shout` gives `BattleShoutAura`, `BattleShoutValue`, `BattleShoutDuration`,
  `BattleShoutCategory`, `battleShoutSpell` and `battleShoutMeta`.
- **Proto type.** `ProtoType()` is `ProtoTristate` where a `Talent` or an `ImpAction` prices an improved
  state, `ProtoInt32` for `KindExternalCD` and `KindItemCount`, and `ProtoBool` otherwise. The pass
  checks it against the compiled message, and renders a row whose field protoc has not retyped yet as a
  shell.
- **Name and AuraName.** `Name` is the client's `SpellName` of `CastID`, or of `SpellID` where no cast
  is pinned. `AuraName` is the `SpellName` of `SpellID`, only where a cast is pinned. `Name` is the label
  unless `Label` overrides it; a spell no class family files has neither and is labelled by its own name. `TestManifestAnchorsMatchTheClient` resolves the
  pins from both.
- **Owner.** The class whose `core.ClassSpellFamilies` entry is the spell's family. It narrows the rank
  lookup, and marks the row "(External)" on that class's settings tab.
- **SkipAuras.** Every effect the parse attaches at 0, as above: the six paladin auras skip
  `A_MOD_HEALING_PCT`.
- **Damage shield.** A plain row whose spell applies `A_DAMAGE_SHIELD` is `KindDamageShield`.
- **Talent ranks.** The trait node's ranks are the points the apply block hands the constructor for an
  improved state.

### The kinds

`KindPlain`, the zero value, is a row whose aura the parse builds outright, and `KindDamageShield` is
read off the spell, as above. The rest are stated: `KindExternalCD` is a count of other players' casts,
which a driver schedules on the cast's cooldown; `KindProc` needs a driver for the trigger;
`KindItemCount` takes a count and applies its amounts once per item; `KindManual` is a row the sim
models by hand through its driver; and `KindFlag` is a sim input rather than a buff (a toggle, or a
number such as `retribution_aura_spell_power`), which names no spell and renders as a commented shell
carrying its `Reason`. A row renders as a shell too when its spell states no aura effect the parse
attaches, when an external cooldown's cast states no cooldown or its aura no duration, or when the
compiled field has another type. No `Proto` value is an enum, and a field that wants one would add its
own value and a name for it in both emitters.

### The apply order

`buffmanifest.Scopes` is Party, Raid, Individual, Debuffs: the rows resolve, render and apply in that
order, each scope in slice order. The apply order is the order the auras register in, and the results
follow it. The party's Retribution Aura and the raid's Thorns are both damage shields, and a hit taken
reaches them in the order they activate, which is the order they registered, so the protection suites' results hold with Party first.
`proto/buffs.proto` keeps its own message order: Raid, Party, Individual, Debuffs.

### Warnings

The pass prints a `buffs:` line for what it resolves but doubts, and writes the row regardless:

- **An unpriced talent.** A passive trait node whose flat or percent spell modifier raises an effect
  value of the row's spell, on a row that names no `Talent`. It prices an improved state the row does
  not offer.
- **A talent with nothing to scale.** A `Talent` that scales the value of a row the parse attaches no
  amount of. The row keeps no ranks, so `TestResolvedBuffInvariants` rejects it as a tristate nothing
  prices.
- **A scope mismatch.** The client states an area or a target the row's slice does not name.
  `TestScopeMatchesTheClientTargeting` holds the exemptions.

### What stays hand-written

Judgement of the Crusader states its effect in a shape no manifest row can carry - a holy school
alone - so its row is a shell and `applyDebuffs` in `sim/core/buffs/debuffs.go` applies the
hand-written aura after the generated ones. The paladin's own aura and judgement ranks
(`PaladinAuraRank`, `JudgementRank`) live in `sim/core/buffs/paladin.go` beside the generated
categories they join.

### Drivers

A row the generator cannot express outright states `Driver: true`, or is a kind that always needs one,
and the apply block calls `drive<Go>` instead. The contract is in `sim/core/buffs/drivers.go`: a buff
row's driver takes the `*Character` and the whole scope message, a debuff row's takes the `*Unit`, the
debuffs and the raid. Handing over the message rather than the one field is what lets Grace of Air read
`party.TotemTwisting`. The driver builds the generated aura with `<Go>Aura(...)` and adds what the
client does not state: a proc trigger, a cooldown, a delay, a regen. Declaring the function is what
makes the row compile.

### Regenerating

`go run ./tools/database/gen_spelldata` writes the buff files with everything else, and `-check` names
them when they are stale. `tools/database` imports `sim/core`, so the generator cannot run while the
tree it generates into does not compile. A change that breaks it - retiring a proto field, deleting a
driver the generated file still calls - needs the generated file cut by hand first: delete the lines
that name the field or symbol that is going away, `go build ./...`, then run the generator, which
writes the whole file back. `tools/gen_buffs_proto` renders `proto/buffs.proto` alone, with nothing but
the manifest, for the one moment the sim cannot build: before protoc has seen a new field.

### The guard tests

`go test ./tools/database/...` needs no client database and runs in CI; only
`TestManifestAnchorsMatchTheClient` and `TestProcShapeOfNamedSpells` skip without one.

| Test                                                                       | What it holds                                                                                                                                        |
| -------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TestUniqueScopeField`, `TestUniqueGoStem`                                 | no two rows collide                                                                                                                                  |
| `TestFlagRowsHaveAReason`, `TestResolvableRowsNameASpell`, `TestProtoType` | the schema rules above                                                                                                                               |
| `TestBuffsMatchProto`                                                      | the committed `proto/buffs.proto` declares every row at its number and type, and nothing else                                                        |
| `TestFieldNaming`, `TestFieldNamesRoundTrip`                               | `GoField()` and `TSField()` reproduce protoc's and protobuf-ts's camel case                                                                          |
| `TestRenderProtoNextIndex`, `TestRenderProtoHeader`                        | the next free number above each message, and the header protoc reads                                                                                 |
| `TestBuffFilesRegenerateFromTheCommittedInputs`                            | the two Go files, the settings inputs and `proto/buffs.proto` are what the committed store inputs render                                             |
| `TestRenderedBuffFilesMatchTheFixtures`, `TestRenderedBuffFilesCompile`    | synthetic rows of every shape render to the committed fixtures, and those fixtures compile against the real `sim/core/buffs` through a build overlay |
| `TestRenderBuffsDebuffsTS*`                                                | the settings inputs each proto type and kind renders                                                                                                 |
| `TestResolvedBuffInvariants`                                               | a tristate row is priced by a talent or an `ImpAction`, and the pinned talent ranks, categories and stat amounts                                     |
| `TestScopeMatchesTheClientTargeting`                                       | a row whose spell states an area aura, or an aura aimed over an area, sits in the scope that targeting names                                         |
| `TestManifestAnchorsMatchTheClient`                                        | with a database, each pinned spell is still the top rank, aura or cast the client's skill lines grant                                                |

The generated constructors' behaviour - categories, stacks, drivers - is held by
`sim/core/buffs_generated_test.go` and `sim/core/debuffs_generated_test.go`. Rewrite the synthetic
fixtures with `UPDATE_BUFF_FIXTURES=1 go test ./tools/database/`.

### Traps

**A negative amount that scales by level floors away from zero.** `Average` floors the folded amount,
so Demoralizing Shout's -196 and -1.4 a level read -205 at 60, and Demoralizing Roar's -193 does too;
a client that truncated the per-level part toward zero would state -204. The store's reading is the one
the sim takes until the game says otherwise.

**A debuff prices at the character's level, not the target's.** `Meta.Options` states `Level`
unconditionally, so `newDebuff` reads the row at `core.CharacterLevel` (60) whatever level the target
carries. Demoralizing Shout's -196 and -1.4 a level reads -205 against a level 60 target and -205 still
against a level 63 raid boss, not the -209 pricing the boss's own level would give: a raid's debuff is
cast by a player of the sim's level, and the target has no character for the parse to read a level off
in the first place.

**`A_PERIODIC_ENERGIZE` and `A_MOD_ATTACK_POWER_PCT` read only for a buff.** Both sit in the table
`BuffAuras()` adds rather than the shared one. Bloodrage reads its rage tick by hand
(`sim/warrior/bloodrage.go`); Berserker Stance Passive's own `A_MOD_ATTACK_POWER_PCT` of 0 is already
parsed onto the stance aura through the shared table (`sim/warrior/stances.go`), which skips and
reports it today because the shared table names no row for it. A shared-table entry for either aura
would start attaching a stat change there instead, changing what the class's own row means.

**Dropping a row renumbers the ones after it.** Each scope's numbers are dense from 1, so a field that
goes away shifts every later number down by one and `TestScopeNumbersAreDense` holds that. Nothing is
live, so no saved payload rides on the old numbers.

**A proto change here breaks whatever a browser holds.** Nothing migrates a saved payload: a field the
manifest retypes makes `fromJson` throw on the enum name an older save spells out. Every `fromJson` of
a settings envelope passes `ignoreUnknownFields`, so a field that only goes away costs nothing.

**Drums are not a Forever consumable.** The client describes no row for the TBC drums - 35476, 35475
and 35478, Battle, War and Restoration, nor their Greater variants - so there is nothing to model and
no manifest row to hang them on. The only "Drums of War" it knows is 1259907, fifteen seconds of party
movement speed, which is no stat buff at all.

**A talent curve only scales the row's first stat.** A row whose talent improves a second amount would
need the generator extended; nothing in the manifest does today.

**A set bonus is not a talent, but it can be a tristate.** The resolver reads `SkillLineAbility`, the
trait trees and the spell effects; it does not read `ItemSetSpell`, so a set that modifies a buff
cannot be a manifest `Talent`. What it can be is the row's `ImpAction`, which is the improved state a
tristate row needs when no trait node prices one, and the icon that state shows. Battlegear of Wrath
is the case: item set 218's `ItemSetSpell` at three pieces (2336) is 23563, an `A_ADD_FLAT_MODIFIER`
of 30 against every effect of the Battle Shout family, which makes the shout worth 169 attack power
rather than 139. The two halves of that are modelled separately. The warrior's own cast reads the
`has_bs_t2` class option - the user's word that this warrior wears the set, not the equipped gear -
and the party's copy is `battle_shout`'s improved state, which `driveBattleShout` reads because the
resolver has no curve to give it. Both call `AddGeneratedFlatBonus`, which raises what the aura
applies and what it bids for its category together, so the stronger of the two copies is the one the
character sheet shows. It is told what the buff is worth without the bonus, because the aura belongs
to the unit rather than to whoever raised it: two warriors in a party wearing the same set ask for
the same total and the second call does nothing. The 30 itself lives in
`buffs.BattleShoutT2Bonus`, with the set and the spell it came from written next to it.

**A party or raid flag means an external caster provides the buff.** The generated apply block builds
the row's `isPlayer=false` copy whenever the proto field is set, so a class port that registers its own
`isPlayer=true` copy has two copies on the character. It either stops setting the flag in
`AddPartyBuffs`/`AddRaidBuffs`, or the row states a `Category` with `SingleAura` and the two copies bid
against each other, so the character sheet shows the buff once. The higher bid deactivates the other
copy; on a tie the incumbent keeps the category when its remaining duration is the longer one, which
is why a druid casting its own Thorns is turned away while the raid's permanent copy is up - both deal
the same 22, so the character strikes back for the same either way. Battle Shout is the worked example:
neither copy is permanent, and the two are worth the same unless one side wears the tier 2 set, so
`TestPlayerBattleShoutTakesTheCategoryOnATie` holds the player's own to the tie and
`TestTheStrongerBattleShoutTakesTheCategory` holds the stronger one to the rest. The rows this decides
are `thorns`, `leader_of_the_pack`, `moonkin_aura` and `trueshot_aura`: `sim/druid/druid.go`,
`sim/druid/feralcat` and `sim/druid/feralbear` raise the party's Leader of the Pack or Moonkin Aura
from a talent, `sim/hunter/hunter.go` raises Trueshot Aura, and the druid's own Thorns waits on the
druid port. `thorns`, `battle_shout`, `leader_of_the_pack` and `moonkin_aura` carry a category, so a
druid registering its own copy of any of them has nothing to decide: the copy joins the same category
and the two bid. The last two share one, `DruidCritAura`, because spell 17007 calls Leader of the
Pack exclusive with Moonkin Aura - a druid's own cast of either joins it, so a party that ticks both
and a druid who casts one are worth the client's 3 and not two threes. Only `trueshot_aura` is left
without a category, so the hunter port has to choose.
