// Every ability the sim registers, and what is actually known about its numbers.
//
// The manifest in ui/core/spells has always held this, and sim/spell_sources_test.go has
// always enforced it, but the only way to read it was to clone the repository. On screen an
// ability nobody has settled looked exactly like one lifted from the client. The icons now
// carry a dot for the unsettled ones; this page is where that dot leads.

import { ActionId } from '@sim/proto/action_id';
import { allSpellSources, type SpellSource } from '@sim/spells/index';
import { useActionId } from '@ui-kit/hooks/useActionId';
import { Icon } from '@ui-kit/Icon';
import clsx from 'clsx';
import { useEffect, useMemo, useRef, useState } from 'react';

import { ProductPage, SITE_BASE, SITE_REPO_URL } from '../ProductPage';

type Tier = 'measured' | 'forever' | 'classic' | 'assumed';

const TIERS: Array<{ key: Tier; label: string; blurb: string }> = [
	{ key: 'measured', label: 'Seen in game', blurb: "Watched happen on a running server, through the client's own damage meter." },
	{ key: 'forever', label: 'Read from the client', blurb: "Taken out of the beta client's own data tables." },
	{ key: 'classic', label: 'Unchanged from Classic', blurb: 'Identical to Classic Era, so the long-established value stands.' },
	{ key: 'assumed', label: 'Still a guess', blurb: 'At least one number here is unconfirmed, and the row says which.' },
];

// Gold is the mark for "read the hover before trusting this", so it means the same thing
// here. Green is the only tier that has been seen happen.
const CHIP: Record<Tier, string> = {
	measured: 'border-evidence-measured text-evidence-measured',
	forever: 'border-white/20',
	classic: 'border-white/20',
	assumed: 'border-evidence-assumed text-evidence-assumed',
};

// What would actually move a number, worst first, with what to send for each.
//
// Hand-written on purpose. The manifest knows which abilities are unsettled, but not which
// unsettled thing matters - Ice Lance's coefficient moves every frost mage on the site, while
// a hunter pet's attack speed is a rounding error. A list generated from
// the manifest would rank those the same and quietly waste the first person who offers to
// help. The counts inside it come from the manifest, so those cannot go stale.
type Need = {
	title: string;
	why: string;
	/** What to send, in the words of someone who has the game open. */
	send: string;
	/** Pre-fills the search box below, so the row in question is one click away. */
	find?: string;
};

// How many rows still carry a guess, across the game or under one directory of sim/.
const guesses = (dir = 'sim/') => allSpellSources().filter(([, s]) => s.source === 'assumed' && s.file.startsWith(dir)).length;

const NEEDS: Array<Need> = [
	{
		title: 'Hunter: Volley, the hawk, and the attack power share at level 60',
		why: `${guesses('sim/hunter/')} of the ${guesses()} abilities that still carry a guess are hunter ones. Arcane Shot and Serpent Sting do scale with attack power although the client gives them none: a beta hunter who swapped Aspect of the Hawk for Cheetah mid-dungeon lost 0.10 to 0.12 of the 35 attack power on each Arcane Shot and about 0.04 on each Serpent Sting tick, which is the sim's 0.11 and 0.035. That share is fitted at level 20, though. Volley keeps Classic's coefficient because the client carries only a placeholder, and the hawk from Summon Hawk swings in a way the client does not describe.`,
		send: 'Once the cap allows it: a DamageMeter.bin from a hunter casting Volley or with Summon Hawk out, with Aspect of the Hawk up and then down. Ten Arcane Shots each way at level 30 or above also checks the share still holds.',
		find: 'hunter',
	},
	{
		title: 'Warrior: is Flurry 25% at five points, or a flat 30%?',
		why: "The talent's rank curve reads 5% attack speed a point, Classic's values, but the buff it triggers now carries a flat 30%. The sim applies the curve. If the buff wins, every Fury warrior on the site swings faster than shown. Flurry sits at level 35, above the beta's level cap, so no public beta log has it yet.",
		send: 'Once the cap allows it: a screenshot of the Flurry buff tooltip with one point in the talent. It shows 5% or 30%, and that settles it.',
		find: 'Flurry',
	},
	{
		title: 'Any tooltip that disagrees with this sim',
		why: 'Five bugs so far passed a value check and were wrong about what the value applied to. Improved Seals scaled half of what it should while every number matched. A rank curve says what a talent’s numbers are, never what they do.',
		send: 'A screenshot, cropped to the tooltip. If it contradicts what is written on a row below, that row is wrong and it takes one picture to prove it.',
	},
];

const tierOf = (s: SpellSource): Tier => s.source as Tier;

/** A row's searchable text, so filtering never has to walk the DOM. */
const haystack = (id: number, s: SpellSource) => `${id} ${s.ability} ${s.file} ${s.note ?? ''} ${(s.assumptions ?? []).join(' ')}`.toLowerCase();

// Every row is an icon lookup, and an icon the bundled database has never heard of goes
// out to Wowhead for it. Firing those on load would be hundreds of requests for the forty
// rows anyone can actually see, so a row resolves its ActionId (icon and href both) only
// once it scrolls into view. One observer for every row; a hidden row intersects nothing,
// so filtering down to it is what brings it into view.
const onSeen = new Map<Element, () => void>();
const rowsInView =
	typeof IntersectionObserver === 'undefined'
		? null
		: new IntersectionObserver(
				(entries, self) => {
					for (const entry of entries) {
						if (!entry.isIntersecting) continue;
						self.unobserve(entry.target);
						onSeen.get(entry.target)?.();
						onSeen.delete(entry.target);
					}
				},
				{ rootMargin: '200px' },
			);

const useSeen = () => {
	const ref = useRef<HTMLAnchorElement>(null);
	const [seen, setSeen] = useState(!rowsInView);
	useEffect(() => {
		const elem = ref.current;
		if (!elem || !rowsInView) return;
		onSeen.set(elem, () => setSeen(true));
		rowsInView.observe(elem);
		return () => {
			rowsInView.unobserve(elem);
			onSeen.delete(elem);
		};
	}, []);
	return [ref, seen] as const;
};

const RowIcon = ({ id }: { id: number }) => {
	const [ref, seen] = useSeen();
	const actionId = useMemo(() => (seen ? ActionId.fromSpellId(id) : undefined), [id, seen]);
	const { iconUrl, href } = useActionId(actionId);
	return (
		<a
			ref={ref}
			className="col-start-1 row-span-2 row-start-1 size-10 rounded-sm bg-black/40 bg-cover"
			target="_blank"
			rel="noreferrer"
			href={href || undefined}
			style={iconUrl ? { backgroundImage: `url('${iconUrl}')` } : undefined}
			data-spell-id={id}
		/>
	);
};

const Chip = ({ tier, children }: { tier: Tier; children: string }) => (
	<span className={clsx('rounded-full border px-2 py-0.5 text-xs whitespace-nowrap', CHIP[tier])}>{children}</span>
);

const Row = ({ id, s, hidden }: { id: number; s: SpellSource; hidden: boolean }) => {
	const tier = tierOf(s);
	return (
		<li
			hidden={hidden}
			className="grid grid-cols-[2.5rem_minmax(0,1fr)] gap-x-2 gap-y-1 border-b border-white/8 py-2 md:grid-cols-[2.5rem_minmax(12rem,18rem)_minmax(0,1fr)] md:items-start"
			data-spell-source={s.source}
			data-testid="evidence-row">
			<RowIcon id={id} />
			<div className="col-start-2 row-start-1 flex min-w-0 flex-wrap items-baseline gap-2">
				<span className="font-semibold wrap-anywhere text-white">{s.ability}</span>
				<span className="text-sm text-gray-500 tabular-nums">{id}</span>
			</div>
			<div className="col-start-2 row-start-2 flex flex-wrap gap-1 self-start">
				<Chip tier={tier}>{TIERS.find(t => t.key === tier)!.label}</Chip>
				{s.measured && <Chip tier="measured">{`Seen ${s.measured.date}`}</Chip>}
			</div>
			<div className="col-span-2 flex min-w-0 flex-col gap-1 text-sm wrap-anywhere md:col-span-1 md:col-start-3 md:row-span-2 md:row-start-1">
				{s.tooltip && <p className="m-0 whitespace-pre-line text-gray-300">{s.tooltip}</p>}
				{s.measured && (
					<p className="m-0 text-evidence-measured/80">
						Client says <strong>{s.measured.client}</strong>; the game&apos;s own meter recorded a largest hit of{' '}
						<strong>{s.measured.biggest}</strong>, via {s.measured.how}.
					</p>
				)}
				{(s.assumptions ?? []).map(a => (
					<p key={a} className="m-0 border-l-3 border-brand pl-2 text-gray-200">
						{a}
					</p>
				))}
				{s.note && <p className="m-0 text-xs text-gray-500">{s.note}</p>}
				<p className="m-0 text-xs text-gray-500">
					<code>{s.file}</code>
				</p>
			</div>
		</li>
	);
};

const TierButton = ({ active, label, count, onClick }: { active: boolean; label: string; count: number; onClick: () => void }) => (
	<button
		type="button"
		data-active={active ? '' : undefined}
		className="inline-flex cursor-pointer items-center gap-1 rounded-full border border-white/18 bg-transparent px-3 py-1.5 text-sm text-gray-300 hover:border-brand hover:text-white data-active:border-brand data-active:bg-brand/15 data-active:text-white"
		onClick={onClick}>
		<span>{label}</span>
		<span className="text-gray-500 tabular-nums">{count}</span>
	</button>
);

export const EvidencePage = () => {
	const entries = useMemo(
		() =>
			allSpellSources()
				.filter(([, s]) => s.source !== 'unreviewed')
				.sort(([, a], [, b]) => a.ability.localeCompare(b.ability))
				.map(([id, s]) => ({ id, s, text: haystack(id, s), tiers: new Set<Tier>(s.measured ? [tierOf(s), 'measured'] : [tierOf(s)]) })),
		[],
	);
	const tally = useMemo(() => {
		const out = new Map<Tier, number>();
		for (const { s } of entries) out.set(tierOf(s), (out.get(tierOf(s)) ?? 0) + 1);
		return out;
	}, [entries]);
	const measured = entries.filter(({ s }) => !!s.measured).length;

	const [search, setSearch] = useState('');
	const [tier, setTier] = useState<Tier | 'all'>('all');
	const searchRef = useRef<HTMLInputElement>(null);
	const query = search.trim().toLowerCase();
	const visible = entries.map(row => (tier === 'all' || row.tiers.has(tier)) && (!query || row.text.includes(query)));
	const shown = visible.filter(Boolean).length;

	/** Drops a most-wanted item straight into the list below it. */
	const find = (value: string) => {
		setSearch(value);
		searchRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	};

	return (
		<ProductPage
			title="Where every number came from"
			subtitle={`Every ability this sim runs, and how much is actually known about it. ${entries.length} of them: ${measured} have been watched happen on a running server, ${tally.get('assumed') ?? 0} still carry a guess, and the rest are read straight out of the beta client.`}>
			<div className="flex flex-col gap-4 text-gray-300">
				<section className="rounded-sm border border-brand/35 bg-brand/6 px-6 py-4" id="most-wanted">
					<h2 className="m-0 text-xl text-white">Most wanted</h2>
					<p className="m-0 mt-2 max-w-208">
						Worst first. Each of these can be closed by one person with the game open.{' '}
						<a className="font-semibold text-brand" href={`${SITE_BASE}scrub/`}>
							Send a file or a screenshot
						</a>{' '}
						&mdash; no account, no form.
					</p>
					<ol className="m-0 mt-4 list-none divide-y divide-white/10 p-0">
						{NEEDS.map((need, i) => (
							<li key={need.title} className="py-4 first:pt-0 last:pb-0">
								<h3 className="m-0 flex items-baseline gap-2 text-base text-white">
									<span className="flex-none text-brand tabular-nums">{i + 1}</span>
									{need.title}
								</h3>
								<p className="m-0 mt-1 text-sm text-gray-400">{need.why}</p>
								<p className="m-0 mt-1 text-sm text-gray-200">
									<strong>What settles it:</strong> {need.send}
								</p>
								{need.find && (
									<button
										type="button"
										className="mt-2 inline-flex cursor-pointer items-center gap-1 rounded-full border border-brand bg-transparent px-3 py-1 text-sm text-white hover:bg-brand/15 focus-visible:bg-brand/15"
										onClick={() => find(need.find!)}>
										<Icon name="magnifying-glass" />
										<span>Show these rows</span>
									</button>
								)}
							</li>
						))}
					</ol>
				</section>

				{/* Sticky so the filter is still reachable a few hundred rows down, which is most of them. */}
				<div className="sticky top-0 z-2 flex flex-col gap-2 bg-black py-2 md:flex-row md:items-center">
					<input
						ref={searchRef}
						className="w-full rounded-sm border border-white/18 bg-black/35 px-3 py-2 text-white focus:border-brand focus:outline-none md:max-w-88 md:flex-[0_1_22rem]"
						type="search"
						placeholder="Filter by name, spell id, file or reason"
						value={search}
						onChange={event => setSearch(event.target.value)}
					/>
					<div className="flex flex-wrap gap-1">
						<TierButton active={tier === 'all'} label="Everything" count={entries.length} onClick={() => setTier('all')} />
						{TIERS.map(t => (
							<TierButton
								key={t.key}
								active={tier === t.key}
								label={t.label}
								count={t.key === 'measured' ? measured : (tally.get(t.key) ?? 0)}
								onClick={() => setTier(t.key)}
							/>
						))}
					</div>
				</div>

				<ul className="m-0 grid list-none grid-cols-[repeat(auto-fit,minmax(min(20rem,100%),1fr))] gap-x-4 gap-y-1 p-0 text-sm text-gray-400">
					{TIERS.map(t => (
						<li key={t.key} className="flex flex-wrap items-baseline gap-2">
							<Chip tier={t.key}>{t.label}</Chip>
							<span>{t.blurb}</span>
						</li>
					))}
				</ul>

				<p className="m-0 text-gray-500 tabular-nums" data-testid="evidence-count">
					Showing {shown} of {entries.length}
				</p>

				<ul className="m-0 list-none p-0">
					{entries.map(({ id, s }, i) => (
						<Row key={id} id={id} s={s} hidden={!visible[i]} />
					))}
				</ul>

				<p className="m-0 mt-4 text-sm text-gray-400">
					Every spell that deals damage in a build the arena ranks has to be listed here; each spec&apos;s <code>TestArena</code> fails otherwise. The
					entries themselves live in{' '}
					<a href={`${SITE_REPO_URL}/tree/master/ui/sim/spells`} target="_blank" rel="noreferrer">
						ui/sim/spells
					</a>
					. If you can move a row up a tier, <a href={`${SITE_BASE}scrub/`}>send the beta&apos;s own numbers</a>.
				</p>
			</div>
		</ProductPage>
	);
};
