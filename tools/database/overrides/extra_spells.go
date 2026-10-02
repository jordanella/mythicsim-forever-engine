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
