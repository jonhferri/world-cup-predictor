<script lang="ts">
	import { pb } from '$lib/pb';
	import { serverClock } from '$lib/serverclock.svelte';
	import { api, type LeagueSummary } from '$lib/api';
	import { language } from '$lib/language.svelte';

	let when = $state('');
	let busy = $state(false);
	let msg = $state('');
	let msgTone = $state<'ok' | 'error'>('error');

	let botCount = $state(3);
	let botLeague = $state('');
	let chatCount = $state(6);
	let chatLeague = $state('');
	let topscorers = $state<{ id: string; name: string; goals: number }[]>([]);
	let leagues = $state<LeagueSummary[]>([]);
	$effect(() => {
		if (serverClock.dev) {
			api
				.myLeagues()
				.then(
					(r) =>
						(leagues = r.leagues.filter((league) => league.inviteCode !== 'GLOBAL'))
				)
				.catch(() => {});
			api
				.devTopscorers()
				.then((r) => (topscorers = r.players))
				.catch(() => {});
		}
	});

	async function genBots() {
		busy = true;
		msg = '';
		try {
			await pb.send('/api/dev/bots', {
				method: 'POST',
				body: { count: botCount, leagueId: botLeague }
			});
			location.reload();
		} catch (e: unknown) {
			msgTone = 'error';
			msg = (e as { message?: string })?.message ?? language.text('Feilet', 'Feila', 'Failed');
			busy = false;
		}
	}

	async function sendBotChat() {
		busy = true;
		msg = '';
		try {
			const result = await pb.send<{ sent: number }>('/api/dev/bot-chat', {
				method: 'POST',
				body: { count: chatCount, leagueId: chatLeague }
			});
			msgTone = 'ok';
			msg = language.text(
				`Sendte ${result.sent} botmelding${result.sent === 1 ? '' : 'er'}.`,
				`Sendte ${result.sent} botmelding${result.sent === 1 ? '' : 'ar'}.`,
				`Sent ${result.sent} bot message${result.sent === 1 ? '' : 's'}.`
			);
		} catch (e: unknown) {
			msgTone = 'error';
			msg = (e as { message?: string })?.message ?? language.text('Feilet', 'Feila', 'Failed');
		} finally {
			busy = false;
		}
	}

	async function saveTopscorers() {
		busy = true;
		msg = '';
		try {
			const map: Record<string, number> = {};
			for (const ts of topscorers) map[ts.id] = ts.goals;
			await api.devSetTopscorers(map);
			msgTone = 'ok';
			msg = language.text('Toppscorere lagret.', 'Toppscorarar lagra.', 'Top scorers saved.');
		} catch (e: unknown) {
			msgTone = 'error';
			msg = (e as { message?: string })?.message ?? language.text('Feilet', 'Feila', 'Failed');
		} finally {
			busy = false;
		}
	}

	$effect(() => {
		serverClock.refresh();
	});

	// Seed the input from the current sim time (or now).
	$effect(() => {
		if (!when) {
			const base = serverClock.simTime
				? new Date(serverClock.simTime)
				: new Date(serverClock.now());
			when = base.toISOString().slice(0, 16);
		}
	});

	const presets: { label: string; ts: string }[] = [
		{ label: 'opening', ts: '2026-06-11T20:00' },
		{ label: 'group-md2-live', ts: '2026-06-15T21:30' },
		{ label: 'after-groups', ts: '2026-06-25T06:00' },
		{ label: 'after-r32', ts: '2026-07-04T06:00' },
		{ label: 'after-qf', ts: '2026-07-12T06:00' },
		{ label: 'after-final', ts: '2026-07-20T00:00' }
	];

	function presetLabel(label: string) {
		const labels: Record<string, [string, string, string]> = {
			opening: ['Åpningskamp', 'Opningskamp', 'Opening match'],
			'group-md2-live': ['Gruppe MD2 live', 'Gruppe MD2 live', 'Group MD2 live'],
			'after-groups': ['Etter gruppene', 'Etter gruppene', 'After groups'],
			'after-r32': ['Etter 32-delsfinaler', 'Etter 32-delsfinalar', 'After R32'],
			'after-qf': ['Etter kvartfinaler', 'Etter kvartfinalar', 'After QF'],
			'after-final': ['Etter finalen', 'Etter finalen', 'After final']
		};
		const [nb, nn, en] = labels[label] ?? [label, label, label];
		return language.text(nb, nn, en);
	}

	async function advance(ts: string) {
		busy = true;
		msg = '';
		try {
			await pb.send('/api/dev/advance', {
				method: 'POST',
				body: { timestamp: ts }
			});
			location.reload(); // re-pull all stores against the new clock
		} catch (e: unknown) {
			msgTone = 'error';
			msg = (e as { message?: string })?.message ?? language.text('Feilet', 'Feila', 'Failed');
			busy = false;
		}
	}

	async function reset() {
		busy = true;
		msg = '';
		try {
			await pb.send('/api/dev/reset', { method: 'POST', body: {} });
			location.reload();
		} catch (e: unknown) {
			msgTone = 'error';
			msg = (e as { message?: string })?.message ?? language.text('Feilet', 'Feila', 'Failed');
			busy = false;
		}
	}
</script>

<p class="kicker">{language.text('Testverktøy', 'Testverktøy', 'Test harness')}</p>
<h1>{language.text('Utviklerverktøy', 'Utviklarverktøy', 'Dev tools')}</h1>

{#if !serverClock.loaded}
	<p class="muted">…</p>
{:else if !serverClock.dev}
	<section class="card">
		<p class="muted">
			{language.text('Avslått. Start serveren med', 'Avslått. Start serveren med', 'Disabled. Start the server with')} <code>WMP_DEV=1</code>
			{language.text('for å simulere turneringen.', 'for å simulere turneringa.', 'to simulate the tournament.')}
		</p>
	</section>
{:else}
	<section class="card">
		<div class="state">
			<span class="kicker">{language.text('Simulert klokke', 'Simulert klokke', 'Simulated clock')}</span>
			<b class="digits"
				>{serverClock.simulated
					? new Date(serverClock.now()).toLocaleString()
					: language.text('live (sanntid)', 'live (sanntid)', 'live (real time)')}</b
			>
		</div>
	</section>

	<section class="card">
			<h3>{language.text('Hopp til', 'Hopp til', 'Jump to')}</h3>
		<p class="muted small">
				{language.text(
					'Kamper før dette tidspunktet blir simulert (ferdige, eller live hvis de er midt i kampen); senere kamper blir nullstilt. Låsing, venners kamptips og VM-tipsfristen følger denne klokken.',
					'Kampar før dette tidspunktet blir simulerte (ferdige, eller live viss dei er midt i kampen); seinare kampar blir nullstilte. Låsing, venetips og VM-tipsfristen følgjer denne klokka.',
					'Matches before this time are simulated (finished, or live if in the middle of the match); later matches are reset. Locks, friends\' match tips, and the World Cup tip deadline follow this clock.'
				)}
		</p>
		<div class="field">
			<input class="input" type="datetime-local" bind:value={when} />
		</div>
		<button
			class="btn"
			disabled={busy || !when}
			onclick={() => advance(when)}>{language.text('Kjør fram', 'Køyr fram', 'Advance')}</button
		>

		<div class="presets">
			{#each presets as p (p.ts)}
				<button
					class="chip"
					disabled={busy}
					onclick={() => advance(p.ts)}>{presetLabel(p.label)}</button
				>
			{/each}
		</div>
	</section>

	<section class="card">
		<h3>{language.text('Lag bot-spillere', 'Lag bot-spelarar', 'Generate bot players')}</h3>
		<p class="muted small">
			{language.text(
				'Hver bot får et helt tilfeldig VM-tips og kamptips for hver kamp, og blir med i valgt liga (eller alle private ligaer) - et live tabelløp.',
				'Kvar bot får eit heilt tilfeldig VM-tips og kamptips for kvar kamp, og blir med i vald liga (eller alle private ligaene dine) - eit live tabelløp.',
				'Each bot gets a fully random World Cup tip and a match tip for every match, and joins the selected league (or all your private leagues) - a live leaderboard race.'
			)}
		</p>
		<div class="field">
			<label for="bc">{language.text('Hvor mange', 'Kor mange', 'How many')}</label>
			<input
				id="bc"
				class="input"
				type="number"
				min="1"
				max="20"
				bind:value={botCount}
			/>
		</div>
		<div class="field">
			<label for="bl">{language.text('Liga', 'Liga', 'League')}</label>
			<select id="bl" class="input" bind:value={botLeague}>
				<option value="">{language.text('Alle private ligaer', 'Alle dei private ligaene mine', 'All my private leagues')}</option>
				{#each leagues as l (l.id)}
					<option value={l.id}>{l.name}</option>
				{/each}
			</select>
		</div>
		<button class="btn" disabled={busy} onclick={genBots}>
			{language.text(`Lag ${botCount} bot${botCount === 1 ? '' : 'er'}`, `Lag ${botCount} bot${botCount === 1 ? '' : 'ar'}`, `Generate ${botCount} bot${botCount === 1 ? '' : 's'}`)}
		</button>
	</section>

	<section class="card">
		<h3>{language.text('Send bot-chat', 'Send bot-chat', 'Send bot chat')}</h3>
		<p class="muted small">
			{language.text(
				'Bruk eksisterende testboter til å poste live meldinger i liga-chatten. Lag boter først hvis ligaen ikke har noen. Hvis du ikke velger liga, sendes meldinger i hver private liga som allerede har boter.',
				'Bruk eksisterande testbotar til å poste live meldingar i liga-chatten. Lag botar først viss ligaen ikkje har nokon. Viss du ikkje vel liga, blir meldingar sende i kvar av dei private ligaene dine som allereie har botar.',
				'Use existing test bots to post live messages into league chat. Generate bots first if the league has none. If no league is chosen, messages are sent in each of your private leagues that already has bots.'
			)}
		</p>
		<div class="field">
			<label for="cc">{language.text('Hvor mange meldinger', 'Kor mange meldingar', 'How many messages')}</label>
			<input
				id="cc"
				class="input"
				type="number"
				min="1"
				max="50"
				bind:value={chatCount}
			/>
		</div>
		<div class="field">
			<label for="cl">{language.text('Liga', 'Liga', 'League')}</label>
			<select id="cl" class="input" bind:value={chatLeague}>
				<option value="">{language.text('Alle private ligaer med boter', 'Alle private ligaene mine med botar', 'All my private leagues with bots')}</option>
				{#each leagues as l (l.id)}
					<option value={l.id}>{l.name}</option>
				{/each}
			</select>
		</div>
		<button class="btn" disabled={busy} onclick={sendBotChat}>
			{language.text(`Send ${chatCount} botmelding${chatCount === 1 ? '' : 'er'}`, `Send ${chatCount} botmelding${chatCount === 1 ? '' : 'ar'}`, `Send ${chatCount} bot message${chatCount === 1 ? '' : 's'}`)}
		</button>
	</section>

	<section class="card">
		<h3>{language.text('Toppscorere', 'Toppscorarar', 'Top scorers')}</h3>
		<p class="muted small">
			{language.text('Sett mål for de forhåndsvalgte kandidatene for å se hvordan det påvirker tabellen.', 'Sett mål for dei forhandsvalde kandidatane for å sjå korleis det påverkar tabellen.', 'Set goals for the seeded candidates to see how it affects the leaderboard.')}
		</p>
		<div class="ts-grid">
			{#each topscorers as ts (ts.id)}
				<div class="field horiz">
					<label for="ts-{ts.id}">{ts.name}</label>
					<input id="ts-{ts.id}" class="input narrow digits" type="number" min="0" max="25" bind:value={ts.goals} />
				</div>
			{/each}
		</div>
		{#if topscorers.length > 0}
			<button class="btn" disabled={busy} onclick={saveTopscorers}>
				{language.text('Lagre mål', 'Lagre mål', 'Save goals')}
			</button>
		{:else}
			<p class="muted small">{language.text('Ingen forhåndsvalgte spillere funnet. Sørg for at databasen er fylt.', 'Ingen forhandsvalde spelarar funne. Sørg for at databasen er fylt.', 'No seeded players found. Ensure database is populated.')}</p>
		{/if}
	</section>

	<section class="card">
		<h3>{language.text('Nullstill', 'Nullstill', 'Reset')}</h3>
		<p class="muted small">
			{language.text('Tøm alle resultater og den simulerte klokken (tilbake til sanntid).', 'Tøm alle resultat og den simulerte klokka (tilbake til sanntid).', 'Clear all results and the simulated clock (back to real time).')}
		</p>
		<button class="btn secondary" disabled={busy} onclick={reset}
			>{language.text('Nullstill alt', 'Nullstill alt', 'Reset all')}</button
		>
	</section>

	{#if msg}<p class:error={msgTone === 'error'} class:notice={msgTone === 'ok'}>{msg}</p>{/if}
{/if}

<style>
	h1 {
		margin: 0.1rem 0 1rem;
	}
	.small {
		font-size: 0.85rem;
	}
	.state {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}
	.state b {
		font-size: 1.2rem;
	}
	.presets {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-top: 0.9rem;
	}
	.chip {
		padding: 0.5rem 0.8rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		color: var(--text);
		font:
			700 0.78rem var(--font);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		cursor: pointer;
	}
	.chip:hover {
		border-color: var(--accent);
	}
	code {
		font-family: var(--font-mono);
		color: var(--accent);
	}
	.notice {
		color: var(--accent);
		font-size: 0.9rem;
	}
	.horiz {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.5rem;
	}
	.ts-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 0 1.5rem;
		margin-bottom: 1rem;
	}
	.narrow {
		width: 70px;
	}
</style>
