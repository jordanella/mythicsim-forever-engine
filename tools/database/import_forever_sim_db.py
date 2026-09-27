"""Writes assets/db_inputs/forever_sim_db.json: the items, enchants and random suffixes of our
Forever sim's database (ElliotWood/Forever master, assets/database/db.json), in this engine's
shapes. gen_db merges it under the client data, so it only fills in what the client does not ship.

    git -C <master checkout> show origin/master:assets/database/db.json > master_db.json
    python tools/database/import_forever_sim_db.py master_db.json assets/db_inputs/forever_sim_db.json

Master stores hit, crit, haste, dodge, parry, block and expertise as percent; this engine takes
rating, at the client's CombatRatings conversion (sim/core/base_stats_auto_gen.go).
"""
import json
import sys
import re
from pathlib import Path

# Read enum identities instead of duplicating their positions. Removing Resilience
# shifted armor, health, mana, MP5 and every resistance in the current engine.
PROTO = (Path(__file__).resolve().parents[2] / "proto/common.proto").read_text()
STAT_IDS = {name: int(value) for name, value in re.findall(r"Stat(\w+)\s*=\s*(\d+);", PROTO.split("enum Stat {")[1].split("}")[0])}
# Legacy Classic stat index -> current stat name and conversion factor.
LEGACY_STATS = {
    0: ("Strength", 1), 1: ("Agility", 1), 2: ("Stamina", 1), 3: ("Intellect", 1), 4: ("Spirit", 1),
    5: ("SpellDamage", 1), 6: ("ArcaneDamage", 1), 7: ("FireDamage", 1), 8: ("FrostDamage", 1),
    9: ("HolyDamage", 1), 10: ("NatureDamage", 1), 11: ("ShadowDamage", 1), 12: ("MP5", 1),
    13: ("SpellHitRating", 10), 14: ("SpellCritRating", 14), 15: ("SpellHasteRating", 10),
    16: ("SpellPiercing", 1), 17: ("AttackPower", 1), 18: ("MeleeHitRating", 10),
    19: ("MeleeCritRating", 14), 20: ("MeleeHasteRating", 10), 21: ("ArmorPenetration", 1),
    22: ("ExpertiseRating", 10), 23: ("Mana", 1), 26: ("Armor", 1), 27: ("RangedAttackPower", 1),
    28: ("DefenseRating", 1), 29: ("BlockRating", 5), 30: ("BlockValue", 1),
    31: ("DodgeRating", 12), 32: ("ParryRating", 15), 34: ("Health", 1),
    35: ("ArcaneResistance", 1), 36: ("FireResistance", 1), 37: ("FrostResistance", 1),
    38: ("NatureResistance", 1), 39: ("ShadowResistance", 1), 40: ("BonusArmor", 1),
    41: ("HealingPower", 1), 42: ("SpellDamage", 1), 43: ("FeralAttackPower", 1),
}
STATS = {old: (STAT_IDS[name], factor) for old, (name, factor) in LEGACY_STATS.items()}
CLASSES = {1: 11, 2: 3, 3: 8, 4: 2, 5: 5, 6: 4, 7: 7, 8: 9, 9: 1}
RANGED_TYPES = {4: 6, 5: 7, 6: 4, 7: 8, 8: 5}
NUM_STATS = max(STAT_IDS.values()) + 1


def convert_stats(values):
    a = list(values) + [0] * (44 - len(values))
    # Forever pays gear hit and crit into both pools and master stores that as the same number
    # in both. This engine sums the two pools (unifyGearHitAndCrit), so count it once.
    for melee, spell in ((18, 13), (19, 14)):
        if a[melee] and a[melee] == a[spell]:
            a[spell] = 0
    out = {}
    for i, v in enumerate(a):
        if v and i != 33:  # Resilience no longer exists in Forever.
            j, per = STATS[i]
            out[j] = out.get(j, 0) + round(v * per, 4)
    return out


def as_array(stats):
    a = [0] * NUM_STATS
    for k, v in stats.items():
        a[k] = v
    return a


def item(m):
    scaling = {'ilvl': m.get('ilvl', 0)}
    stats = convert_stats(m.get('stats', []))
    if m.get('bonusPhysicalDamage'):
        stats[STAT_IDS['PhysicalDamage']] = m['bonusPhysicalDamage']
    if stats:
        scaling['stats'] = {str(k): v for k, v in sorted(stats.items())}
    if m.get('weaponDamageMin'):
        scaling['weaponDamageMin'] = m['weaponDamageMin']
        scaling['weaponDamageMax'] = m['weaponDamageMax']
    out = {k: m[k] for k in ('id', 'name', 'icon', 'type', 'armorType', 'weaponType', 'handType', 'weaponSpeed',
                             'phase', 'quality', 'unique', 'setName', 'setId', 'expansion', 'factionRestriction',
                             'randomSuffixOptions', 'requiredProfession') if m.get(k)}
    if m.get('rangedWeaponType'):
        out['rangedWeaponType'] = RANGED_TYPES.get(m['rangedWeaponType'], m['rangedWeaponType'])
    if m.get('classAllowlist'):
        out['classAllowlist'] = [CLASSES[c] for c in m['classAllowlist']]
    # Rep sources name factions by a different enum here; the rest share one shape.
    sources = [s for s in m.get('sources', []) if 'rep' not in s]
    if sources:
        out['sources'] = sources
    out['scalingOptions'] = {'0': scaling}
    return out


def enchant(m):
    out = {k: v for k, v in m.items() if k not in ('stats', 'classAllowlist')}
    out['stats'] = as_array(convert_stats(m.get('stats', [])))
    if m.get('classAllowlist'):
        out['classAllowlist'] = [CLASSES[c] for c in m['classAllowlist']]
    return out


def main(src, dst):
    master = json.load(open(src, encoding='utf-8'))
    out = {
        'items': [item(m) for m in master['items']],
        'enchants': [enchant(m) for m in master['enchants']],
        # Forever's random suffixes are flat enchantments, like Classic's (see ItemEquipmentBaseStats).
        'randomSuffixes': [{'id': r['id'], 'name': r['name'], 'stats': as_array(convert_stats(r['stats']))}
                           for r in master['randomSuffixes']],
        'zones': master['zones'],
        'npcs': master['npcs'],
    }
    with open(dst, 'w', encoding='utf-8', newline='\n') as f:
        f.write('{\n')
        for i, key in enumerate(out):
            f.write(f'"{key}":[\n')
            f.write(',\n'.join(json.dumps(x, separators=(',', ':'), ensure_ascii=False) for x in out[key]))
            f.write('\n]' + (',' if i < len(out) - 1 else '') + '\n')
        f.write('}\n')
    print({k: len(v) for k, v in out.items()})


if __name__ == '__main__':
    main(sys.argv[1], sys.argv[2])
