<script lang="ts">
	import DeadlineCountdown from '$lib/components/DeadlineCountdown.svelte';
	import { forecastStore as fs } from '$lib/forecast.svelte';
	import { tipsStore } from '$lib/tips.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import PlayerPicker from '$lib/components/PlayerPicker.svelte';
	import { Lock, Check } from '@lucide/svelte';
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

	function initials(name: string) {
		return name
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('');
	}

	let goldenBootLeaders = $derived(
		fs.goldenBoot.leaders.length > 0 ? fs.goldenBoot.leaders : fs.goldenBoot.shortlist.slice(0, 10)
	);

	let gkPlayers = $derived(fs.allPlayers.filter((p) => p.position === 'GK'));
</script>

<div class="stickyhead" use:collapseOnScroll>
	<p class="kicker">Whole tournament</p>
	<div class="sh-expand">
		<div class="sh-inner">
			<h1>Tournament Awards</h1>
			<p class="muted desc">
				Pick the award winners. Group standings and the bracket are auto-derived from your match
				tips.
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
			<Lock size={16} /> The tournament has started — award picks are final.
		</div>
	{/if}

	<!-- Golden Boot -->
	<section class="card award-card">
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
			<PlayerPicker players={fs.allPlayers} bind:value={fs.goldenBootName} locked={fs.locked} placeholder="Search Golden Boot candidate…" />
		{/if}
	</section>

	<!-- Golden Ball -->
	<section class="card award-card">
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
			<PlayerPicker players={fs.allPlayers} bind:value={fs.goldenBallPlayer} locked={fs.locked} placeholder="Search Golden Ball candidate…" />
		{/if}
	</section>

	<!-- Golden Glove -->
	<section class="card award-card">
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
	</section>

	<!-- Best Young Player -->
	<section class="card award-card">
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
			<PlayerPicker players={fs.allPlayers} bind:value={fs.bestYoungPlayer} locked={fs.locked} placeholder="Search young player…" />
		{/if}
	</section>

	<!-- Most Assists -->
	<section class="card award-card">
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
			<PlayerPicker players={fs.allPlayers} bind:value={fs.mostAssistsPlayer} locked={fs.locked} placeholder="Search assists leader…" />
		{/if}
	</section>

	<!-- Live top scorers (for reference) -->
	{#if goldenBootLeaders.length > 0}
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
						<tr>
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
	{/if}

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
	.award-card {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
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
	.gb-live h3 {
		margin-bottom: 0.6rem;
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
	.gb-updated {
		margin: 0 0 0.6rem;
		text-align: right;
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
	.mini-headshot {
		display: inline-grid;
		place-items: center;
		border-radius: 50%;
		background: var(--surface);
		border: 1px solid var(--border);
		object-fit: cover;
		flex: none;
		width: 28px;
		height: 28px;
		font-size: 0.65rem;
	}
	.fallback {
		font-family: var(--font-display);
		font-weight: 800;
		color: var(--muted);
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
	:global(:root[data-theme='worldcup']) .award-card,
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
	:global(:root[data-theme='worldcup']) .award-card::before,
	:global(:root[data-theme='worldcup']) .gb-live::before,
	:global(:root[data-theme='worldcup']) .lockbar::before {
		display: none;
	}
	:global(:root[data-theme='worldcup']) .savestat {
		background: color-mix(in srgb, var(--surface-2) 78%, transparent);
		border-color: color-mix(in srgb, var(--accent) 12%, var(--border));
	}
</style>
