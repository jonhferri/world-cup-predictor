<script lang="ts">
	import { onDestroy } from 'svelte';
	import {
		tipsStore,
		isLocked,
		teamsResolved,
		groupStageBucket,
		type Match,
		type FriendTip,
		type TipComponents,
		type Player
	} from '$lib/tips.svelte';
	import { vibrate } from '$lib/haptics';
	import Flag from './Flag.svelte';
	import Stepper from './Stepper.svelte';
	import { teamDisplayName } from '$lib/teamNames';
	import { Lock, ChevronDown, Check, Users } from '@lucide/svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';
	import OddsBadge from './OddsBadge.svelte';
	import { api, type CrowdDistribution } from '$lib/api';

	const FRIENDS_PREVIEW_COUNT = 10;

	let { match }: { match: Match } = $props();

	let locked = $derived(isLocked(match));
	let resolved = $derived(teamsResolved(match));
	let home = $derived(tipsStore.team(match.homeTeam));
	let away = $derived(tipsStore.team(match.awayTeam));
	let existing = $derived(tipsStore.tips[match.id]);
	let isKO = $derived(match.stage !== 'group');
	let played = $derived(match.status === 'finished' || !!match.finalizedAt);
	let live = $derived(match.status === 'live');
	let pts = $derived(tipsStore.scores[match.id]);
	let matchOdds = $derived(tipsStore.odds[match.id]);
	let showDecimal = $state(false);
	let showOdds = $derived(!locked && !played && !live && !!matchOdds);
	let canEdit = $derived(!locked && resolved);
	let open = $state(false);
	let bodyVisible = $derived(open || canEdit);
	let advancedName = $derived(
		isKO && match.advancer ? teamDisplayName(tipsStore.team(match.advancer)) : ''
	);

	// Turbo state.
	let stageGroupKey = $derived(match.stage === 'group' ? groupStageBucket(match.roundLabel) : match.stage);
	let isAutoTurbo = $derived(match.stage === 'FINAL' || match.stage === '3RD');

	// Editable working copy.
	let ftH = $state(0);
	let ftA = $state(0);
	let etH = $state(0);
	let etA = $state(0);
	let pen = $state(''); // penalty winner team id
	let firstTeam = $state('');
	let firstPlayer = $state('');
	let turbo = $state(false);
	let busy = $state(false);
	let msg = $state('');
	let savedOk = $state(false);
	let saveToastRun = $state(0);
	let saveToastTimer: ReturnType<typeof setTimeout> | null = null;
	const t = $derived(strings[language.resolved]);

	// Players for first-scorer pick — loaded once per match when the card is editable.
	let players = $state<Player[]>([]);
	let playerSearch = $state('');
	let playerDropdownOpen = $state(false);
	let filteredPlayers = $derived(
		playerSearch.trim().length > 0
			? players.filter((p) =>
					p.name.toLowerCase().includes(playerSearch.trim().toLowerCase())
				)
			: players
	);

	$effect(() => {
		if (!canEdit || !match.homeTeam || !match.awayTeam) return;
		tipsStore
			.playersForTeams([match.homeTeam, match.awayTeam])
			.then((list) => (players = list))
			.catch(() => {});
	});

	// Seed the editor from the saved tip whenever it changes.
	$effect(() => {
		const t = tipsStore.tips[match.id];
		ftH = t?.ftHome ?? 0;
		ftA = t?.ftAway ?? 0;
		etH = t?.etHome ?? 0;
		etA = t?.etAway ?? 0;
		pen = t?.penWinner ?? '';
		firstTeam = t?.firstTeam ?? '';
		firstPlayer = t?.firstPlayer ?? '';
		playerSearch = t?.firstPlayer ?? '';
		turbo = t?.turbo ?? false;
	});

	let ftTie = $derived(isKO && ftH === ftA);
	let etTie = $derived(ftTie && etH === etA);

	// Keep ET >= FT (cumulative) as the user edits FT.
	$effect(() => {
		if (etH < ftH) etH = ftH;
		if (etA < ftA) etA = ftA;
	});

	let advancerId = $derived(
		!isKO
			? ''
			: ftH !== ftA
				? ftH > ftA
					? match.homeTeam
					: match.awayTeam
				: etH !== etA
					? etH > etA
						? match.homeTeam
						: match.awayTeam
					: pen
	);
	let advancerName = $derived(
		advancerId ? teamDisplayName(tipsStore.team(advancerId), '—') : ''
	);

	const kickoff = $derived(
		new Date(match.kickoff).toLocaleString(language.locale, {
			weekday: 'short',
			day: 'numeric',
			month: 'short',
			hour: '2-digit',
			minute: '2-digit'
		})
	);

	$effect(() => {
		const t = tipsStore.tips[match.id];
		const unchanged =
			(t?.ftHome ?? 0) === ftH &&
			(t?.ftAway ?? 0) === ftA &&
			(t?.etHome ?? 0) === etH &&
			(t?.etAway ?? 0) === etA &&
			(t?.penWinner ?? '') === pen &&
			(t?.firstTeam ?? '') === firstTeam &&
			(t?.firstPlayer ?? '') === firstPlayer &&
			(t?.turbo ?? false) === turbo;
		if (!unchanged) {
			savedOk = false;
		}
	});

	function clearSaveToastTimer() {
		if (saveToastTimer !== null) {
			clearTimeout(saveToastTimer);
			saveToastTimer = null;
		}
	}

	function triggerSaveFeedback() {
		clearSaveToastTimer();
		saveToastRun += 1;
		savedOk = true;
		vibrate(14);
		saveToastTimer = setTimeout(() => {
			savedOk = false;
			saveToastTimer = null;
		}, 1400);
	}

	onDestroy(() => {
		clearSaveToastTimer();
	});

	async function save() {
		msg = '';
		clearSaveToastTimer();
		savedOk = false;
		busy = true;
		try {
			await tipsStore.save({
				id: existing?.id,
				match: match.id,
				ftHome: ftH,
				ftAway: ftA,
				etHome: etH,
				etAway: etA,
				penWinner: pen,
				advancer: '',
				firstTeam,
				firstPlayer,
				turbo
			});
			triggerSaveFeedback();
		} catch (e: unknown) {
			msg =
				(e as { message?: string })?.message ??
				language.text('Kunne ikke lagre tipset.', 'Kunne ikkje lagre tipset.', 'Could not save tip.');
		} finally {
			busy = false;
		}
	}

	// Friends' picks (only available after kickoff) — toggles open/closed.
	let friends = $state<FriendTip[] | null>(null);
	let friendsBusy = $state(false);
	let showAllFriends = $state(false);
	let sortedFriends = $derived.by<FriendTip[]>(() => {
		const rows = [...(friends ?? [])];
		rows.sort((left, right) => {
			if (left.isMe !== right.isMe) return left.isMe ? -1 : 1;
			if (left.points !== right.points) return right.points - left.points;
			return left.name.localeCompare(right.name, language.locale);
		});
		return rows;
	});
	let hiddenFriendsCount = $derived(
		Math.max(sortedFriends.length - FRIENDS_PREVIEW_COUNT, 0)
	);
	let visibleFriends = $derived(
		showAllFriends ? sortedFriends : sortedFriends.slice(0, FRIENDS_PREVIEW_COUNT)
	);
	async function toggleFriends() {
		if (friends !== null) {
			friends = null;
			showAllFriends = false;
			return;
		}
		friendsBusy = true;
		try {
			friends = await tipsStore.friends(match.id);
			showAllFriends = false;
		} catch {
			friends = [];
			showAllFriends = false;
		} finally {
			friendsBusy = false;
		}
	}

	// Crowd prediction (global). Fetched lazily after kickoff, once per open.
	let crowd = $state<CrowdDistribution | null>(null);
	let crowdLoaded = $state(false);
	$effect(() => {
		// Re-fetch when match id changes or lock state flips to true.
		if (!locked) {
			crowd = null;
			crowdLoaded = false;
			return;
		}
		if (crowdLoaded) return;
		crowdLoaded = true;
		api.matchCrowd(match.id)
			.then((c) => {
				crowd = c;
			})
			.catch(() => {
				crowd = null;
			});
	});
	const crowdReady = $derived(
		!!crowd && crowd.locked && !!crowd.outcomes && (crowd.total ?? 0) > 0
	);

	// Static confetti particle definitions — varied angles, distances, hues.
	const confettiParticles = [
		{ dx: -48, dy: -52, hue: 48,  delay: 0   },
		{ dx:  32, dy: -64, hue: 142, delay: 30  },
		{ dx:  62, dy: -28, hue: 48,  delay: 55  },
		{ dx: -62, dy: -18, hue: 220, delay: 20  },
		{ dx:  18, dy: -68, hue: 48,  delay: 45  },
		{ dx: -28, dy: -58, hue: 0,   delay: 10  },
		{ dx:  52, dy: -44, hue: 48,  delay: 65  },
		{ dx: -52, dy: -32, hue: 142, delay: 35  },
		{ dx:  12, dy: -48, hue: 340, delay: 15  },
		{ dx: -18, dy: -72, hue: 48,  delay: 50  },
		{ dx:  44, dy: -58, hue: 220, delay: 25  },
		{ dx: -38, dy: -46, hue: 0,   delay: 40  },
	] as const;

	function label(side: 'home' | 'away') {
		const t = side === 'home' ? home : away;
		if (t) return { name: teamDisplayName(t), iso2: t.iso2, code: t.fifaCode };
		const raw = side === 'home' ? match.homeLabel : match.awayLabel;
		return { name: raw, iso2: '', code: raw };
	}
	let H = $derived(label('home'));
	let A = $derived(label('away'));
</script>

<div class="tc card" class:locked>
	<button
		class="head"
		class:direct={canEdit}
		onclick={() => {
			if (!canEdit) open = !open;
		}}
		aria-expanded={bodyVisible}
		aria-disabled={canEdit}
	>
		<div class="teams">
			<span class="t">
				<Flag iso2={H.iso2} code={H.code} /> <span class="tn">{H.name}</span>
			</span>
			<span class="score digits">
				{#if played || live}
					<b>{match.ftHome}</b><span class="cln">:</span><b>{match.ftAway}</b>
				{:else if existing}
					<span class="pred">{existing.ftHome}<span class="cln">:</span>{existing.ftAway}</span>
				{:else}
					<span class="muted">–:–</span>
				{/if}
			</span>
			<span class="t right">
				<span class="tn">{A.name}</span> <Flag iso2={A.iso2} code={A.code} />
			</span>
		</div>
		{#if showOdds}
			<OddsBadge odds={matchOdds} source={tipsStore.oddsSource} bind:showDecimal />
		{/if}
		<div class="meta">
			<span class="muted"
				>{match.stage === 'group'
					? `${t.tipCard.stageGroup} ${match.groupLetter} · ${match.roundLabel}`
					: match.roundLabel} · {kickoff}</span
			>
			<span class="spacer"></span>
			{#if isAutoTurbo}
				<span class="turbo-pill auto" title="Match automatically doubled">⚡ 2×</span>
			{:else if locked && existing?.turbo}
				<span class="turbo-pill on" title="Turbo active – points doubled">⚡ 2×</span>
			{:else if !locked && (!tipsStore.turboedStageGroups().has(stageGroupKey) || existing?.turbo)}
				<button
					class="turbo-btn"
					class:on={turbo}
					onclick={() => { turbo = !turbo; }}
					title={turbo ? 'Turbo on – click to remove (save to confirm)' : 'Use 2× turbo on this match'}
				>⚡</button>
			{/if}
			{#if played}
				<span class="pill done" class:perfect={pts === 6}>
					FT
					{#if pts !== undefined}
						<b class="ptv" class:ok={pts > 0}>
							{#if pts === 6}<span class="star">★</span>{/if}
							{pts > 0 ? '+' : ''}{pts}&thinsp;p
						</b>
					{/if}
				</span>
			{:else if live}
					<span class="pill livep"><span class="dot"></span> {t.tipCard.live}</span>
			{:else if locked}
					<span class="pill"><Lock size={12} /> {t.tipCard.locked}</span>
			{:else if existing}
					<span class="pill ok"><Check size={12} /> {t.tipCard.result}</span>
			{:else if canEdit}
					<span class="pill missing">{t.tipCard.missing}</span>
			{/if}
			{#if !canEdit}
				<ChevronDown size={16} class="cv {open ? 'up' : ''}" />
			{/if}
		</div>
		{#if pts === 6 && played}
			{#key pts}
				<div class="confetti" aria-hidden="true">
					{#each confettiParticles as p}
						<span
							class="cp"
							style="--dx:{p.dx}px; --dy:{p.dy}px; --hue:{p.hue}; --delay:{p.delay}ms"
						></span>
					{/each}
				</div>
			{/key}
		{/if}
	</button>

	{#if bodyVisible}
		<div class="body">
			{#if isKO && !resolved}
					<p class="muted">{t.tipCard.loading}</p>
			{:else if locked}
				{#if played && advancedName}
					<p class="resline muted">
						{t.tipCard.lockedResult} <b>{match.ftHome}:{match.ftAway}</b> · {t.tipCard.goThrough}:
						<b>{advancedName}</b>
					</p>
				{/if}
				{#if existing}
					<div class="yourtip" class:scored={played}>
						<span class="ylabel">{t.tipCard.result}</span>
						<span class="yscore digits"
							>{existing.ftHome}<span class="cln">:</span>{existing.ftAway}</span
						>
						{#if isKO && existing.advancer}
							<span class="yadv"
								>→ {teamDisplayName(tipsStore.team(existing.advancer), '—')}</span
							>
						{/if}
						<span class="spacer"></span>
						{#if played && pts !== undefined}
							<span class="ypts" class:ok={pts > 0} class:perfect={pts === 6}>
								{#if pts === 6}<span class="star">★</span>{/if}
								{pts > 0 ? '+' : ''}{pts} p
							</span>
						{/if}
					</div>
				{:else}
						<p class="muted">{t.tipCard.noTipLocked}</p>
				{/if}
				{#if existing?.firstTeam || existing?.firstPlayer}
					<div class="first-scorer-locked">
						{#if existing.firstTeam}
							{@const ft = tipsStore.team(existing.firstTeam)}
							<span class="fs-label">1st team:</span>
							<span class="fs-val">
								{#if ft}<Flag iso2={ft.iso2} code={ft.fifaCode} />{/if}
								{ft ? teamDisplayName(ft) : existing.firstTeam}
							</span>
						{/if}
						{#if existing.firstPlayer}
							<span class="fs-label">1st scorer:</span>
							<span class="fs-val">{existing.firstPlayer}</span>
						{/if}
					</div>
				{/if}

				<button
					class="btn secondary friendsbtn"
					class:on={friends !== null}
					onclick={toggleFriends}
					disabled={friendsBusy}
				>
					<Users size={16} />
						{friends !== null ? t.tipCard.hideFriendTips : t.tipCard.showFriendTips}
				</button>
				{#if friends}
					{#if friends.length === 0}
							<p class="muted small">{t.tipCard.noFriendTips}</p>
					{:else}
						<table class="friends">
							<thead>
								<tr>
									<th></th>
									<th class="ftip">{language.text('Tips', 'Tips', 'Tip')}</th>
									<th class="ftip">⚽ 1st</th>
									<th class="fpts">{language.text('P', 'P', 'Pts')}</th>
								</tr>
							</thead>
							<tbody>
								{#each visibleFriends as f (f.userId)}
									{@const c = f.components}
									{@const ftExact = c && (isKO ? c.koFtExact > 0 : c.exact > 0)}
									{@const ftTendency = c && (isKO ? c.koFtTendency > 0 : c.tendency > 0) && !ftExact}
									<tr class:fme={f.isMe}>
										<td class="fname">{f.name}{#if f.turbo} <span class="fturbo" title="Turbo active">⚡</span>{/if}</td>
										<td class="ftip">
											<span
												class="fscore"
												class:fscore-exact={ftExact}
												class:fscore-tendency={ftTendency}
											>
												{f.ftHome}:{f.ftAway}
											</span>
											{#if f.advancer}
												<span class="fadv">→ {teamDisplayName(tipsStore.team(f.advancer))}</span>
											{/if}
											{#if c}
												<span class="fcomps">
													{#if isKO}
														{#if c.koAdvancer > 0}<span class="fcomp fc-tendency" title="Correct team to advance">→</span>{/if}
														{#if c.koFtTendency > 0}<span class="fcomp fc-tendency" title="Correct FT outcome">W</span>{/if}
														{#if c.koFtGoalDiff > 0}<span class="fcomp fc-diff" title="Correct FT goal difference">Δ</span>{/if}
														{#if c.koFtExactHome > 0 || c.koFtExactAway > 0}<span class="fcomp fc-goals" title="Exact FT goals">G</span>{/if}
														{#if c.koFtExact > 0}<span class="fcomp fc-exact" title="Exact FT score">=</span>{/if}
														{#if c.koEtTendency > 0 || c.koEtExact > 0 || c.koEtGoalDiff > 0 || c.koEtExactHome > 0 || c.koEtExactAway > 0}<span class="fcomp fc-et" title="ET scoring">ET</span>{/if}
													{:else}
														{#if c.tendency > 0}<span class="fcomp fc-tendency" title="Correct outcome">W</span>{/if}
														{#if c.exact > 0}<span class="fcomp fc-exact" title="Exact score">=</span>{/if}
														{#if c.totalGoals > 0}<span class="fcomp fc-goals" title="Exact goals (home or away)">G</span>{/if}
														{#if c.goalDiff > 0}<span class="fcomp fc-diff" title="Correct goal difference">Δ</span>{/if}
													{/if}
												</span>
											{/if}
										</td>
										<td class="ftip fscorer">
											{#if f.firstTeam}
												<span class="fts" class:fts-ok={c && c.firstTeamScorer > 0}>{tipsStore.team(f.firstTeam)?.fifaCode ?? teamDisplayName(tipsStore.team(f.firstTeam))}</span>
											{/if}
											{#if f.firstPlayer}
												<span class="fps" class:fps-ok={c && c.firstPlayerScorer > 0}>{f.firstPlayer.split(' ').pop()}</span>
											{/if}
											{#if !f.firstTeam && !f.firstPlayer}
												<span class="muted">—</span>
											{/if}
										</td>
										<td class="fpts">
											{#if f.points >= 0}
												<span class:fok={f.points > 0} class:fperfect={f.points === 6}>
													{f.points > 0 ? '+' : ''}{f.points}
												</span>
											{:else}
												<span class="muted">—</span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
						{#if sortedFriends.length > FRIENDS_PREVIEW_COUNT}
							<div class="friends-actions">
								<button class="btn secondary morefriends" onclick={() => (showAllFriends = !showAllFriends)}>
									{#if showAllFriends}
										{language.text('Vis færre', 'Vis færre', 'Show fewer')}
									{:else}
										{language.text(`Vis ${hiddenFriendsCount} flere`, `Vis ${hiddenFriendsCount} fleire`, `Show ${hiddenFriendsCount} more`)}
									{/if}
								</button>
							</div>
						{/if}
					{/if}
				{/if}
				{#if crowdReady && crowd?.outcomes}
					{@const o = crowd.outcomes}
					<div
						class="crowd"
						data-testid="crowd-bar"
						aria-label={`${t.tipCard.crowdTitle}: ${o.home.pct}% ${t.tipCard.crowdHome}, ${o.draw.pct}% ${t.tipCard.crowdDraw}, ${o.away.pct}% ${t.tipCard.crowdAway}`}
					>
						<div class="crowd-head">
							<span class="crowd-title">{t.tipCard.crowdTitle}</span>
							<span class="muted small">{crowd.total} {t.tipCard.crowdTotal}</span>
						</div>
						<div class="crowd-bar" role="img" aria-hidden="true">
							{#if o.home.pct > 0}
								<div class="seg seg-home" style="width: {o.home.pct}%">
									{#if o.home.pct >= 12}<span>{o.home.pct}%</span>{/if}
								</div>
							{/if}
							{#if !crowd.isKO && o.draw.pct > 0}
								<div class="seg seg-draw" style="width: {o.draw.pct}%">
									{#if o.draw.pct >= 12}<span>{o.draw.pct}%</span>{/if}
								</div>
							{/if}
							{#if o.away.pct > 0}
								<div class="seg seg-away" style="width: {o.away.pct}%">
									{#if o.away.pct >= 12}<span>{o.away.pct}%</span>{/if}
								</div>
							{/if}
						</div>
						<ul class="crowd-legend muted small">
							<li><span class="dot seg-home"></span>{H.name} · {o.home.count} ({o.home.pct}%)</li>
							{#if !crowd.isKO}
								<li><span class="dot seg-draw"></span>{t.tipCard.crowdDraw} · {o.draw.count} ({o.draw.pct}%)</li>
							{/if}
							<li><span class="dot seg-away"></span>{A.name} · {o.away.count} ({o.away.pct}%)</li>
						</ul>
					</div>
				{/if}
			{:else}
				<!-- Editable -->
				<div class="enter">
					<Stepper bind:value={ftH} />
					<span class="sep">:</span>
					<Stepper bind:value={ftA} />
				</div>

				{#if ftTie}
						<div class="phase">{language.text('Etter ekstraomganger', 'Etter ekstraomgangar', 'After extra time')}</div>
					<div class="enter">
						<Stepper bind:value={etH} min={ftH} />
						<span class="sep">:</span>
						<Stepper bind:value={etA} min={ftA} />
					</div>
				{/if}

				{#if etTie}
						<div class="phase">{language.text('Straffer - hvem går videre?', 'Straffar - kven går vidare?', 'Penalties - who goes through?')}</div>
					<div class="pens">
						<button
							class="pen"
							class:sel={pen === match.homeTeam}
							onclick={() => (pen = match.homeTeam)}
						>
							{teamDisplayName(home)}
						</button>
						<button
							class="pen"
							class:sel={pen === match.awayTeam}
							onclick={() => (pen = match.awayTeam)}
						>
							{teamDisplayName(away)}
						</button>
					</div>
				{/if}

				{#if isKO && advancerName}
					<p class="adv muted">{t.tipCard.goThrough}: <b>{advancerName}</b></p>
				{/if}

				<!-- First scorer picks -->
				{#if players.length > 0 || match.homeTeam}
					<div class="first-scorer-section">
						<div class="fs-col">
							<p class="phase">First team to score</p>
							<div class="first-team-btns">
								<button
									class="first-team-btn"
									class:sel={firstTeam === match.homeTeam}
									onclick={() => { firstTeam = firstTeam === match.homeTeam ? '' : match.homeTeam; }}
								>
									<Flag iso2={H.iso2} code={H.code} />{H.name}
								</button>
								<button
									class="first-team-btn"
									class:sel={firstTeam === match.awayTeam}
									onclick={() => { firstTeam = firstTeam === match.awayTeam ? '' : match.awayTeam; }}
								>
									<Flag iso2={A.iso2} code={A.code} />{A.name}
								</button>
							</div>
						</div>
						{#if players.length > 0}
							<div class="fs-col">
								<p class="phase">First player to score</p>
								<div class="player-picker">
									<input
										class="player-input"
										type="text"
										placeholder="Search player…"
										bind:value={playerSearch}
										onfocus={() => (playerDropdownOpen = true)}
										onblur={() => setTimeout(() => (playerDropdownOpen = false), 160)}
										oninput={() => { if (playerSearch !== firstPlayer) firstPlayer = ''; }}
									/>
									{#if playerSearch && playerSearch === firstPlayer}
										<button
											class="player-clear"
											onclick={() => { firstPlayer = ''; playerSearch = ''; }}
											aria-label="Clear player"
										>×</button>
									{/if}
									{#if playerDropdownOpen && filteredPlayers.length > 0}
										<ul class="player-dropdown">
											{#each filteredPlayers as p (p.id)}
												<li>
													<button
														onmousedown={(e) => e.preventDefault()}
														onclick={() => {
															firstPlayer = p.name;
															playerSearch = p.name;
															firstTeam = p.teamId;
															playerDropdownOpen = false;
														}}
														class:selected={firstPlayer === p.name}
													>
														<span class="p-name">
															{#if tipsStore.team(p.teamId)}{@const ptm = tipsStore.team(p.teamId)}<Flag iso2={ptm.iso2} code={ptm.fifaCode} />{/if}
															{p.name}
														</span>
														<span class="p-pos muted">{p.position}</span>
													</button>
												</li>
											{/each}
										</ul>
									{/if}
								</div>
							</div>
						{/if}
					</div>
				{/if}

				{#if msg}<p class="error">{msg}</p>{/if}
				<div class="save-status">
					<button class="save-mini" onclick={save} disabled={busy}>
						{#if busy}{t.tipCard.loading}{:else}{t.tipCard.save}{/if}
					</button>
					<div class="save-indicator" aria-live="polite">
						{#if savedOk}
							{#key saveToastRun}
								<span class="ok-toast"><Check size={16} /> {t.tipCard.saved}</span>
							{/key}
						{/if}
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.save-status {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 36px;
		margin-top: 0.5rem;
	}
	.save-mini {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-height: 36px;
		padding: 0.5rem 0.9rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-weight: 700;
		cursor: pointer;
	}
	.save-mini:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
	.save-indicator {
		position: absolute;
		left: 50%;
		bottom: calc(100% - 0.1rem);
		transform: translateX(-50%);
		pointer-events: none;
	}
	.ok-toast {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0.38rem 0.72rem;
		border-radius: var(--radius-pill);
		border: 1px solid color-mix(in srgb, var(--success) 28%, var(--border));
		background: color-mix(in srgb, var(--success) 12%, var(--surface));
		box-shadow: var(--shadow-pop);
		color: var(--success);
		font-weight: 600;
		white-space: nowrap;
		animation: save-toast 1.4s cubic-bezier(0.22, 1, 0.36, 1) forwards;
	}
	@keyframes save-toast {
		0% {
			opacity: 0;
			transform: translateY(0.45rem) scale(0.96);
		}
		15% {
			opacity: 1;
			transform: translateY(0) scale(1);
		}
		72% {
			opacity: 1;
			transform: translateY(-0.12rem) scale(1);
		}
		100% {
			opacity: 0;
			transform: translateY(-1.05rem) scale(0.98);
		}
	}
	.tc {
		padding: 0;
		overflow: hidden;
	}
	.head {
		position: relative;
		width: 100%;
		background: none;
		border: none;
		color: var(--text);
		text-align: left;
		padding: 0.85rem 1rem;
		display: block;
		cursor: pointer;
	}
	.head.direct {
		cursor: default;
	}
	.teams {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.5rem;
	}
	.t {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		min-width: 0;
	}
	.t.right {
		justify-content: flex-end;
	}
	.tn {
		font-weight: 700;
		font-size: 1.05rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	@media (max-width: 500px) {
		.tn {
			font-size: 0.9rem;
			white-space: normal;
			overflow: visible;
			text-overflow: clip;
			word-break: break-word;
		}
	}
	.t.right {
		justify-content: flex-end;
	}
	.tn {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-weight: 600;
	}
	.score b {
		font-size: 1.1rem;
	}
	.score {
		padding: 0 0.4rem;
	}
	.meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.5rem;
		font-size: 0.8rem;
	}
	:global(.tc .cv) {
		transition: transform 0.15s ease;
		color: var(--muted);
	}
	:global(.tc .cv.up) {
		transform: rotate(180deg);
	}
	.pill.ok {
		color: var(--success);
		border-color: var(--success);
	}
	.pill.missing {
		color: var(--warning);
		border-color: color-mix(in srgb, var(--warning) 42%, var(--border));
	}
	.body {
		padding: 0.25rem 1rem 1rem;
		border-top: 1px solid var(--border);
	}
	.enter {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1.5rem;
		margin: 1.2rem 0;
	}
	.sep {
		font-weight: 800;
		opacity: 0.3;
		font-size: 1.5rem;
	}
	.phase {
		text-align: center;
		font-size: 0.8rem;
		color: var(--muted);
		margin-top: 0.6rem;
	}
	.pens {
		display: flex;
		gap: 0.6rem;
		margin: 0.6rem 0;
	}
	.pen {
		flex: 1;
		padding: 0.7rem;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--text);
		font-weight: 600;
	}
	.pen.sel {
		background: var(--text);
		color: var(--bg);
		border-color: var(--text);
	}
	.adv {
		text-align: center;
		margin: 0.5rem 0;
	}
	.pill.done {
		gap: 0.35rem;
		color: var(--muted);
	}
	.pill.done .ptv {
		font-family: var(--font-mono);
		font-weight: 700;
		color: var(--muted);
	}
	.pill.done .ptv.ok {
		color: var(--accent);
	}
	.pill.done.perfect {
		background: color-mix(in srgb, var(--gold) 15%, transparent);
		border-color: color-mix(in srgb, var(--gold) 40%, transparent);
	}
	.pill.done.perfect .ptv.ok {
		color: var(--gold);
		text-shadow: 0 0 6px color-mix(in srgb, var(--gold) 40%, transparent);
	}
	.star {
		color: var(--gold);
		font-size: 0.85em;
		margin-right: 0.1rem;
	}
	.ypts.perfect {
		color: var(--gold);
	}
	.confetti {
		position: absolute;
		inset: 0;
		pointer-events: none;
		overflow: visible;
	}
	.cp {
		position: absolute;
		bottom: 0.6rem;
		right: 1.2rem;
		width: 5px;
		height: 5px;
		border-radius: 1px;
		background: hsl(var(--hue) 88% 58%);
		animation: cpburst 0.65s cubic-bezier(0.22, 0.61, 0.36, 1) var(--delay) both;
	}
	@keyframes cpburst {
		0%   { transform: translate(0, 0) rotate(0deg) scale(1); opacity: 1; }
		100% { transform: translate(var(--dx), var(--dy)) rotate(300deg) scale(0.2); opacity: 0; }
	}
	.pill.livep {
		color: var(--bg);
		background: var(--live);
		border-color: var(--live);
	}
	.pill.livep .dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--bg);
		animation: pulse 1.1s ease-in-out infinite;
	}
	@keyframes pulse {
		50% {
			opacity: 0.25;
		}
	}
	.score .pred {
		color: var(--muted);
		font-size: 0.95rem;
	}
	.resline {
		margin: 0.4rem 0 0.7rem;
		font-size: 0.9rem;
	}
	.yourtip {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.7rem 0.85rem;
		margin: 0.2rem 0 0.85rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-left: 3px solid var(--border-strong);
		border-radius: var(--radius-sm);
	}
	.yourtip.scored {
		border-left-color: var(--gold);
	}
	.ylabel {
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.yscore {
		font-size: 1.25rem;
		font-weight: 800;
	}
	.yadv {
		font-size: 0.85rem;
		color: var(--muted);
	}
	.ypts {
		font-family: var(--font-mono);
		font-weight: 700;
		font-size: 0.85rem;
		padding: 0.15rem 0.5rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		color: var(--muted);
	}
	.ypts.ok {
		color: var(--bg);
		background: var(--text);
		border-color: var(--text);
	}
	.friendsbtn.on {
		border-color: var(--border-strong);
		color: var(--text);
	}
	.friends {
		width: 100%;
		border-collapse: collapse;
		margin-top: 0.6rem;
		font-size: 0.88rem;
	}
	.friends th,
	.friends td {
		padding: 0.35rem 0.3rem;
		border-bottom: 1px solid var(--border);
		text-align: left;
	}
	.friends th {
		font-size: 0.72rem;
		font-weight: 600;
		color: var(--muted);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}
	.friends tr.fme td {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
		font-weight: 700;
	}
	.fname {
		width: 100%;
	}
	.ftip {
		white-space: nowrap;
		font-weight: 700;
	}
	.fscorer {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		font-weight: 400;
		font-size: 0.78rem;
	}
	.fscore {
		font-weight: 700;
		color: var(--muted);
		transition: color 0.1s;
	}
	.fscore-tendency {
		color: var(--success);
	}
	.fscore-exact {
		color: var(--gold);
	}
	.fcomps {
		display: flex;
		gap: 2px;
		margin-top: 2px;
	}
	.fcomp {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 14px;
		height: 14px;
		border-radius: 2px;
		font-size: 0.58rem;
		font-weight: 800;
		letter-spacing: 0;
		color: #fff;
	}
	.fc-tendency { background: var(--success); }
	.fc-exact    { background: var(--gold); }
	.fc-goals    { background: color-mix(in srgb, var(--accent) 80%, #2a8a3d); }
	.fc-diff     { background: color-mix(in srgb, var(--muted) 70%, #555); }
	.fc-et       { background: color-mix(in srgb, var(--gold) 70%, #b45a1e); font-size: 0.5rem; width: auto; padding: 0 3px; }
	.fts {
		font-weight: 700;
		color: var(--muted);
		font-size: 0.75rem;
		letter-spacing: 0.04em;
		transition: color 0.1s;
	}
	.fts-ok {
		color: var(--accent);
	}
	.fps {
		color: var(--muted);
		font-size: 0.75rem;
		transition: color 0.1s;
	}
	.fps-ok {
		color: var(--gold);
	}
	.fadv {
		display: block;
		font-size: 0.75rem;
		font-weight: 400;
		color: var(--muted);
	}
	.fpts {
		text-align: right;
		font-family: var(--font-mono);
		font-weight: 700;
		white-space: nowrap;
		padding-left: 0.6rem;
	}
	.fok { color: var(--accent); }
	.fperfect { color: var(--gold); }
	.fturbo { font-size: 0.75rem; opacity: 0.8; }
	.friends-actions {
		display: flex;
		justify-content: center;
		margin-top: 0.7rem;
	}
	.morefriends {
		min-width: 9.5rem;
	}
	.crowd {
		margin-top: 0.85rem;
		padding-top: 0.7rem;
		border-top: 1px dashed var(--border);
	}
	.crowd-head {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		margin-bottom: 0.4rem;
	}
	.crowd-title {
		font-size: 0.85rem;
		font-weight: 600;
	}
	.crowd-bar {
		display: flex;
		height: 1.3rem;
		width: 100%;
		border-radius: 0.5rem;
		overflow: hidden;
		background: var(--surface-2);
		border: 1px solid var(--border);
	}
	.crowd-bar .seg {
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.72rem;
		font-weight: 700;
		color: #fff;
		min-width: 0;
		overflow: hidden;
		white-space: nowrap;
	}
	.seg-home { background: color-mix(in srgb, var(--accent) 75%, #2a8a3d); }
	.seg-draw { background: color-mix(in srgb, var(--muted) 35%, #6b6f78); }
	.seg-away { background: color-mix(in srgb, var(--gold, #d49a18) 60%, #b45a1e); }
	.crowd-legend {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem 0.9rem;
		list-style: none;
		padding: 0;
		margin: 0.5rem 0 0;
		font-size: 0.75rem;
	}
	.crowd-legend li {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
	}
	.crowd-legend .dot {
		display: inline-block;
		width: 0.65rem;
		height: 0.65rem;
		border-radius: 50%;
	}
	.tc:not(.locked) .body {
		background: color-mix(in srgb, var(--surface-2) 62%, transparent);
	}
	:global(:root[data-theme='worldcup']) .tc {
		background:
			radial-gradient(circle at 12% 0%, rgba(143, 197, 143, 0.08), transparent 30%),
			linear-gradient(180deg, rgba(13, 34, 40, 0.96), rgba(7, 17, 25, 0.98)),
			var(--surface);
		border-color: color-mix(in srgb, var(--accent) 12%, var(--border));
		box-shadow: 0 16px 42px -34px rgba(0, 0, 0, 0.9), inset 0 1px 0 rgba(255, 255, 255, 0.035);
	}
	:global(:root[data-theme='worldcup']) .tc::before {
		display: none;
	}
	:global(:root[data-theme='worldcup']) .head {
		background: linear-gradient(180deg, rgba(255, 255, 255, 0.018), transparent);
	}
	:global(:root[data-theme='worldcup']) .body,
	:global(:root[data-theme='worldcup']) .tc:not(.locked) .body {
		background: color-mix(in srgb, var(--surface-2) 58%, transparent);
		border-top-color: color-mix(in srgb, var(--accent) 12%, var(--border));
	}
	:global(:root[data-theme='worldcup']) .pen,
	:global(:root[data-theme='worldcup']) .yourtip {
		background: color-mix(in srgb, var(--surface-2) 78%, transparent);
		border-color: color-mix(in srgb, var(--accent) 12%, var(--border));
	}
	:global(:root[data-theme='worldcup']) .pen.sel,
	:global(:root[data-theme='worldcup']) .ypts.ok {
		background: linear-gradient(180deg, color-mix(in srgb, var(--accent) 42%, var(--surface-2)), var(--surface-2));
		border-color: color-mix(in srgb, var(--accent) 36%, var(--border));
		color: var(--text);
	}
	.num {
		font-weight: 700;
	}
	.small {
		font-size: 0.85rem;
	}
	.first-scorer-section {
		margin-top: 0.6rem;
		padding-top: 0.5rem;
		border-top: 1px dashed var(--border);
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.3rem 0.8rem;
		align-items: start;
	}
	@media (max-width: 560px) {
		.first-scorer-section {
			grid-template-columns: 1fr;
			gap: 0.4rem 0;
		}
		.first-team-btns {
			flex-direction: row;
		}
	}
	.fs-col {
		min-width: 0;
	}
	.first-scorer-section .phase {
		text-align: left;
		margin-top: 0;
		margin-bottom: 0.25rem;
	}
	.first-team-btns {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}
	.first-team-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.45rem 0.55rem;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-size: 0.82rem;
		font-weight: 600;
		cursor: pointer;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
		width: 100%;
	}
	.first-team-btn.sel {
		background: var(--text);
		color: var(--bg);
		border-color: var(--text);
	}
	.player-picker {
		position: relative;
	}
	.player-input {
		width: 100%;
		padding: 0.45rem 1.8rem 0.45rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-size: 0.84rem;
		box-sizing: border-box;
	}
	.player-input:focus {
		outline: 2px solid color-mix(in srgb, var(--accent) 50%, transparent);
		outline-offset: -1px;
	}
	.player-clear {
		position: absolute;
		right: 0.4rem;
		top: 50%;
		transform: translateY(-50%);
		background: none;
		border: none;
		color: var(--muted);
		font-size: 1.1rem;
		cursor: pointer;
		padding: 0 0.15rem;
		line-height: 1;
	}
	.player-dropdown {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		box-shadow: var(--shadow-pop);
		list-style: none;
		margin: 2px 0 0;
		padding: 0.15rem 0;
		max-height: 320px;
		overflow-y: auto;
	}
	.player-dropdown li button {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.4rem;
		width: 100%;
		padding: 0.32rem 0.6rem;
		background: none;
		border: none;
		color: var(--text);
		font: inherit;
		font-size: 0.83rem;
		cursor: pointer;
		text-align: left;
	}
	.player-dropdown li button:hover,
	.player-dropdown li button.selected {
		background: color-mix(in srgb, var(--accent) 10%, transparent);
	}
	.p-pos {
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		flex-shrink: 0;
	}
	.first-scorer-locked {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.35rem 0.6rem;
		padding: 0.5rem 0.7rem;
		margin: 0.25rem 0 0.7rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		font-size: 0.82rem;
	}
	.fs-label {
		font-weight: 700;
		color: var(--muted);
		text-transform: uppercase;
		font-size: 0.7rem;
		letter-spacing: 0.07em;
	}
	.fs-val {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		font-weight: 600;
	}
	.turbo-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.6rem;
		height: 1.6rem;
		border-radius: 50%;
		border: 1px solid var(--border);
		background: var(--surface-2);
		color: var(--muted);
		font-size: 0.85rem;
		cursor: pointer;
		padding: 0;
		line-height: 1;
		transition: background 0.15s, border-color 0.15s, color 0.15s, box-shadow 0.15s;
		flex-shrink: 0;
	}
	.turbo-btn:hover {
		border-color: var(--accent);
		color: var(--accent);
	}
	.turbo-btn.on {
		background: color-mix(in srgb, var(--accent) 18%, var(--surface-2));
		border-color: var(--accent);
		color: var(--accent);
		box-shadow: 0 0 6px color-mix(in srgb, var(--accent) 35%, transparent);
	}
	.turbo-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		padding: 0.15rem 0.45rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface-2);
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.04em;
		color: var(--muted);
		flex-shrink: 0;
	}
	.turbo-pill.on {
		border-color: color-mix(in srgb, var(--accent) 50%, var(--border));
		background: color-mix(in srgb, var(--accent) 12%, var(--surface-2));
		color: var(--accent);
		box-shadow: 0 0 6px color-mix(in srgb, var(--accent) 25%, transparent);
	}
	.turbo-pill.auto {
		border-color: color-mix(in srgb, var(--gold) 50%, var(--border));
		background: color-mix(in srgb, var(--gold) 10%, var(--surface-2));
		color: var(--gold);
	}
</style>
