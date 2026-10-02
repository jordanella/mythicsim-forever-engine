import * as PresetUtils from '@app/preset_utils';
import { Debuffs, PartyBuffs, RaidBuffs } from '@generated/proto/buffs';
import { ConsumesSpec, Profession, Race } from '@generated/proto/common';
import { SavedTalents } from '@generated/proto/ui';
import { ProtectionWarrior_Options as ProtectionWarriorOptions, WarriorStance } from '@generated/proto/warrior';
import { OtherDefaults as SimUIOtherDefaults } from '@sim/spec_config';

import * as WarriorPresets from '../shared/presets';
import ForeverNoReckApl from './apls/dps_no_reck.apl.json';
import ForeverReckApl from './apls/dps_reck.apl.json';
import ForeverProtectionApl from './apls/protection.apl.json';
import LaunchGear from './gear_sets/launch.gear.json';
import P0BisGear from './gear_sets/p0.bis.gear.json';
import P1BisGear from './gear_sets/p1.bis.gear.json';

// Preset options for this spec.
// Eventually we will import these values for the raid sim too, so its good to
// keep them in a separate file.

// Master's rotation presets, in master's order; Protection is master's default.
export const ROTATION_PRESET_PROTECTION = PresetUtils.makePresetAPLRotation('Protection', ForeverProtectionApl);
export const ROTATION_PRESET_NO_RECK = PresetUtils.makePresetAPLRotation('DPS (No Reck)', ForeverNoReckApl);
export const ROTATION_PRESET_RECK = PresetUtils.makePresetAPLRotation('DPS (With Reck)', ForeverReckApl);

// The two builds our Forever sim ships for the tank.
export const ProtectionTalents = PresetUtils.makePresetTalents('Protection', SavedTalents.create({ talentsString: '31--255533123330201351' }));
export const DeepProtectionTalents = PresetUtils.makePresetTalents('Protection 1/0/50', SavedTalents.create({ talentsString: '1--255533123331221351' }));

export const DefaultOptions = ProtectionWarriorOptions.create({
	classOptions: {
		queueDelay: 250,
		startingRage: 0,
		useBattleShout: true,
		defaultStance: WarriorStance.WarriorStanceDefensive,
	},
});

// Master's tank set: Flask of the Titans and Elixir of Superior Defense, no sapper or stone.
export const DefaultConsumables = ConsumesSpec.create({
	flaskId: 13510, // Flask of the Titans
	battleElixirId: 13452, // Elixir of the Mongoose
	guardianElixirId: 3825, // Elixir of Lesser Fortitude
	defenseElixirId: 13445, // Elixir of Greater Defense
	strengthBuffId: 12451, // Juju Power
	attackPowerBuffId: 12460, // Juju Might
	zanzaId: 8410, // R.O.I.D.S.
	alcoholId: 21151, // Rumsey Rum Black Label
	dragonbreathChili: true,
	foodId: 20452, // Smoked Desert Dumplings
	potId: 13442, // Mighty Rage Potion
});

export const DefaultRaidBuffs = RaidBuffs.create({
	giftOfTheWild: true,
	prayerOfFortitude: true,
	fireResistanceAura: true,
});

export const DefaultPartyBuffs = PartyBuffs.create({
	...WarriorPresets.DefaultPartyBuffs,
	devotionAura: true,
});

export const DefaultDebuffs = Debuffs.create({
	...WarriorPresets.DefaultDebuffs,
	insectSwarm: true,
});

export const OtherDefaults: Partial<SimUIOtherDefaults> = {
	reactionTime: 200, // master's default
	profession1: Profession.Blacksmithing,
	profession2: Profession.Enchanting,
	race: Race.RaceDwarf,
};

// Our Forever sim's gear presets (master ui/<spec>/gear_sets).
export const GEAR_LAUNCH = PresetUtils.makePresetGear('Launch', LaunchGear);
export const GEAR_P0_BIS = PresetUtils.makePresetGear('Pre-BiS', P0BisGear);
export const GEAR_P1_BIS = PresetUtils.makePresetGear('P1 BiS', P1BisGear);
export const DEFAULT_GEAR = GEAR_LAUNCH;
export const BUILD_TANKY = PresetUtils.makePresetBuild('Tanky', { gear: DEFAULT_GEAR, talents: ProtectionTalents, rotation: ROTATION_PRESET_PROTECTION });
export const GEAR_PRESETS = [GEAR_LAUNCH, GEAR_P0_BIS, GEAR_P1_BIS];
