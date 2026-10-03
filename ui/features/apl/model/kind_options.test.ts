import { makePlayer } from '@features/apl/testing';
import { APLRotation } from '@generated/proto/apl';
import { Spec } from '@generated/proto/common';
import { describe, expect, it, vi } from 'vitest';

import { actionKinds } from './action_kinds';
import { actionKindOptions, valueKindOptions } from './kind_options';
import { valueKinds } from './value_kinds';

vi.mock('@i18n/config', () => ({ default: { t: (key: string) => key } }));

const player = (): any => makePlayer(APLRotation.create());

describe('valueKindOptions includeIf', () => {
	it('drops a prepull-excluded kind when isPrepull is true, and keeps it otherwise', () => {
		const notPrepull = valueKindOptions(player(), false, false).map(option => option.value);
		const prepull = valueKindOptions(player(), true, false).map(option => option.value);
		expect(notPrepull).toContain('currentTime');
		expect(prepull).not.toContain('currentTime');
	});
});

describe('actionKindOptions includeIf', () => {
	it('drops a prepull-excluded kind when isPrepull is true, and keeps it otherwise', () => {
		const notPrepull = actionKindOptions(player(), false).map(option => option.value);
		const prepull = actionKindOptions(player(), true).map(option => option.value);
		expect(notPrepull).toContain('multidot');
		expect(prepull).not.toContain('multidot');
	});
});

describe('kind tooltip', () => {
	it('wraps the short description and appends the full description when one exists', () => {
		const options = valueKindOptions(player(), false, false);
		const withFull = options.find(option => option.value === 'const');
		expect(withFull?.tooltip).toBe(`<p>${valueKinds.const.shortDescription}</p> ${valueKinds.const.fullDescription}`);
	});

	it('is just the short description when there is no full description', () => {
		const options = valueKindOptions(player(), false, false);
		const shortOnly = options.find(option => option.value === 'cmp');
		expect(valueKinds.cmp.fullDescription).toBeUndefined();
		expect(shortOnly?.tooltip).toBe(valueKinds.cmp.shortDescription);
	});
});

describe('kinds the engine has no handler for', () => {
	// Each is kept in the table so an old rotation naming one still renders, but none is offered:
	// the engine disables any action whose condition uses one.
	const unsupported = [
		'currentSolarEnergy',
		'currentLunarEnergy',
		'druidCurrentEclipsePhase',
		'currentGenericResource',
		'dotCritPercentIncrease',
		'protectionPaladinDamageTakenLastGlobal',
	] as const;

	it.each([Spec.SpecBalanceDruid, Spec.SpecProtectionPaladin, Spec.SpecMage])('are never offered to %s', spec => {
		const offered = valueKindOptions({ ...player(), getSpec: () => spec }, false, false).map(option => option.value);
		for (const kind of unsupported) {
			expect(valueKinds[kind]).toBeDefined();
			expect(offered).not.toContain(kind);
		}
	});
});

describe('table insertion order', () => {
	it('valueKindOptions preserves the value-kind table order', () => {
		const order = Object.keys(valueKinds);
		const indices = valueKindOptions(player(), false, false).map(option => order.indexOf(option.value));
		expect(indices).toEqual([...indices].sort((a, b) => a - b));
		expect(new Set(indices).size).toBe(indices.length);
	});

	it('actionKindOptions preserves the action-kind table order', () => {
		const order = Object.keys(actionKinds);
		const indices = actionKindOptions(player(), false).map(option => order.indexOf(option.value));
		expect(indices).toEqual([...indices].sort((a, b) => a - b));
		expect(new Set(indices).size).toBe(indices.length);
	});
});
