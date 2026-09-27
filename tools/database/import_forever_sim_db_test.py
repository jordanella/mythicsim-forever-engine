import unittest
from import_forever_sim_db import convert_stats, as_array, item, STAT_IDS

class LegacyStatsTest(unittest.TestCase):
    def test_removed_resilience_does_not_shift_surviving_stats(self):
        values = [0] * 44
        for index, value in {12: 6, 23: 100, 26: 565, 33: 20, 34: 120, 35: 10, 39: 15, 40: 50}.items():
            values[index] = value
        actual = convert_stats(values)
        expected = {STAT_IDS[name]: value for name, value in {
            'MP5': 6, 'Mana': 100, 'Armor': 565, 'Health': 120,
            'ArcaneResistance': 10, 'ShadowResistance': 15, 'BonusArmor': 50,
        }.items()}
        self.assertEqual(actual, expected)
        self.assertEqual(len(as_array(actual)), 41)

    def test_physical_damage_uses_current_enum(self):
        result = item({'id': 1, 'bonusPhysicalDamage': 3})
        self.assertEqual(result['scalingOptions']['0']['stats'], {str(STAT_IDS['PhysicalDamage']): 3})

if __name__ == '__main__': unittest.main()
