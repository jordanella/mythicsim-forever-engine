import { getLang } from '@i18n/locale_service';

import { CHARACTER_LEVEL } from '../constants/mechanics';
import { Database } from './database';

export type WowheadTooltipItemParams = {
	/**
	 * @description Item ID
	 * @see item - mapped value from wowhead
	 * */
	itemId: number;
	/**
	 * @description Item level
	 * @see ilvl - mapped value from wowhead
	 * */
	itemLevel?: number;
	/**
	 * @description Level
	 * @see lvl - mapped value from wowhead
	 * */
	level?: number;
	/**
	 * @description Enchant
	 * @see ench - mapped value from wowhead
	 * */
	enchantIds?: number[];
	/**
	 * @description Gems
	 * @see gems - mapped value from wowhead
	 * */
	gemIds?: number[];
	/**
	 * @description Extra Socket
	 * @see sock - mapped value from wowhead
	 * */
	hasExtraSocket?: boolean;
	/**
	 * @description Item Set Pieces
	 * @see pcs - mapped value from wowhead
	 * */
	setPieceIds?: number[];
	/**
	 * @description Random Enchantment
	 * @see rand - mapped value from wowhead
	 * */
	randomEnchantmentId?: number;
	/**
	 * @description Reforges
	 * @see forg - mapped value from wowhead
	 * */
	reforgeId?: number;
	/**
	 * @description Upgrades
	 * @see upgd - mapped value from wowhead
	 * */
	upgradeStep?: number;
	/**
	 * @description Transmogrified to
	 * @see transmog - mapped value from wowhead
	 * */
	transmogId?: number;
};

export type WowheadTooltipSpellParams = {
	/**
	 * @description Spell ID
	 * @see spell - mapped value from wowhead
	 * */
	spellId: number;
	/**
	 * @description Level
	 * @see lvl - mapped value from wowhead
	 * */
	level?: number;
	/**
	 * @description Buff
	 * @see buff - mapped value from wowhead
	 * */
	useBuffAura?: boolean;
	/**
	 * @description Difficulty
	 * @see dd - mapped value from wowhead
	 * */
	difficultyId?: 14 | 15 | 16;
};

// Wowhead serves each expansion under its own url segment and its own `dataEnv`
// id. Porting this file to a sibling sim repo means changing the one line below:
// the domain, and every url built from it, follows. The literal-union key means
// an unmapped id is a compile error rather than a `.../undefined/...` url.
const WOWHEAD_EXPANSIONS = {
	4: 'classic',
	5: 'tbc',
	15: 'mop-classic',
	16: 'forever',
} as const;
type WowheadExpansionEnv = keyof typeof WOWHEAD_EXPANSIONS;

export const WOWHEAD_EXPANSION_ENV: WowheadExpansionEnv = 16;
export const WOWHEAD_DOMAIN = WOWHEAD_EXPANSIONS[WOWHEAD_EXPANSION_ENV];

// Wowhead's Forever data only has the items and spells it has seen change: Classic items
// like Truestrike Shoulders 404 there. Master asks Classic Era (env 4) for everything, which
// 404s on what Forever added (items past 25000, spells past 100000, a few reused Classic ids).
// So Classic Era for what Classic had, Forever for the rest. Checked 2026-09-23 against every
// tooltip the 19 spec pages ask for on load: the only 404s left are TBC ids neither one has.
// Classic ids Forever reused or renamed: Classic Era shows the old name and tooltip, or 404s.
// Every id under 100000 the sim or talent trees use whose Forever client name (1.60.1.70009)
// differs from Classic Era's (1.15.9.69722); Wowhead Forever serves each under its Forever name.
const FOREVER_ONLY_SPELLS = new Set([
	87, // Windwalk (not in Classic Era)
	132, // Detect Invisibility (Classic Era: Detect Lesser Invisibility)
	424, // Earthquake (not in Classic Era)
	603, // Bane of Doom (Classic Era: Curse of Doom)
	// Holy Strike (not in Classic Era)
	678,
	679,
	680,
	1866,
	2495,
	5569,
	10332,
	10333,
	// Bane of Agony (Classic Era: Curse of Agony)
	980,
	1014,
	6217,
	11711,
	11712,
	11713,
	11078, // Wake of Fire (Classic Era: Improved Fire Blast)
	11237, // Improved Channeling (Classic Era: Improved Arcane Missiles)
	11242, // Arcane Impact (Classic Era: Improved Arcane Explosion)
	11247, // Arcane Geometry (Classic Era: Magic Attunement)
	11252, // Arcane Shielding (Classic Era: Improved Mana Shield)
	11743, // Detect Invisibility (Classic Era: Detect Greater Invisibility)
	12281, // Weaponmaster (Classic Era: Sword Specialization)
	12295, // Improved Tactical Mastery (Classic Era: Tactical Mastery)
	13960, // Hack and Slash (Classic Era: Sword Specialization)
	14084, // Improved Distract (not in Classic Era)
	14913, // Twilight Focus (Classic Era: Healing Focus)
	15258, // Shadow Weaving (Classic Era: Shadow Vulnerability)
	16086, // Improved Fire Nova (Classic Era: Improved Fire Totems)
	16268, // Spirit Weapons (Classic Era: Parry)
	16538, // Bastion (Classic Era: One-Handed Weapon Specialization)
	16578, // Elemental Alacrity (Classic Era: Lightning Mastery)
	// Blood Frenzy (Classic Era: Primal Fury)
	16958,
	16959,
	16966, // Shredding Attacks (Classic Era: Improved Shred)
	17002, // Feral Swiftness (Classic Era: Feline Swiftness)
	17069, // Naturalist (Classic Era: Improved Healing Touch)
	17245, // Overgrowth (Classic Era: Improved Nature's Grasp)
	17804, // Soul Siphon (Classic Era: Improved Drain Life)
	17927, // Agonizing Flames (Classic Era: Improved Searing Pain)
	18425, // Silenced - Kick (Classic Era: Kick - Silenced)
	18459, // Incineration (Classic Era: Incinerate)
	18498, // Silenced (Classic Era: Shield Bash - Silenced)
	18662, // Bane of Doom Effect (Classic Era: Curse of Doom Effect)
	18731, // Fel Vitality (Classic Era: Fel Intellect)
	18789, // Burning Shadow (Classic Era: Burning Wish)
	18791, // Touch of Fire (Classic Era: Touch of Shadow)
	18827, // Improved Bane of Agony (Classic Era: Improved Curse of Agony)
	19337, // Chastise (Classic Era: Fear Ward)
	19376, // Survival Tactics (Classic Era: Trap Mastery)
	19426, // Lethal Attacks (Classic Era: Lethal Shots)
	19552, // Deadly Aspects (Classic Era: Improved Aspect of the Hawk)
	// Seal of Fury (not in Classic Era)
	20163,
	20231,
	20415,
	20416,
	20417,
	20418,
	20419,
	20421,
	20422,
	20423,
	// Judgement of Fury (not in Classic Era)
	20183,
	20232,
	20411,
	20412,
	20413,
	20414,
	20224, // Improved Seals (Classic Era: Improved Seal of Righteousness)
	23602, // Master of Defense (Classic Era: Shield Specialization, so its rage row read as a second Shield Specialization)
	// Lacerate (not in Classic Era)
	24118,
	24119,
	24120,
	24293, // Improved Tracking (Classic Era: Monster Slaying)
	// Demoralizing Screech (Classic Era: Screech)
	24423,
	24424,
	24577,
	24578,
	24579,
	24580,
	24581,
	24582,
	28999, // Elemental Reach (Classic Era: Storm Reach)
	29187, // Natural Grace (Classic Era: Healing Grace)
	36936, // Totemic Recall (not in Classic Era)
	66842, // Call of the Elements (not in Classic Era)
	66843, // Call of the Ancestors (not in Classic Era)
	66844, // Call of the Spirits (not in Classic Era)
]);
const wowheadEnvFor = (entity: WowheadEntity, id: number): WowheadExpansionEnv => {
	if (entity === 'item') return id < 25000 ? 4 : WOWHEAD_EXPANSION_ENV;
	if (entity === 'spell') return id < 100000 && !FOREVER_ONLY_SPELLS.has(id) ? 4 : WOWHEAD_EXPANSION_ENV;
	return WOWHEAD_EXPANSION_ENV;
};
export const wowheadTooltipDomain = (entity: WowheadEntity, id: number) => {
	const env = wowheadEnvFor(entity, id);
	return { env, domain: WOWHEAD_EXPANSIONS[env] };
};

export const buildWowheadTooltipDataset = async (options: WowheadTooltipItemParams | WowheadTooltipSpellParams) => {
	const lang = getLang();
	const params = new URLSearchParams();
	const langPrefix = lang && lang != 'en' ? lang + '.' : '';
	const { env, domain } = 'spellId' in options ? wowheadTooltipDomain('spell', options.spellId) : wowheadTooltipDomain('item', options.itemId);
	params.set('domain', `${langPrefix}${domain}`);
	params.set('dataEnv', String(env));

	params.set('lvl', String(options.level || CHARACTER_LEVEL));

	if ('spellId' in options) {
		if (options.spellId) {
			params.set('spell', String(options.spellId));
		}
		if (options.useBuffAura) {
			const data = await Database.getSpellIconData(options.spellId);
			if (data.hasBuff) params.set('buff', '1');
		}
	}

	if ('itemId' in options) {
		params.set('item', String(options.itemId));
		if (options.itemLevel) {
			params.set('ilvl', String(options.itemLevel));
		}
		if (options.gemIds?.length) {
			params.set('gems', options.gemIds.join(':'));
		}
		if (options.enchantIds) {
			params.set('ench', options.enchantIds.join(':'));
		}
		if (options.reforgeId) {
			params.set('forg', String(options.reforgeId));
		}
		if (options.randomEnchantmentId) {
			params.set('rand', String(options.randomEnchantmentId));
		}
		if (typeof options.upgradeStep === 'number') {
			params.set('upgd', String(options.upgradeStep));
		}
		if (options.setPieceIds?.length) {
			params.set('pcs', options.setPieceIds.join(':'));
		}
		if (options.hasExtraSocket) {
			params.set('sock', '');
		}
		if (options.transmogId) {
			params.set('transmog', String(options.transmogId));
		}
	}

	return decodeURIComponent(params.toString());
};

export function getWowheadLanguagePrefix(): string {
	const lang = getLang();
	return lang === 'en' ? '' : `${lang}/`;
}

// Every wowhead link this app builds hangs off one of these. The entity links
// keep the bare host they have always used (domain per id, see wowheadTooltipDomain); the gear planner page is the `www.`
// form we show to users verbatim in the importer.
export const WOWHEAD_GEAR_PLANNER_URL = `https://www.wowhead.com/${WOWHEAD_DOMAIN}/gear-planner`;
export const WOWHEAD_ICON_BASE_URL = 'https://wow.zamimg.com/images/wow/icons';

type WowheadEntity = 'item' | 'spell' | 'quest' | 'npc' | 'zone';

// `https://wowhead.com/<domain>/<lang>/<entity>=<id>` — the language segment
// is empty for English.
export function wowheadEntityUrl(entity: WowheadEntity, id: number, rank = 0, definitionId = 0): string {
	const url = `https://wowhead.com/${wowheadTooltipDomain(entity, id).domain}/${getWowheadLanguagePrefix()}${entity}=${id}`;
	const params = new URLSearchParams();
	if (definitionId > 0) params.set('def', String(definitionId));
	if (rank > 0) params.set('rank', String(rank));
	const query = params.toString();
	return query ? `${url}?${query}` : url;
}

export function wowheadIconUrl(iconLabel: string, size: 'large' | 'medium' | 'small' = 'large'): string {
	return `${WOWHEAD_ICON_BASE_URL}/${size}/${iconLabel}.jpg`;
}
