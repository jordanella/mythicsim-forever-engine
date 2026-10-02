import * as PresetUtils from '@app/preset_utils';
import { ConsumesSpec, Profession, Race, Spec } from '@generated/proto/common';
import {
	FeralBearDruid_Options as DruidOptions,
	FeralBearDruid_Rotation as DruidRotation,
	FeralBearDruid_Rotation_SwipeUsage as SwipeUsage,
} from '@generated/proto/druid';
import { SavedTalents } from '@generated/proto/ui';
import { OtherDefaults as SimUIOtherDefaults } from '@sim/spec_config';

import LaunchGear from './gear_sets/launch.gear.json';

export const DefaultSimpleRotation = DruidRotation.create({
	maintainFaerieFire: true,
	maintainDemoralizingRoar: true,
	maulRageThreshold: 50,
	swipeUsage: SwipeUsage.SwipeUsage_WithEnoughAP,
	swipeApThreshold: 2700,
});

import DefaultApl from './apls/default.apl.json';
export const ROTATION_SIMPLE = PresetUtils.makePresetSimpleRotation('Simple', Spec.SpecFeralBearDruid, DefaultSimpleRotation);
export const ROTATION_DEFAULT = PresetUtils.makePresetAPLRotation('Bear', DefaultApl);

export const BearTankTalents = PresetUtils.makePresetTalents('Bear Tank 0/31/20', SavedTalents.create({ talentsString: '-50032302120132010501-0550325' }));

export const DefaultOptions = DruidOptions.create({
	startingRage: 0,
});

// Master's consumables, as the Forever client's items.
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
	potId: 13455, // Greater Stoneshield Potion
});

export const OtherDefaults: Partial<SimUIOtherDefaults> = {
	profession1: Profession.Engineering,
	race: Race.RaceTauren,
	distanceFromTarget: 0,
	reactionTime: 200, // master's default
};

// Our Forever sim's gear presets (master ui/<spec>/gear_sets).
export const GEAR_LAUNCH = PresetUtils.makePresetGear('Launch', LaunchGear);
export const DEFAULT_GEAR = GEAR_LAUNCH;
export const GEAR_PRESETS = [GEAR_LAUNCH];
