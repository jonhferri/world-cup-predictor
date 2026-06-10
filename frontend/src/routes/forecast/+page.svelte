<script lang="ts">
	import DeadlineCountdown from '$lib/components/DeadlineCountdown.svelte';
	import { api, type GoldenBootSearchResult } from '$lib/api';
	import { forecastStore as fs, type GoldenBootPlayer } from '$lib/forecast.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import { vibrate } from '$lib/haptics';
	import { teamDisplayName } from '$lib/teamNames';
	import { Lock, Check, Trophy } from '@lucide/svelte';
	import { collapseOnScroll } from '$lib/actions';
	import { language } from '$lib/language.svelte';

	let saveState = $state<'idle' | 'saving' | 'saved' | 'error'>('idle');
	let err = $state('');
	$effect(() => {
		if (!fs.loaded) fs.load().catch((e) => (err = e?.message ?? 'Load failed'));
	});

	let primed = false;
	let timer: ReturnType<typeof setTimeout>;
	$effect(() => {
		const snapshot = JSON.stringify([fs.goldenBootPlayer]);
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

	let goldenBootById = $derived.by(() => {
		const out: Record<string, GoldenBootPlayer> = {};
		for (const player of [...fs.goldenBoot.shortlist, ...fs.goldenBoot.leaders]) out[player.id] = player;
		return out;
	});
	let goldenBootPick = $derived(goldenBootById[fs.goldenBootPlayer]);
	let goldenBootLeaders = $derived(
		fs.goldenBoot.leaders.length > 0 ? fs.goldenBoot.leaders : fs.goldenBoot.shortlist.slice(0, 10)
	);
	let goldenBootSearchQuery = $state('');
	let goldenBootSearchResults = $state<GoldenBootSearchResult[]>([]);
	let goldenBootSearchLoading = $state(false);
	let goldenBootSearchError = $state('');
	let goldenBootSearchPendingKey = $state('');
	let goldenBootSearchApiAvailable = $state(true);
	let goldenBootSearchTimer: ReturnType<typeof setTimeout>;

	$effect(() => {
		const query = goldenBootSearchQuery.trim();
		if (fs.locked) {
			goldenBootSearchLoading = false;
			goldenBootSearchError = '';
			return;
		}
		if (query.length < 2) {
			goldenBootSearchResults = [];
			goldenBootSearchLoading = false;
			goldenBootSearchError = '';
			return;
		}

		let cancelled = false;
		clearTimeout(goldenBootSearchTimer);
		goldenBootSearchTimer = setTimeout(async () => {
			goldenBootSearchLoading = true;
			goldenBootSearchError = '';
			try {
				const response = await api.searchGoldenBootPlayers(query);
				if (cancelled) return;
				goldenBootSearchResults = response.players;
				goldenBootSearchApiAvailable = response.apiAvailable;
			} catch (e: unknown) {
				if (cancelled) return;
				goldenBootSearchResults = [];
				goldenBootSearchError = (e as { message?: string })?.message ?? 'Could not search players.';
			} finally {
				if (!cancelled) goldenBootSearchLoading = false;
			}
		}, 280);

		return () => {
			cancelled = true;
			clearTimeout(goldenBootSearchTimer);
		};
	});

	function tname(id: string) {
		return teamDisplayName(fs.team(id));
	}
	function initials(name: string) {
		return name
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('');
	}
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
	function pickGoldenBoot(playerId: string) {
		if (fs.locked) return;
		fs.goldenBootPlayer = playerId;
		vibrate(15);
	}
	function sortGoldenBootPlayers(players: GoldenBootPlayer[]) {
		return [...players].sort((first, second) => {
			const firstRank = first.rank ?? 0;
			const secondRank = second.rank ?? 0;
			if ((firstRank === 0) !== (secondRank === 0)) return firstRank !== 0 ? -1 : 1;
			if (firstRank !== 0 && secondRank !== 0 && firstRank !== secondRank) return firstRank - secondRank;
			if (first.goals !== second.goals) return second.goals - first.goals;
			return first.name.localeCompare(second.name);
		});
	}
	function searchResultToPlayer(player: GoldenBootSearchResult): GoldenBootPlayer {
		return {
			id: player.id ?? '',
			name: player.name,
			teamId: player.teamId,
			teamName: player.teamName,
			photoUrl: player.photoUrl,
			goals: player.goals,
			assists: player.assists,
			rank: player.rank,
			eligible: player.eligible,
			seeded: false,
			syncedAt: fs.goldenBoot.updatedAt
		};
	}
	function upsertGoldenBootCandidate(player: GoldenBootPlayer) {
		const shortlist = sortGoldenBootPlayers([
			...fs.goldenBoot.shortlist.filter((current) => current.id !== player.id),
			player
		]);
		const shouldShowLeader =
			player.rank > 0 || fs.goldenBoot.leaders.some((current) => current.id === player.id);
		const leaders = shouldShowLeader
			? sortGoldenBootPlayers([
					...fs.goldenBoot.leaders.filter((current) => current.id !== player.id),
					player
				]).slice(0, 10)
			: fs.goldenBoot.leaders;
		fs.goldenBoot = {
			...fs.goldenBoot,
			shortlist,
			leaders,
			updatedAt: player.syncedAt || fs.goldenBoot.updatedAt
		};
	}
	async function chooseGoldenBootSearch(player: GoldenBootSearchResult) {
		if (fs.locked) return;
		goldenBootSearchPendingKey = player.key;
		goldenBootSearchError = '';
		try {
			const chosen =
				player.id && player.eligible
					? searchResultToPlayer(player)
					: (await api.ensureGoldenBootPlayer(player)).player;
			upsertGoldenBootCandidate(chosen);
			pickGoldenBoot(chosen.id);
			goldenBootSearchQuery = '';
			goldenBootSearchResults = [];
		} catch (e: unknown) {
			goldenBootSearchError =
				(e as { message?: string })?.message ?? 'Could not add this player.';
		} finally {
			goldenBootSearchPendingKey = '';
		}
	}
</script>

<div class="stickyhead" use:collapseOnScroll>
	<p class="kicker">Whole tournament</p>
	<div class="sh-expand">
		<div class="sh-inner">
			<h1>Golden Boot</h1>
			<p class="muted desc">
				Pick your Golden Boot winner. Group standings and the bracket are auto-derived from your
				match tips.
				{#if fs.locked}<b>Locked.</b>{:else}Locks at kickoff.{/if}
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
			<Lock size={16} /> The tournament has started — the Golden Boot pick is final.
		</div>
	{/if}

	<div class="gb-head">
		<p class="muted small">
			Pick a player or search for an outsider. A correct pick gives 15 points.
		</p>
		{#if fs.goldenBoot.updatedAt}
			<span class="cnt">Updated {updatedAt(fs.goldenBoot.updatedAt)}</span>
		{/if}
	</div>

	{#if goldenBootPick}
		<section class="card gb-pick">
			<Trophy size={20} />
			<span class="headshot-wrap">
				{#if goldenBootPick.photoUrl}
					<img class="headshot" src={goldenBootPick.photoUrl} alt="" loading="lazy" />
				{:else}
					<span class="headshot fallback">{initials(goldenBootPick.name)}</span>
				{/if}
			</span>
			<span class="gb-main">
				<i>Your Golden Boot pick</i>
				<b>{goldenBootPick.name}</b>
			</span>
			<Flag
				iso2={fs.team(goldenBootPick.teamId)?.iso2 ?? ''}
				code={fs.team(goldenBootPick.teamId)?.fifaCode ?? ''}
			/>
		</section>
	{/if}

	<section class="card gb-search">
		<div class="gb-search-head">
			<div>
				<h3>Search more players</h3>
				<p class="muted small">Add an outsider if they are not in the shortlist.</p>
			</div>
			{#if !goldenBootSearchApiAvailable}
				<span class="muted small api-note">Live API search unavailable</span>
			{/if}
		</div>
		<input
			class="gb-search-input"
			type="search"
			bind:value={goldenBootSearchQuery}
			placeholder="Search by player name…"
			aria-label="Search for a Golden Boot player"
			disabled={fs.locked}
		/>

		{#if goldenBootSearchError}
			<p class="error small">{goldenBootSearchError}</p>
		{:else if goldenBootSearchQuery.trim().length < 2}
			<p class="muted small">Type to search...</p>
		{:else if goldenBootSearchLoading}
			<p class="muted small">Searching…</p>
		{:else if goldenBootSearchResults.length === 0}
			<p class="muted small">
				{goldenBootSearchApiAvailable
					? 'No matching World Cup candidates found.'
					: 'No local candidates matched, and live API search is unavailable.'}
			</p>
		{:else}
			<div class="gb-search-results">
				{#each goldenBootSearchResults as player (player.key)}
					<button
						class="gb-search-result"
						class:picked={fs.goldenBootPlayer === (player.id ?? '')}
						disabled={fs.locked ||
							(goldenBootSearchPendingKey !== '' &&
								goldenBootSearchPendingKey !== player.key)}
						onclick={() => chooseGoldenBootSearch(player)}
					>
						<span class="headshot-wrap">
							{#if player.photoUrl}
								<img class="headshot" src={player.photoUrl} alt="" loading="lazy" />
							{:else}
								<span class="headshot fallback">{initials(player.name)}</span>
							{/if}
						</span>
						<span class="gb-main">
							<b>{player.name}</b>
							<span class="gb-search-meta">
								<Flag
									iso2={fs.team(player.teamId)?.iso2 ?? ''}
									code={fs.team(player.teamId)?.fifaCode ?? ''}
								/>
								<span>{player.teamName}</span>
								<span>·</span>
								<span>Goals {player.goals}</span>
							</span>
						</span>
						<span class="gb-search-action">
							{#if goldenBootSearchPendingKey === player.key}
								Adding…
							{:else if player.id && player.eligible}
								Pick
							{:else if player.existing}
								Add to list
							{:else}
								Add player
							{/if}
						</span>
					</button>
				{/each}
			</div>
		{/if}
	</section>

	<section class="card gb-list">
		<h3>Shortlist</h3>
		{#if fs.goldenBoot.shortlist.filter((p) => p.seeded).length === 0}
			<p class="muted small">No candidates yet.</p>
		{:else}
			<div class="gb-grid">
				{#each fs.goldenBoot.shortlist.filter((p) => p.seeded) as player (player.id)}
					<button
						class="gb-player"
						class:picked={fs.goldenBootPlayer === player.id}
						disabled={fs.locked}
						onclick={() => pickGoldenBoot(player.id)}
					>
						<span class="headshot-wrap">
							{#if player.photoUrl}
								<img class="headshot" src={player.photoUrl} alt="" loading="lazy" />
							{:else}
								<span class="headshot fallback">{initials(player.name)}</span>
							{/if}
						</span>
						<span class="gb-main">
							<b>{player.name}</b>
							<span class="gb-search-meta" style="margin-top: 0.15rem;">
								<Flag
									iso2={fs.team(player.teamId)?.iso2 ?? ''}
									code={fs.team(player.teamId)?.fifaCode ?? ''}
								/>
								<span>{player.teamName}</span>
							</span>
						</span>
						{#if fs.goldenBootPlayer === player.id}
							<span class="gb-player-status"><Check size={17} /></span>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</section>

	<section class="card gb-live">
		<div class="gb-live-head">
			<h3>Top scorers</h3>
			{#if fs.goldenBoot.updatedAt}
				<p class="muted small gb-updated">Updated {updatedAt(fs.goldenBoot.updatedAt)}</p>
			{/if}
		</div>
		<table class="gb-table">
			<thead>
				<tr>
					<th>#</th>
					<th>Player</th>
					<th>Team</th>
					<th class="num">Goals</th>
				</tr>
			</thead>
			<tbody>
				{#each goldenBootLeaders as player (player.id)}
					<tr class:picked={fs.goldenBootPlayer === player.id}>
						<td>{player.rank || '–'}</td>
						<td>
							<span class="gb-row-player">
								{#if player.photoUrl}
									<img class="mini-headshot" src={player.photoUrl} alt="" loading="lazy" />
								{:else}
									<span class="mini-headshot fallback">{initials(player.name)}</span>
								{/if}
								<b>{player.name}</b>
							</span>
						</td>
						<td>
							<span class="gb-team">
								<Flag
									iso2={fs.team(player.teamId)?.iso2 ?? ''}
									code={fs.team(player.teamId)?.fifaCode ?? ''}
								/>
								<span class="tm-full">{player.teamName}</span>
								<span class="tm-short"
									>{fs.team(player.teamId)?.fifaCode ?? player.teamName}</span
								>
							</span>
						</td>
						<td class="num digits">{player.goals}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>

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

<style>
	h1 {
		margin: 0.25rem 0 0.2rem;
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
	.cnt {
		font-family: var(--font-mono);
		font-weight: 700;
		padding: 0.2rem 0.6rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		color: var(--muted);
		white-space: nowrap;
	}
	.ind {
		display: inline-grid;
		place-items: center;
	}
	.ind.ok {
		color: var(--success);
	}
	.ind.no {
		color: var(--danger);
	}
	.gb-head {
		display: flex;
		align-items: flex-start;
		gap: 1rem;
		margin-bottom: 0.7rem;
	}
	.gb-head .small {
		flex: 1;
		margin: 0;
	}
	.gb-pick {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		border-color: color-mix(in srgb, var(--gold) 42%, var(--border));
	}
	.gb-main {
		display: grid;
		gap: 0.15rem;
		min-width: 0;
		flex: 1;
	}
	.gb-main b,
	.gb-main i {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.gb-main i {
		font-style: normal;
		font-size: 0.78rem;
		color: var(--muted);
	}
	.gb-list h3,
	.gb-search h3,
	.gb-live h3 {
		margin: 0 0 0.7rem;
	}
	.gb-search {
		display: grid;
		gap: 0.75rem;
	}
	.gb-search-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}
	.gb-search-head .small {
		margin: 0;
	}
	.gb-search-input {
		width: 100%;
		padding: 0.8rem 0.9rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
	}
	.gb-search-input::placeholder {
		color: var(--muted);
	}
	.gb-search-results {
		display: grid;
		gap: 0.55rem;
	}
	.gb-search-result {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.7rem;
		padding: 0.65rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		text-align: left;
	}
	.gb-search-result.picked {
		border-color: color-mix(in srgb, var(--success) 48%, var(--border));
		background: color-mix(in srgb, var(--success) 9%, var(--surface-2));
	}
	.gb-search-result:disabled {
		opacity: 0.88;
	}
	.gb-search-meta {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		color: var(--muted);
		font-size: 0.8rem;
		min-width: 0;
	}
	.gb-search-meta span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.gb-search-action {
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		color: var(--muted);
		white-space: nowrap;
	}
	.api-note {
		text-align: right;
	}
	.gb-updated {
		margin: 0 0 0.6rem;
		text-align: right;
	}
	.gb-live-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}
	.gb-live-head h3 {
		margin-bottom: 0.6rem;
	}
	.gb-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
		gap: 0.65rem;
	}
	.gb-player {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		min-height: 56px;
		padding: 0.85rem 0.5rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		text-align: center;
		position: relative;
	}
	.gb-player .headshot-wrap,
	.gb-player .headshot {
		width: 54px;
		height: 54px;
	}
	.gb-player .gb-main {
		display: flex;
		flex-direction: column;
		align-items: center;
		width: 100%;
	}
	.gb-player-status {
		position: absolute;
		top: 0.4rem;
		right: 0.4rem;
	}
	.gb-player.picked {
		border-color: color-mix(in srgb, var(--success) 48%, var(--border));
		background: color-mix(in srgb, var(--success) 9%, var(--surface-2));
	}
	.gb-player:disabled {
		cursor: default;
		opacity: 0.88;
	}
	.headshot-wrap,
	.headshot,
	.mini-headshot {
		display: inline-grid;
		place-items: center;
		border-radius: 50%;
		background: var(--surface);
		border: 1px solid var(--border);
		object-fit: cover;
		flex: none;
	}
	.headshot,
	.headshot-wrap {
		width: 42px;
		height: 42px;
	}
	.mini-headshot {
		width: 28px;
		height: 28px;
		font-size: 0.65rem;
	}
	.fallback {
		font-family: var(--font-display);
		font-weight: 800;
		color: var(--muted);
	}
	.gb-table {
		width: 100%;
		border-collapse: collapse;
	}
	.gb-table th,
	.gb-table td {
		padding: 0.55rem 0.35rem;
		border-bottom: 1px solid var(--border);
		text-align: left;
	}
	.gb-table th {
		color: var(--muted);
		font-size: 0.78rem;
		font-weight: 700;
	}
	.gb-table tr.picked td {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
	}
	.gb-row-player,
	.gb-team {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		min-width: 0;
	}
	.gb-row-player b,
	.gb-team {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.tm-short {
		display: none;
	}
	@media (max-width: 500px) {
		.tm-full {
			display: none;
		}
		.tm-short {
			display: inline;
		}
	}
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
	:global(:root[data-theme='worldcup']) .gb-pick,
	:global(:root[data-theme='worldcup']) .gb-search,
	:global(:root[data-theme='worldcup']) .gb-list,
	:global(:root[data-theme='worldcup']) .gb-live,
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
	:global(:root[data-theme='worldcup']) .gb-pick::before,
	:global(:root[data-theme='worldcup']) .gb-search::before,
	:global(:root[data-theme='worldcup']) .gb-list::before,
	:global(:root[data-theme='worldcup']) .gb-live::before,
	:global(:root[data-theme='worldcup']) .lockbar::before {
		display: none;
	}
	:global(:root[data-theme='worldcup']) .gb-search-input,
	:global(:root[data-theme='worldcup']) .gb-search-result,
	:global(:root[data-theme='worldcup']) .savestat {
		background: color-mix(in srgb, var(--surface-2) 78%, transparent);
		border-color: color-mix(in srgb, var(--accent) 12%, var(--border));
	}
</style>
