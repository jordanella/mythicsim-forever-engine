# Frostfire Bolt support

## Evidence that the spell belongs to Forever

The committed client store is generated from Blizzard client 1.60.1.70009. Its Mage
ladder contains baseline Frostfire Bolt ranks 401502 (level 40), 1237312 (level 50)
and 1237313 (level 60). The prior client spellbook audit in `docs/beta-pass/mage.md`
independently records ranks 2 and 3 as new learned spells in Forever. Rank 1 also
existed in Season of Discovery, but Forever has its own learned ranks and values.

Public corroboration: [Forever rank 3](https://www.wowhead.com/forever/spell=1237313/frostfire-bolt)
and [rank 2](https://www.wowhead.com/forever/spell=1237312/frostfire-bolt). This is
client/database verification, not a recording of a character casting it on the server.

Reproduce the exact source values with `go run ./tools/spelldata -family mage/FrostfireBolt`.
Rank 3 has a 3-second cast, 370 mana cost, 35-yard range, missile speed 24, Fire and
Frost school flags, a 40% slow, and 3 ticks of 19 over 9 seconds. The direct coefficient
is 0.814; the periodic coefficient is zero. The client marks periodic damage as able
to crit. Damage reads the client store's level-60 average, consistent with the other
Mage spells. Wowhead's displayed direct-damage range differs from that level-specific
store output, so its display is not used as a numeric input to the simulation.

## Implementation

Register every rank without a talent gate. Apply the periodic effect only after the
missile lands. Include the spell in the Mage Fire, Frost, damaging and chill masks,
Improved Fireball's cast-time modifier, Hot Streak and Missile Barrage. School-based
mods supply Critical Mass, Fire Power, Piercing Ice, Ice Shards, Frost Channeling,
Improved Scorch and Combustion. Direct crits trigger Ignite and Master of Elements;
periodic crits do not trigger the direct-hit talents.

For Frostfire only, the core checks the lower resistance and takes the larger school
spell-power, school hit and damage multiplier, so shared school effects are applied
once. Separate Fire and Frost talent mods can both apply. Other mixed schools retain
their existing behavior.

The optional `Frostfire hybrid (experimental)` APL uses Mana Ruby, cooldowns,
Evocation below 10% mana, Pyroblast at three Hot Streak stacks and otherwise the
highest Frostfire Bolt rank. A known-aura guard prevents hard-cast Pyroblast spam
when Hot Streak is not talented. It does not add Scorch, Fire Blast or Ice Lance to
the screenshot's proposed damage priority. Existing Mage defaults are unchanged.

## Remaining server validation

Binary resistance follows Frostbolt and the [original WoWSims SoD Frostfire Bolt](https://github.com/wowsims/sod/blob/master/sim/mage/frostfire_bolt.go).
The lower-resistance choice is explicit in the Forever tooltip. Its all-or-nothing
resistance behavior and the strongest-school bonus rule still need Forever combat-log
confirmation. The patch retains the existing engine assumptions for Fingers of Frost
arriving during a cast and Missile Barrage's proc chance. These limits are why the
new rotation is marked experimental. PvP slow duration, Permafrost and Frostbite are
outside this raid-boss implementation, consistent with the existing Frost spells.

## Validation

`go test --tags=with_db` passed for every `sim` package except the web server, which
requires a built website, plus `cmd/wowsimcli/...`. No existing golden changed.
Focused tests cover every rank, timing, periodic coefficient, mana cost/refund,
crit damage, direct versus periodic procs, Fingers charges, Missile Barrage, both
orders of unequal resistances, and avoiding doubled hit, power or damage bonuses.

The Frostfire patch is applied on engine `dfe1697f6a`, based on upstream
`8dc19a4241`. The existing seven patches, corrected item data, and newly merged
weapon and enchant procs, and Omen of Clarity corrections are preserved unchanged.
The application integration records the exact-image hybrid and reference-build runs.
