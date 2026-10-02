# Druid beta pass (17 September 2026)

Beta client `1.60.1.69893` against Classic Era `1.15.9.69722`, read with `tools/data_watch/spell_client.py`, the
talent curves with `tools/data_watch/trait_curve.mjs`, and `../beta/druid.json`.

## How the numbers were read

A rank's damage is the client's base plus `EffectRealPointsPerLevel` for every level from the rank's level up to its
max level (capped at 60), rounded down for the low end and up for the high end, the rule the mage pass used. Against the
Era client it reproduces every Wrath, Starfire and Moonfire rank the sim carried except Moonfire rank 4 (sim 44-53, Era
47-55), which now follows the rule too. Dot totals are the client's tick times the tick count; the sim's per tick
coefficients are the client's per tick values, so nothing needed converting.

Two splits the client and the sim make differently:
- **Hurricane** is an area trigger in Forever that casts a separate damage spell each second (1278965, 1278968,
  1278759). The sim keeps the dot on the parent spell and its tick damage is that spell's.
- **Lacerate**: the sim's bleed carries 414647, which in the client is the weapon damage half. The numbers are the
  client's; only the ids are swapped.

## Spells changed (old -> new, level 60 rank unless noted)

| Spell | Change |
|---|---|
| Wrath (8 ranks) | rank 8 248-277 -> 62-69, about a quarter at every rank from 3 up; mana 180 -> 120 (every rank about a third cheaper); coefficients ranks 1-3 .123/.231/.443 -> .429/.486/.571 |
| Starfire (7 ranks) | rank 7 496-584 -> 350-412, about 30% less at every rank |
| Moonfire (10 ranks) | rank 10 195-228 plus 384 over 12 sec -> 128-151 plus 240; every rank now .15 direct and .13 a tick (rank 1 was .06/.052) |
| Insect Swarm (5 ranks) | rank 5 324 -> 186 over 12 sec |
| Hurricane (3 ranks) | tick 134 -> 132 (70 -> 68, 100 -> 98 at ranks 1-2); **the 1 min cooldown is gone**; slow 25% -> 20% (not modelled) |
| Shred (4 ranks) | 225% -> 155% weapon damage, same flat bonus. The client's text still prints Classic's "plus 180" |
| Claw | gained a 110% weapon damage effect (was 100%) |
| Rake (4 ranks) | rank 4 58 plus 32 a tick -> 61 plus 34; every rank slightly up |
| Rip (6 ranks) | rank 6 17 + 28 per combo point a tick -> 15 + 25.5; per point lower at every rank, base lower at ranks 4-6 |
| Mangle (Bear) | 15 -> 20 Rage; flat bonus 26 -> 26 / 38 / 59 / 77 at levels 25 / 40 / 50 / 60 (ranks 407995, 1238069, 1238070, 1238073) |
| Lacerate | 10 -> 15 Rage; hit 20% -> 10% weapon damage per stack; bleed 29.8 -> 15 a tick per stack at 58+ (10 at 42, 12 at 50) |
| Enrage | adds 10 Rage up front under Forever |
| Frenzied Regeneration | one rank (22842): 1% of maximum health per Rage instead of 20 health |
| Barkskin | 20% of all damage for 12 sec -> 20% of Physical damage for 15 sec (the sim had not matched Era's 15 sec either) |
| Demoralizing Roar | 193 -> 204 attack power at 60 (client rank 5 is 193 plus 1.4 a level) |

Unchanged in the client and left alone: Ferocious Bite, Maul, Swipe (3 targets), Faerie Fire, Innervate, Tiger's Fury's
Forever shape, the form passives (Dire Bear 180 AP / 1240 health / 360% armor, Cat Form 120 AP at 60), Moonkin Form's
cost, Nature's Grace (10% haste and GCD for 3 sec), every cast time.

`ui/core/spells/druid.json`: the spells above are `forever` with tooltips; Ferocious Bite, Maul and Swipe are `classic`.
Still `assumed`: Mangle (Cat), Faerie Fire (Feral) (neither exists in Forever, see below), Mangle (Bear) and Lacerate
(threat only) and Dire Bear Form (threat only). The nine `unreviewed` entries left (40, 46, 50, 56, 60, 66, 880, 1180,
1495) are not spells: the id walk in `sim/spell_sources_test.go` reads every integer in Hurricane's anonymous rank
table (levels, mana costs) as a spell id. Hurricane's damage values, which it also read, are now floats, and their three
entries are gone.

## DPS (Average-Default, before -> after)

| test | before | after |
|---|---|---|
| TestForeverMoonkin | 400.9 | 290.4 |
| TestP1Balance | 440.0 | 294.4 |
| TestP1Feral | 452.2 | 381.2 |
| TestP1FeralTank | 740.8 dps, 2256.7 tps | 591.5 dps, 1585.3 tps |

## Checklist lines

### Druid section
- **Resolved** `berserk.go:16`, Berserk cooldown: 417141 has a 3 min cooldown, 15 sec, no cost. No change.
- **Resolved** `demoralizing_roar.go:24`, Feral Aggression baseline: the client's roar is stronger than Classic with
  5/5 Feral Aggression, 193 -> 204 at 60 (see table).
- **Resolved** `druid.go:122`, Improved Mark of the Wild baseline: Mark and Gift of the Wild are 385 armor, 16 stats,
  27 resistances in the client, Classic's 285 / 12 / 20 plus 35%. `sim/core/buffs.go` floors the 35% and gives 384 armor
  (out of scope, reported).
- **Resolved, not removed** `faerie_fire.go:26`, Faerie Fire (Feral) baseline: it is **gone**, not baseline. 16857 and
  17390-17392 are absent from the Forever spellbook and 17392 from the spell tables. The sim still registers it because
  the cat rotation (`sim/druid/feral/rotation*.go`) and both feral APLs are built around it; removing it is a rotation
  change, left as a new TODO.
- **Resolved** `forms.go:223`, Furor's Cat half: curve 20-100, and the text derives all three numbers from it (20% of
  the Energy, 2 a second, 20 max per point), as the sim had.
- **Resolved** `forms.go:342`, Furor's Bear half: 20% per point, 17057 grants 10 Rage, as the sim had.
- **Resolved** `lacerate.go:16`, Lacerate: trained at 42/50/58 in Forever, numbers now the client's (see table). Threat
  stays **open**: the client tables do not carry it.
- **Resolved, not removed** `mangle.go:17`, Mangle (Cat) Energy cost: there is no cat Mangle in Forever. The talent
  teaches the Bear Mangle only, and Season of Discovery's 407993 is gone. Kept for the cat rotation, left as a new TODO.
- **Resolved** `mangle.go:71`, Mangle (Bear): 20 Rage (was 15), 6 sec cooldown confirmed, damage by rank (see table).
  Threat stays **open**, not in the client tables.
- **Resolved** `talents.go:301`, Eclipse: curve 0.17 / 0.33 / 0.5 sec, so rank 3 is 0.5 sec, not 0.51. Fixed.
- **Resolved** `talents.go:404`, Primal Fury: both curves 50 / 100%, 5 Rage (16959). No change.
- **Resolved** `talents.go:453`, Natural Reaction dodge: curve 1-5%. No change.
- **Resolved** `talents.go:456`, Natural Reaction Rage: curve 20-100%, 5 Rage (417053). No change.
- **Resolved** `talents.go:522`, Naturalist: curve 1-5%. No change.
- **Superseded** `tigers_fury.go`: client 1.60.1.70170 removed Tiger's Fury; `shifting_power.go` replaces it (patch 41
  in `docs/mythicsim-patches.md`). Until then Forever had one Tiger's Fury (5217, learned at 24), 15% for 6 sec, 30 sec
  cooldown, no cost; 6793, 9845 and 9846 no longer exist.
- **Resolved** `wrath.go:53`, Improved Wrath: curves 10-50% cost and 0.1-0.5 sec cast. No change.

### Talents the sim does not read
- **Open, stays not simulated** Overgrowth: extra Entangling Roots targets, no raid damage.
- **Open, stays not simulated** Gift of the Earthmother: 0.5 sec off the GCD of Rejuvenation, Swiftmend and Wild
  Growth; healing only, and the sim has no restoration spec.
- **Open, stays not simulated** Wild Growth: a party heal (408120 / 1238214 / 1238215, 97 a tick at rank 3, 1050 mana,
  6 sec cooldown); healing only.

### Baseline ability changes
No druid lines; the spellbook diff below is the answer to "diff each class spellbook against Classic Era".

## Spellbook, Forever against Era (`--learned druid`)

The Era list includes Season of Discovery's rune spells, so some "removals" are just SoD.

New in Forever:
- Mangle ranks 2-4 (1238069, 1238070, 1238073) and Lacerate ranks 2-3 (1235826, 1235827): implemented as above.
- **Feral Charge** 1238122 (Bear, 5 Rage, 15 sec cooldown, root and interrupt): no damage, not implemented.
- Wild Growth ranks 2-3, Revive ranks 2-5: healing and resurrection, not implemented.
- Feral Combat 1306742 (a passive with no data), Journeyman Riding, Equip Transmog Outfit.

Gone in Forever:
- **Faerie Fire (Feral)** all ranks, **Mangle (Cat)** 407993, **Savage Roar**, **Thrash (Cat)**. The last two were
  Season of Discovery runes the sim never modelled.
- Tiger's Fury ranks 2-4 and Frenzied Regeneration ranks 2-3: collapsed to one spell each.
- Gift of Nature and Tranquil Spirit per rank spells (talents moved to trait curves), Barkskin Effect (DND), Command
  (racial), and the SoD tuning passives.

## Needs a sim/core change (not made)
- `core.DemoralizingRoarAura` builds from Classic's 138 and Feral Aggression points; the druid now asks for 6 points to
  get the client's 204. A Forever value in core would be cleaner.
- `sim/core/buffs.go` improved Gift of the Wild floors to 384 armor; the client says 385.
