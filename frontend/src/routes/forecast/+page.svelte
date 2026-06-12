<script lang="ts">
	import DeadlineCountdown from '$lib/components/DeadlineCountdown.svelte';
	import { forecastStore as fs, koKey } from '$lib/forecast.svelte';
	import { tipsStore } from '$lib/tips.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import PlayerPicker from '$lib/components/PlayerPicker.svelte';
	import { Lock, Check, ChevronUp, ChevronDown } from '@lucide/svelte';
	import { collapseOnScroll } from '$lib/actions';
	import { language } from '$lib/language.svelte';

	let saveState = $state<'idle' | 'saving' | 'saved' | 'error'>('idle');
	let err = $state('');

	$effect(() => {
		if (!fs.loaded) {
			fs.load()
				.then(() => fs.loadAllPlayers())
				.catch((e) => (err = e?.message ?? 'Load failed'));
		} else if (fs.allPlayers.length === 0) {
			fs.loadAllPlayers().catch(() => {});
		}
	});

	let primed = false;
	let timer: ReturnType<typeof setTimeout>;
	$effect(() => {
		const snapshot = JSON.stringify([
			fs.groupOrder,
			fs.thirds,
			fs.bracket,
			fs.goldenBootName,
			fs.goldenBallPlayer,
			fs.goldenGlovePlayer,
			fs.bestYoungPlayer,
			fs.mostAssistsPlayer
		]);
		if (!fs.loaded || fs.locked) return;
		if (!primed) {
			primed = true;
			return;
		}
		void snapshot;
		clearTimeout(timer);
		timer = setTimeout(async () => {
			saveState = 'saving';
			err = '';
			try {
				await fs.save();
				saveState = 'saved';
			} catch (e: unknown) {
				saveState = 'error';
				err = (e as { message?: string })?.message ?? 'Could not save — your changes were not saved.';
			}
		}, 1000);
		return () => clearTimeout(timer);
	});

	const stageLabel: Record<string, string> = {
		R32: 'Round of 32',
		R16: 'Round of 16',
		QF: 'Quarter-finals',
		SF: 'Semi-finals',
		'3RD': 'Third-place play-off',
		FINAL: 'Final'
	};

	let koByStage = $derived.by(() => {
		const order = ['R32', 'R16', 'QF', 'SF', '3RD', 'FINAL'];
		const map = new Map<string, typeof fs.knockout>(order.map((s) => [s, []]));
		for (const m of fs.knockout) {
			map.get(m.stage)?.push(m);
		}
		return order
			.map((s) => ({ stage: s, matches: map.get(s) ?? [] }))
			.filter((x) => x.matches.length > 0);
	});

	let actualBestThirds = $derived(fs.actualBestThirds());

	function updatedAt(iso?: string) {
		if (!iso) return 'Not synced yet';
		const date = new Date(iso);
		if (!Number.isFinite(date.getTime())) return '';
		return new Intl.DateTimeFormat(language.locale, {
			day: '2-digit',
			month: 'short',
			hour: '2-digit',
			minute: '2-digit'
		}).format(date);
	}

	let gkPlayers = $derived(fs.allPlayers.filter((p) => p.position === 'GK'));
</script>

<div class="stickyhead" use:collapseOnScroll>
	<p class="kicker">Whole tournament</p>
	<div class="sh-expand">
		<div class="sh-inner">
			<h1>Tournament Predictions</h1>
			<p class="muted desc">
				Predict group standings, the knockout bracket, and award winners.
				{#if fs.locked}<b>Locked.</b>{:else}Locks at tournament kickoff.{/if}
			</p>
			{#if !fs.locked && fs.tournamentStart}
				<DeadlineCountdown deadline={fs.tournamentStart} label="Locks" compact />
			{/if}
		</div>
	</div>
</div>

{#if err}<p class="error">{err}</p>{/if}

{#if !fs.loaded}
	<p class="muted">Loading…</p>
{:else}
	{#if fs.locked}
		<div class="card lockbar">
			<Lock size={16} /> The tournament has started — predictions are final.
		</div>
	{/if}

	<!-- ── Group Stage ── -->
	{#if fs.groups.length > 0}
		<section class="pred-section">
			<h2 class="section-head">Group Stage</h2>
			<p class="section-desc muted">
				{#if fs.locked}Your predicted final standings in each group.{:else}Drag to reorder teams within each group — predict where each team finishes.{/if}
			</p>
			<div class="groups-grid">
				{#each fs.groups as group}
					{@const order = fs.groupOrder[group.letter] ?? group.teams}
					{@const actual = fs.actualOrder(group.letter)}
					<div class="group-card card">
						<div class="group-letter-bar">Group {group.letter}</div>
						{#each order as teamId, idx}
							{@const team = fs.team(teamId)}
							{@const isCorrect = actual !== null && actual[idx] === teamId}
							{@const isWrong = actual !== null && actual[idx] !== teamId}
							<div class="group-row" class:correct={isCorrect} class:wrong={isWrong}>
								<span class="g-pos digits">{idx + 1}</span>
								<Flag iso2={team?.iso2 ?? ''} code={team?.fifaCode ?? ''} />
								<span class="g-name">{team?.name ?? teamId}</span>
								{#if actual !== null}
									{#if isCorrect}
										<Check size={13} class="cind cok" />
									{:else}
										<span class="cind cerr">✗</span>
									{/if}
								{:else if !fs.locked}
									<div class="g-arrows">
										<button
											class="arr"
											onclick={() => fs.move(group.letter, idx, -1)}
											disabled={idx === 0}
											aria-label="Move up"
										><ChevronUp size={13} /></button>
										<button
											class="arr"
											onclick={() => fs.move(group.letter, idx, 1)}
											disabled={idx === order.length - 1}
											aria-label="Move down"
										><ChevronDown size={13} /></button>
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/each}
			</div>
		</section>
	{/if}

	<!-- ── Best Third ── -->
	{#if fs.groups.length > 0}
		<section class="pred-section">
			<h2 class="section-head">Best Third-Place Teams</h2>
			<p class="section-desc muted">
				Pick the 8 third-placed teams that advance to the Round of 32.
				<span class="thirds-count" class:full={fs.chosenThirdLetters.length >= fs.maxThirds}>
					{fs.chosenThirdLetters.length}&thinsp;/&thinsp;{fs.maxThirds}
				</span>
			</p>
			<div class="thirds-grid">
				{#each fs.groups as group}
					{@const thirdId = fs.groupThird(group.letter)}
					{@const team = thirdId ? fs.team(thirdId) : null}
					{@const picked = fs.chosenThirdLetters.includes(group.letter)}
					{@const okResult = actualBestThirds?.has(thirdId)}
					<button
						class="third-btn"
						class:picked
						class:correct={actualBestThirds !== null && okResult && picked}
						class:wrong={actualBestThirds !== null && !okResult && picked}
						onclick={() => !fs.locked && fs.toggleThird(group.letter)}
						disabled={fs.locked || (!picked && fs.chosenThirdLetters.length >= fs.maxThirds)}
					>
						<span class="third-grp">3rd {group.letter}</span>
						{#if team}
							<Flag iso2={team.iso2} code={team.fifaCode} />
							<span class="third-name">{team.name}</span>
						{:else}
							<span class="muted third-name">—</span>
						{/if}
					</button>
				{/each}
			</div>
		</section>
	{/if}

	<!-- ── Knockout Bracket ── -->
	{#if koByStage.length > 0}
		<section class="pred-section">
			<h2 class="section-head">Knockout Bracket</h2>
			<p class="section-desc muted">Pick the winner of each match round by round.</p>
		</section>
		{#each koByStage as { stage, matches }}
			<section class="pred-section ko-section">
				<h3 class="ko-stage-head">{stageLabel[stage] ?? stage}</h3>
				<div class="ko-grid" class:ko-single={matches.length === 1}>
					{#each matches as match}
						{@const [homeId, awayId] = fs.sides(match)}
						{@const home = homeId ? fs.team(homeId) : null}
						{@const away = awayId ? fs.team(awayId) : null}
						{@const pick = fs.bracket[koKey(match)]}
						{@const actual = fs.advancerOf(match.num)}
						<div class="ko-match">
							<button
								class="ko-team"
								class:picked={!!homeId && pick === homeId}
								class:correct={!!actual && actual === homeId && pick === homeId}
								class:wrong={!!actual && actual !== homeId && pick === homeId}
								onclick={() => homeId && !fs.locked && fs.pick(match, homeId)}
								disabled={!homeId || fs.locked}
							>
								{#if home}
									<Flag iso2={home.iso2} code={home.fifaCode} />
									<span>{home.name}</span>
								{:else}
									<span class="muted">{match.homeLabel}</span>
								{/if}
							</button>
							<span class="ko-vs">vs</span>
							<button
								class="ko-team"
								class:picked={!!awayId && pick === awayId}
								class:correct={!!actual && actual === awayId && pick === awayId}
								class:wrong={!!actual && actual !== awayId && pick === awayId}
								onclick={() => awayId && !fs.locked && fs.pick(match, awayId)}
								disabled={!awayId || fs.locked}
							>
								{#if away}
									<Flag iso2={away.iso2} code={away.fifaCode} />
									<span>{away.name}</span>
								{:else}
									<span class="muted">{match.awayLabel}</span>
								{/if}
							</button>
						</div>
					{/each}
				</div>
			</section>
		{/each}
	{/if}

	<!-- ── Tournament Awards ── -->
	<section class="pred-section">
		<h2 class="section-head">Tournament Awards</h2>
		<p class="section-desc muted">Predict the individual award winners.</p>
	</section>

	<div class="awards-grid">
		<!-- Golden Boot -->
		<div class="card award-card">
			<div class="award-head">
				<span class="award-icon" aria-hidden="true">⚽</span>
				<div>
					<h3>Golden Boot</h3>
					<p class="muted small">Most goals scored in the tournament.</p>
				</div>
				{#if fs.goldenBootName}<span class="award-check"><Check size={16} /></span>{/if}
			</div>
			{#if fs.locked}
				{@render lockedPick(fs.goldenBootName, fs.allPlayers)}
			{:else if fs.allPlayers.length === 0}
				<p class="muted small">Loading players…</p>
			{:else}
				<PlayerPicker
					players={fs.allPlayers}
					bind:value={fs.goldenBootName}
					locked={fs.locked}
					placeholder="Search Golden Boot candidate…"
				/>
			{/if}
		</div>

		<!-- Golden Ball -->
		<div class="card award-card">
			<div class="award-head">
				<span class="award-icon" aria-hidden="true">🏆</span>
				<div>
					<h3>Golden Ball</h3>
					<p class="muted small">Best overall player of the tournament.</p>
				</div>
				{#if fs.goldenBallPlayer}<span class="award-check"><Check size={16} /></span>{/if}
			</div>
			{#if fs.locked}
				{@render lockedPick(fs.goldenBallPlayer, fs.allPlayers)}
			{:else if fs.allPlayers.length === 0}
				<p class="muted small">Loading players…</p>
			{:else}
				<PlayerPicker
					players={fs.allPlayers}
					bind:value={fs.goldenBallPlayer}
					locked={fs.locked}
					placeholder="Search Golden Ball candidate…"
				/>
			{/if}
		</div>

		<!-- Golden Glove -->
		<div class="card award-card">
			<div class="award-head">
				<span class="award-icon" aria-hidden="true">🧤</span>
				<div>
					<h3>Golden Glove</h3>
					<p class="muted small">Best goalkeeper of the tournament.</p>
				</div>
				{#if fs.goldenGlovePlayer}<span class="award-check"><Check size={16} /></span>{/if}
			</div>
			{#if fs.locked}
				{@render lockedPick(fs.goldenGlovePlayer, gkPlayers.length > 0 ? gkPlayers : fs.allPlayers)}
			{:else if fs.allPlayers.length === 0}
				<p class="muted small">Loading players…</p>
			{:else}
				<PlayerPicker
					players={gkPlayers.length > 0 ? gkPlayers : fs.allPlayers}
					bind:value={fs.goldenGlovePlayer}
					locked={fs.locked}
					placeholder="Search goalkeeper…"
				/>
			{/if}
		</div>

		<!-- Best Young Player -->
		<div class="card award-card">
			<div class="award-head">
				<span class="award-icon" aria-hidden="true">⭐</span>
				<div>
					<h3>Best Young Player</h3>
					<p class="muted small">Best player born on or after January 1, 2004 (under 21).</p>
				</div>
				{#if fs.bestYoungPlayer}<span class="award-check"><Check size={16} /></span>{/if}
			</div>
			{#if fs.locked}
				{@render lockedPick(fs.bestYoungPlayer, fs.allPlayers)}
			{:else if fs.allPlayers.length === 0}
				<p class="muted small">Loading players…</p>
			{:else}
				<PlayerPicker
					players={fs.allPlayers}
					bind:value={fs.bestYoungPlayer}
					locked={fs.locked}
					placeholder="Search young player…"
				/>
			{/if}
		</div>

		<!-- Most Assists -->
		<div class="card award-card award-card--wide">
			<div class="award-head">
				<span class="award-icon" aria-hidden="true">🎯</span>
				<div>
					<h3>Most Assists</h3>
					<p class="muted small">Player with the most assists in the tournament.</p>
				</div>
				{#if fs.mostAssistsPlayer}<span class="award-check"><Check size={16} /></span>{/if}
			</div>
			{#if fs.locked}
				{@render lockedPick(fs.mostAssistsPlayer, fs.allPlayers)}
			{:else if fs.allPlayers.length === 0}
				<p class="muted small">Loading players…</p>
			{:else}
				<PlayerPicker
					players={fs.allPlayers}
					bind:value={fs.mostAssistsPlayer}
					locked={fs.locked}
					placeholder="Search assists leader…"
				/>
			{/if}
		</div>
	</div>

	{#if !fs.locked}
		<div class="savebar">
			<span class="savestat" class:err={saveState === 'error'}>
				{#if saveState === 'saving'}
					Saving…
				{:else if saveState === 'error'}
					{err || 'Save failed'}
				{:else if saveState === 'saved'}
					<Check size={15} /> Saved · changes are saved automatically
				{:else}
					Changes are saved automatically
				{/if}
			</span>
		</div>
	{/if}
{/if}

{#snippet lockedPick(name: string, players: typeof fs.allPlayers)}
	{#if name}
		{@const picked = players.find((p) => p.name === name)}
		{@const team = picked ? tipsStore.teams[picked.teamId] : undefined}
		<div class="locked-pick">
			{#if team}<Flag iso2={team.iso2} code={team.fifaCode} />{/if}
			<b>{name}</b>
			{#if team}<span class="muted small">{team.name}</span>{/if}
		</div>
	{:else}
		<p class="muted small">No pick made.</p>
	{/if}
{/snippet}

<style>
	h1 {
		margin: 0.25rem 0 0.2rem;
	}
	h3 {
		margin: 0 0 0.25rem;
	}
	.small {
		font-size: 0.85rem;
	}
	.lockbar {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--warning);
	}
	.stickyhead {
		position: sticky;
		top: var(--topbar-h);
		z-index: 20;
		margin: 0 -1rem;
		padding: 0.6rem 1rem 0.75rem;
		background: var(--bg);
		border-bottom: 1px solid var(--border);
	}
	.stickyhead h1 {
		margin: 0.1rem 0 0;
	}
	.stickyhead .desc {
		margin: 0.3rem 0 0;
		font-size: 0.9rem;
	}
	@media (min-width: 900px) {
		.stickyhead {
			top: 0;
			margin: 0 -2rem;
			padding: 0.75rem 2rem 0.85rem;
		}
	}

	/* Section headers */
	.pred-section {
		margin-top: 1.5rem;
	}
	.pred-section:first-of-type {
		margin-top: 1rem;
	}
	.section-head {
		font-size: 1rem;
		font-weight: 800;
		letter-spacing: 0.03em;
		text-transform: uppercase;
		margin: 0 0 0.2rem;
		color: var(--muted);
	}
	.section-desc {
		margin: 0 0 0.75rem;
		font-size: 0.85rem;
	}

	/* ── Groups ── */
	.groups-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.6rem;
	}
	@media (max-width: 500px) {
		.groups-grid {
			grid-template-columns: 1fr;
		}
	}
	.group-card {
		padding: 0;
		overflow: hidden;
	}
	.group-letter-bar {
		padding: 0.45rem 0.75rem;
		font-size: 0.75rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
		border-bottom: 1px solid var(--border);
		background: color-mix(in srgb, var(--surface-2) 50%, transparent);
	}
	.group-row {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.35rem 0.6rem 0.35rem 0.45rem;
		border-bottom: 1px solid var(--border);
		font-size: 0.82rem;
		transition: background 0.12s;
	}
	.group-row:last-child {
		border-bottom: none;
	}
	.group-row.correct {
		background: color-mix(in srgb, var(--success) 8%, transparent);
	}
	.group-row.wrong {
		background: color-mix(in srgb, var(--danger) 6%, transparent);
	}
	.g-pos {
		width: 1rem;
		text-align: center;
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--muted);
		flex-shrink: 0;
	}
	.g-name {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-weight: 600;
	}
	:global(.cind) {
		flex-shrink: 0;
		font-size: 0.75rem;
	}
	:global(.cok) {
		color: var(--success);
	}
	.cerr {
		color: var(--danger);
	}
	.g-arrows {
		display: flex;
		flex-direction: column;
		gap: 1px;
		flex-shrink: 0;
	}
	.arr {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.3rem;
		height: 1.1rem;
		background: none;
		border: none;
		color: var(--muted);
		cursor: pointer;
		border-radius: var(--radius-sm);
		padding: 0;
	}
	.arr:disabled {
		opacity: 0.2;
		cursor: default;
	}
	.arr:not(:disabled):hover {
		background: var(--surface-2);
		color: var(--text);
	}

	/* ── Thirds ── */
	.thirds-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.4rem;
	}
	@media (max-width: 500px) {
		.thirds-grid {
			grid-template-columns: repeat(2, 1fr);
		}
	}
	.thirds-count {
		display: inline-flex;
		align-items: center;
		padding: 0.1rem 0.45rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface-2);
		font-size: 0.75rem;
		font-weight: 700;
		margin-left: 0.5rem;
	}
	.thirds-count.full {
		border-color: var(--success);
		color: var(--success);
	}
	.third-btn {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.45rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-size: 0.8rem;
		cursor: pointer;
		text-align: left;
		transition: background 0.12s, border-color 0.12s;
		min-width: 0;
	}
	.third-btn:disabled {
		opacity: 0.45;
		cursor: default;
	}
	.third-btn.picked {
		background: color-mix(in srgb, var(--accent) 12%, var(--surface-2));
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
	}
	.third-btn.correct {
		background: color-mix(in srgb, var(--success) 10%, var(--surface-2));
		border-color: color-mix(in srgb, var(--success) 45%, var(--border));
	}
	.third-btn.wrong {
		background: color-mix(in srgb, var(--danger) 8%, var(--surface-2));
		border-color: color-mix(in srgb, var(--danger) 35%, var(--border));
	}
	.third-grp {
		font-size: 0.68rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
		flex-shrink: 0;
	}
	.third-name {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-weight: 600;
	}

	/* ── KO Bracket ── */
	.ko-section {
		margin-top: 1rem;
	}
	.ko-stage-head {
		font-size: 0.78rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
		margin: 0 0 0.45rem;
	}
	.ko-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.45rem;
	}
	.ko-grid.ko-single {
		grid-template-columns: 1fr;
		max-width: 420px;
	}
	@media (max-width: 500px) {
		.ko-grid {
			grid-template-columns: 1fr;
		}
	}
	.ko-match {
		display: flex;
		align-items: stretch;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		overflow: hidden;
		background: var(--surface-2);
	}
	.ko-team {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.55rem 0.65rem;
		background: none;
		border: none;
		color: var(--text);
		font: inherit;
		font-size: 0.82rem;
		font-weight: 600;
		cursor: pointer;
		text-align: left;
		min-width: 0;
		transition: background 0.12s;
	}
	.ko-team:disabled {
		cursor: default;
		color: var(--muted);
	}
	.ko-team:not(:disabled):hover {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
	}
	.ko-team.picked {
		background: color-mix(in srgb, var(--accent) 15%, var(--surface-2));
	}
	.ko-team.correct {
		background: color-mix(in srgb, var(--success) 12%, var(--surface-2));
	}
	.ko-team.wrong {
		background: color-mix(in srgb, var(--danger) 8%, var(--surface-2));
	}
	.ko-team span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.ko-vs {
		display: flex;
		align-items: center;
		padding: 0 0.3rem;
		font-size: 0.72rem;
		font-weight: 700;
		color: var(--muted);
		border-left: 1px solid var(--border);
		border-right: 1px solid var(--border);
		background: color-mix(in srgb, var(--surface-2) 60%, transparent);
		flex-shrink: 0;
	}

	/* ── Awards ── */
	.awards-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.6rem;
		/* allow absolute-positioned dropdowns to overflow */
		overflow: visible;
	}
	@media (max-width: 520px) {
		.awards-grid {
			grid-template-columns: 1fr;
		}
	}
	.award-card {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		/* let PlayerPicker dropdown escape the card boundary */
		overflow: visible;
	}
	.award-card--wide {
		grid-column: 1 / -1;
	}
	.award-head {
		display: flex;
		align-items: flex-start;
		gap: 0.7rem;
	}
	.award-icon {
		font-size: 1.4rem;
		line-height: 1;
		flex-shrink: 0;
		margin-top: 0.1rem;
	}
	.award-head > div {
		flex: 1;
		min-width: 0;
	}
	.award-head .small {
		margin: 0;
		color: var(--muted);
	}
	.award-check {
		color: var(--success);
		flex-shrink: 0;
		margin-top: 0.25rem;
	}
	.locked-pick {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.5rem 0.75rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		font-size: 0.88rem;
	}

	/* ── Save bar ── */
	.savebar {
		position: sticky;
		bottom: calc(var(--nav-h) + 0.5rem);
		display: flex;
		justify-content: center;
		margin-top: 1.5rem;
		pointer-events: none;
	}
	.savestat {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.8rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--muted);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		padding: 0.4rem 0.85rem;
	}
	.savestat.err {
		color: var(--danger);
		border-color: var(--danger);
		text-transform: none;
		letter-spacing: 0;
	}

	/* ── Theme overrides ── */
	:global(:root[data-theme='worldcup']) .group-card,
	:global(:root[data-theme='worldcup']) .award-card,
	:global(:root[data-theme='worldcup']) .lockbar {
		background:
			radial-gradient(circle at 14% 0%, rgba(143, 197, 143, 0.075), transparent 32%),
			linear-gradient(180deg, rgba(13, 34, 40, 0.96), rgba(7, 17, 25, 0.98)),
			var(--surface);
		border-color: color-mix(in srgb, var(--accent) 12%, var(--border));
		box-shadow:
			0 16px 42px -34px rgba(0, 0, 0, 0.9),
			inset 0 1px 0 rgba(255, 255, 255, 0.035);
	}
	:global(:root[data-theme='worldcup']) .group-card::before,
	:global(:root[data-theme='worldcup']) .award-card::before,
	:global(:root[data-theme='worldcup']) .lockbar::before {
		display: none;
	}
	:global(:root[data-theme='worldcup']) .savestat {
		background: color-mix(in srgb, var(--surface-2) 78%, transparent);
		border-color: color-mix(in srgb, var(--accent) 12%, var(--border));
	}
	:global(:root[data-theme='worldcup']) .ko-match,
	:global(:root[data-theme='worldcup']) .third-btn {
		background: color-mix(in srgb, var(--surface-2) 65%, transparent);
		border-color: color-mix(in srgb, var(--accent) 10%, var(--border));
	}
</style>
