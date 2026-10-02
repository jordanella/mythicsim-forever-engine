package overrides

// A spell the store carries although no class, item, enchant or set bonus reaches it. Reason and
// Source weigh the same as an Override's, and the generator refuses an entry without a reason.
type Extra struct {
	SpellID int32
	Reason  string
	Source  string
}

// The spells the generator adds to the store's roots by hand.
var ExtraSpells = []Extra{}

// One-rank talent nodes whose spell is an ability the talent grants, which the generator would
// otherwise leave off the class file: a one-rank node on a spell that is not passive is usually an
// ability the game teaches (Hemorrhage, Water Shield) and is filed under the skill line's own ranks.
// Listing the spell here puts the ability on a single-rank ladder of its own, through the skill line
// row that grants it (AcquireMethod 3), the way Cat Form is. Reason and Source weigh the same as an
// Override's.
var TalentGrantedAbilities = []Extra{
	{
		SpellID: 1322605,
		Reason: "Shifting Power is the Feral talent of client 1.60.1.70170 that converts Mana into Energy. Its node " +
			"is one rank on an ability nothing trains, only the talent grants it, so the sim reads it as a " +
			"single-rank family like Cat Form instead of through the talent curve.",
		Source: "SpellEffect 1358458 (energize, 40 Energy), SpellPower 314986 (55% of base mana), SpellCooldowns 101953 (16 s, 1 s global cooldown), SkillLineAbility 59569 (granted)",
	},
}

// INTERIM, drop when the database is regenerated with the client hotfix cache. A regeneration from the
// CDN tables alone (db2tool --cdn without --dbcache, run for build 1.60.1.70170 on 2026-10-02) has
// no ItemSparse rows for the items the live client adds through DBCache.bin, so the gear filter that
// roots the store at every item effect misses the spells below. Every one is still the effect of an
// item in assets/database/db.json that the committed item procs register, and registering a proc whose
// spell is not in the store panics. They are listed here so the store keeps them until the real hotfix
// overlay supplies the items again; with the overlay they are roots on their own and this list
// is redundant.
var interimHotfixItemSpells = []int32{
	1133, 430432, 459593, 459594, 459595, 459596, 459598, 459599, 459600, 459601, 459602, 459603, 459604,
	459605, 459606, 459607, 459608, 460339, 463001, 1213390, 1213395, 1213398, 1213405, 1213407, 1214155,
	1215404, 1216968, 1216997, 1222994, 1222997, 1282503, 1291551, 1291568, 1291748, 1291749, 1291758, 1291782,
	1292039, 1292222, 1292252, 1292268, 1292560, 1292575, 1292581, 1292594, 1292670, 1292674, 1292679, 1292683,
	1293183, 1293306, 1293331, 1293701, 1300128, 1306515, 1306572, 1306578, 1306580, 1306583, 1309315, 1309369,
	1312176, 1314011, 1314040, 1314305, 1314412, 1315339, 1315767, 1315778, 1316039, 1316865, 1316928, 1318250,
	1319047, 1320498, 1320579, 1321572,
}

func init() {
	for _, id := range interimHotfixItemSpells {
		ExtraSpells = append(ExtraSpells, Extra{
			SpellID: id,
			Reason:  "An item in db.json casts it, and its ItemSparse row only arrives with the client hotfix cache, which the 70170 regeneration did not have.",
			Source:  "Root of the committed 1.60.1.70124 store (ItemRoots in assets/db_inputs/spell_store_inputs.json at c30f5d2dd5).",
		})
	}
}
