import * as PresetUtils from '@app/preset_utils';
import { ConsumesSpec, Profession, Race, Spec } from '@generated/proto/common';
import {
	FeralCatDruid_Options as FeralDruidOptions,
	FeralCatDruid_Rotation as FeralCatDruidRotation,
	FeralCatDruid_Rotation_FinishingMove as FinishingMove,
} from '@generated/proto/druid';
import { SavedTalents } from '@generated/proto/ui';

import DefaultApl from './apls/default.apl.json';
import SimpleVaelApl from './apls/simple_vael.apl.json';
import LaunchGear from './gear_sets/launch.gear.json';
import P0BisGear from './gear_sets/p0.bis.gear.json';
import P2BisGear from './gear_sets/p2.bis.gear.json';
import P2PreBisGear from './gear_sets/p2.pre-bis.gear.json';

export const DefaultOptions = FeralDruidOptions.create({});

// Master's consumables, as the Forever client's items. Goblin Sapper drops Cat Form; the APL
// shifts back.
export const DefaultConsumables = ConsumesSpec.create({
	flaskId: 13511, // Flask of Distilled Wisdom
	battleElixirId: 13452, // Elixir of the Mongoose
	strengthBuffId: 12451, // Juju Power
	attackPowerBuffId: 12460, // Juju Might
	zanzaId: 8412, // Ground Scorpok Assay
	dragonbreathChili: true,
	foodId: 13928, // Grilled Squid
	potId: 13444, // Major Mana Potion
	conjuredId: 12662, // Demonic Rune
	goblinSapper: true,
});

export const OtherDefaults = {
	distanceFromTarget: 0,
	profession1: Profession.Engineering,
	profession2: Profession.Leatherworking,
	race: Race.RaceTauren,
	reactionTime: 200, // master's default
};

export const DefaultRotation = FeralCatDruidRotation.create({
	finishingMove: FinishingMove.Rip,
	biteweave: true,
	ripMinComboPoints: 5,
	biteMinComboPoints: 5,
	mangleTrick: true,
	maintainFaerieFire: true,
});

export const SIMPLE = PresetUtils.makePresetSimpleRotation('Simple', Spec.SpecFeralCatDruid, DefaultRotation);

export const APL = PresetUtils.makePresetAPLRotation('Feral', DefaultApl);
export const APL_SIMPLE_VAEL = PresetUtils.makePresetAPLRotation('Simple Vaelastrasz', SimpleVaelApl);

export const FeralTalents = PresetUtils.makePresetTalents('Feral', SavedTalents.create({ talentsString: '-55210032021132212051-05503' }));
export const FeralCatTalents = PresetUtils.makePresetTalents('Feral Cat 9/35/7', SavedTalents.create({ talentsString: '050022-55000032121032212051-052' }));

// Our Forever sim's gear presets (master ui/<spec>/gear_sets).
export const GEAR_LAUNCH = PresetUtils.makePresetGear('Launch', LaunchGear);
export const GEAR_P0_BIS = PresetUtils.makePresetGear('Pre-BiS', P0BisGear);
export const GEAR_P2_PRE_BIS = PresetUtils.makePresetGear('P2 Pre-BiS', P2PreBisGear);
export const GEAR_P2_BIS = PresetUtils.makePresetGear('P2 BiS', P2BisGear);
export const DEFAULT_GEAR = GEAR_P0_BIS;
export const GEAR_PRESETS = [GEAR_LAUNCH, GEAR_P0_BIS, GEAR_P2_PRE_BIS, GEAR_P2_BIS];
