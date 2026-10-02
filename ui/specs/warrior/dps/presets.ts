import * as PresetUtils from '@app/preset_utils';
import { RaidBuffs } from '@generated/proto/buffs';
import { HandType, ItemSlot, Profession, Race, Spec } from '@generated/proto/common';
import { SavedTalents } from '@generated/proto/ui';
import { DpsWarrior_Options as WarriorOptions, WarriorStance } from '@generated/proto/warrior';
import { Player } from '@sim/player/player';

import * as WarriorPresets from '../shared/presets';
import ForeverNoReckApl from './apls/dps_no_reck.apl.json';
import ForeverReckApl from './apls/dps_reck.apl.json';
import ArmsLaunchGear from './gear_sets/arms_launch.gear.json';
import LaunchGear from './gear_sets/launch.gear.json';
import P0BisGear from './gear_sets/p0.bis.gear.json';
import Phase1Gear from './gear_sets/phase_1.gear.json';
import Phase2Gear from './gear_sets/phase_2.gear.json';

// Preset options for this spec.
// Eventually we will import these values for the raid sim too, so its good to
// keep them in a separate file.

export const isArmsSpec = (player: Player<Spec.SpecDpsWarrior>) =>
	player.getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.handType === HandType.HandTypeTwoHand;

export const isArmsKebabSpec = (player: Player<Spec.SpecDpsWarrior>) => player.getTalents().mortalStrike && isFurySpec(player);

export const isFurySpec = (player: Player<Spec.SpecDpsWarrior>) =>
	player.getTalents().bloodthirst ||
	player.getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.handType === HandType.HandTypeMainHand ||
	player.getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.handType === HandType.HandTypeOneHand;

// Master's rotation presets, in master's order; No Reck is master's default.
export const ROTATION_PRESET_NO_RECK = PresetUtils.makePresetAPLRotation('DPS (No Reck)', ForeverNoReckApl);
export const ROTATION_PRESET_RECK = PresetUtils.makePresetAPLRotation('DPS (With Reck)', ForeverReckApl);

// The three builds our Forever sim ships.
export const DpsTalents = PresetUtils.makePresetTalents('DPS', SavedTalents.create({ talentsString: '30305013-05553005150010501' }));
export const FuryTalents = PresetUtils.makePresetTalents('Fury 17/34/0', SavedTalents.create({ talentsString: '30305213-55051105050010501' }));
export const ArmsTalents = PresetUtils.makePresetTalents('Arms 39/12/0', SavedTalents.create({ talentsString: '32305213132515201-5502' }));

export const DefaultOptions = WarriorOptions.create({
	classOptions: {
		queueDelay: 250,
		startingRage: 0,
		useBattleShout: true,
		defaultStance: WarriorStance.WarriorStanceBerserker,
	},
});

export const DefaultConsumables = WarriorPresets.DefaultConsumables;

export const DefaultRaidBuffs = RaidBuffs.create({
	giftOfTheWild: true,
});

export const OtherDefaults = {
	reactionTime: 200, // master's default
	race: Race.RaceHuman,
	profession1: Profession.Alchemy,
	profession2: Profession.Engineering,
};

// Our Forever sim's gear presets (master ui/<spec>/gear_sets).
export const GEAR_LAUNCH = PresetUtils.makePresetGear('Launch', LaunchGear);
export const GEAR_ARMS_LAUNCH = PresetUtils.makePresetGear('Launch (Arms)', ArmsLaunchGear);
export const GEAR_P0_BIS = PresetUtils.makePresetGear('Pre-BiS', P0BisGear);
export const GEAR_PHASE_1 = PresetUtils.makePresetGear('P1 BiS', Phase1Gear);
export const GEAR_PHASE_2 = PresetUtils.makePresetGear('P2 BiS', Phase2Gear);
export const DEFAULT_GEAR = GEAR_P0_BIS;
// Master's order.
export const GEAR_PRESETS = [GEAR_PHASE_2, GEAR_LAUNCH, GEAR_ARMS_LAUNCH, GEAR_PHASE_1, GEAR_P0_BIS];
