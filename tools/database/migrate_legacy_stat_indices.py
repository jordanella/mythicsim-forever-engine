"""One-time repair of the inherited pre-Resilience-removal snapshot.

Run at the repository root, then go run ./tools/sync_db_binary.
The source hash prevents a second migration or applying this to another snapshot.
Current client stats are retained unless the complete map matches the old
inherited map. Armor uses the pinned Forever planner where it has an entry.
"""
import copy
import hashlib
import json
import re
from pathlib import Path

SOURCE = Path('assets/db_inputs/forever_sim_db.json')
EXPECTED = '2e50e9895bbcb9337bedb1cd235a3b1094babdc8cd8bfc0ceca782f7c17b63ff'


def remap(stats):
    return {str(int(k) - 1 if int(k) > 30 else int(k)): v
            for k, v in stats.items() if int(k) != 30 and v}


def array(values):
    result = [0] * 41
    for k, v in remap(dict(enumerate(values))).items():
        result[int(k)] = v
    return result


def write(path, data):
    # Match the engine's one-record-per-line generated JSON format.
    path.write_text('{\n' + ',\n'.join(json.dumps(k) + ':[' + '\n' +
        ',\n'.join(json.dumps(v, separators=(',', ':'), ensure_ascii=False) for v in rows) + '\n]'
        for k, rows in data.items()) + '\n}\n')


def main():
    raw = SOURCE.read_bytes()
    if hashlib.sha256(raw).hexdigest() != EXPECTED:
        raise SystemExit('Legacy snapshot differs or was already migrated; refusing to remap it.')
    old = json.loads(raw)
    new = copy.deepcopy(old)
    for item in new['items']:
        for opt in item.get('scalingOptions', {}).values():
            if 'stats' in opt: opt['stats'] = remap(opt['stats'])
    for section in ['enchants', 'randomSuffixes']:
        for entry in new[section]: entry['stats'] = array(entry.get('stats', []))
    old_items = {i['id']: i for i in old['items']}
    new_items = {i['id']: i for i in new['items']}
    text = Path('assets/db_inputs/wowhead_forever_gearplanner.txt').read_text()
    planner = json.loads(re.sub(r',\s*}', '}', text[text.index('{'):text.index('});')+1]))
    for name in ['db', 'leftover_db']:
        path = Path(f'assets/database/{name}.json')
        db = json.loads(path.read_text())
        changed = 0
        for item in db['items']:
            before = copy.deepcopy(item)
            prior = old_items.get(item['id'], {})
            fixed = new_items.get(item['id'], {})
            for key, opt in item.get('scalingOptions', {}).items():
                previous = prior.get('scalingOptions', {}).get(key, {}).get('stats', {})
                inherited = fixed.get('scalingOptions', {}).get(key, {}).get('stats', {})
                if opt.get('stats') and opt['stats'] == previous:
                    opt['stats'] = copy.deepcopy(inherited)
                stats = opt.setdefault('stats', {})
                # Client rows can carry new offensive stats but no base armor.
                if not stats.get('30') and inherited.get('30'):
                    if stats.get('31') == previous.get('31'):
                        stats.pop('31', None)
                    stats['30'] = max(0, inherited['30'] - stats.get('31', 0))
                wh = planner.get(str(item['id']), {}).get('stats', {})
                if key == '0' and 'armor' in wh:
                    stats['30'] = wh['armor']
                    stats['31'] = wh.get('armorbonus', 0)
                opt['stats'] = {k:v for k,v in stats.items() if v}
                if not opt['stats']: opt.pop('stats')
            if item != before: changed += 1
        for section in ['enchants', 'randomSuffixes']:
            def identity(x):
                return tuple(x.get(k) for k in (['effectId','itemId','spellId','name'] if section=='enchants' else ['id']))
            prior = {identity(x):x for x in old[section]}
            for entry in db.get(section, []):
                inherited = prior.get(identity(entry))
                if inherited and entry.get('stats') == inherited.get('stats'):
                    entry['stats'] = array(entry['stats'])
        write(path, db)
        print(name, 'changed item records', changed)
    write(SOURCE, new)

if __name__ == '__main__': main()
