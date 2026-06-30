<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { language } from '$lib/language.svelte';
	import {
		ArrowLeft,
		CheckCircle2,
		Clock,
		Info,
		ListChecks,
		Medal,
		Network,
		Telescope,
		Trophy,
		Users,
		Volleyball,
		X
	} from '@lucide/svelte';

	let flow = $derived.by(() => [
		{
			icon: Telescope,
			title: language.text('Velg toppscorer før avspark', 'Vel toppscorar før avspark', 'Pick your Golden Boot winner before kickoff'),
			text: language.text(
				'Velg din toppscorervinner og søk opp en outsider om du tror på det. Grupper og sluttspilltre utledes automatisk fra kamptipsene dine.',
				'Vel toppscorarvinnar og søk opp ein outsider om du trur på det. Grupper og sluttspeltre vert utleidde automatisk frå kamptipsa dine.',
				'Pick your Golden Boot winner and search for an outsider if you back one. Group standings and the knockout bracket are auto-derived from your match tips.'
			)
		},
		{
			icon: Volleyball,
			title: language.text('Kamptips før hver kamp', 'Kamptips før kvar kamp', 'Match tips before every game'),
			text: language.text(
				'Tipp resultatet for hver kamp. Du kan endre helt fram til avspark.',
				'Tipp resultatet for kvar kamp. Du kan endre heilt fram til avspark.',
				'Pick the score for every match. You can change it right up until kickoff.'
			)
		},
		{
			icon: Clock,
			title: language.text('Tipset låses', 'Tipset låser seg', 'The tip locks'),
			text: language.text(
				'Når kampen starter, låses tipset, og tipsene til venner blir synlige i ligaene.',
				'Når kampen startar, blir tipset låst, og tipsa til vener blir synlege i ligaene.',
				'When the game starts, your tip locks and friends’ tips become visible in leagues.'
			)
		},
		{
			icon: Trophy,
			title: language.text('Poeng underveis', 'Poeng undervegs', 'Points along the way'),
			text: language.text(
				'Resultater, tabeller og poeng oppdateres gjennom gruppespill og sluttspill.',
				'Resultat, tabellar og poeng blir oppdaterte gjennom gruppespel og sluttspel.',
				'Results, tables, and points update continuously through the group stage and knockout rounds.'
			)
		}
	]);

	let matchPoints = $derived.by(() => [
		{ label: language.text('Rett utfall (gruppespill)', 'Rett utfall (gruppespel)', 'Correct outcome (group stage)'), value: '5', detail: language.text('1/X/2 basert på resultatet etter 90 min', '1/X/2 basert på resultatet etter 90 min', '1/X/2 based on the 90-minute result') },
		{ label: language.text('Eksakt hjemmemål (gruppespill)', 'Eksakt heimemål (gruppespel)', 'Exact home goals (group stage)'), value: '+5', detail: language.text('du tippa nøyaktig antall hjemmemål', 'du tippa nøyaktig tal heimemål', 'you guessed the home team\'s exact goal count') },
		{ label: language.text('Eksakt bortemål (gruppespill)', 'Eksakt bortemål (gruppespel)', 'Exact away goals (group stage)'), value: '+5', detail: language.text('du tippa nøyaktig antall bortemål', 'du tippa nøyaktig tal bortemål', 'you guessed the away team\'s exact goal count') },
		{ label: language.text('Eksakt resultat (gruppespill)', 'Eksakt resultat (gruppespel)', 'Exact score (group stage)'), value: '+10', detail: language.text('begge lag nøyaktig riktig etter 90 min', 'begge lag nøyaktig riktig etter 90 min', 'both teams\' goals exactly right after 90 min') },
		{ label: language.text('Rett målforskjell (gruppespill)', 'Rett målforskjell (gruppespel)', 'Correct goal difference (group stage)'), value: '+5', detail: language.text('for eksempel ettmålsseier eller uavgjort', 'til dømes eittmålsiger eller uavgjort', 'for example a one-goal win or a draw') },
		{ label: language.text('Sluttspill: rett FT-målforskjell + mål', 'Sluttspel: rett FT-målforskjell + mål', 'Knockout: correct FT goal diff + goals'), value: 'maks 20', detail: language.text('målforskjell (5) + eksakt hjemmemål (5) + eksakt bortemål (5) + eksakt bonus (5) etter 90 min', 'målforskjell (5) + eksakt heimemål (5) + eksakt bortemål (5) + eksakt bonus (5) etter 90 min', 'goal diff (5) + exact home (5) + exact away (5) + exact bonus (5) after 90 min') },
		{ label: language.text('Sluttspill: rett ET-målforskjell + mål', 'Sluttspel: rett ET-målforskjell + mål', 'Knockout: correct ET goal diff + goals'), value: '+maks 20', detail: language.text('kun hvis kamp gikk til ekstraomganger og du tippa uavgjort etter 90 min', 'berre viss kampen gjekk til ekstraomgangar og du tippa uavgjort etter 90 min', 'only if match went to extra time and you predicted a draw at FT') },
		{ label: language.text('Sluttspill: rett lag videre', 'Sluttspel: rett lag vidare', 'Knockout: correct team to advance'), value: '+5', detail: language.text('laget som faktisk gikk videre (inkl. etter straffespark)', 'laget som faktisk gjekk vidare (inkl. etter straffespark)', 'the team that actually advanced (including after penalties)') },
		{ label: language.text('Første lag til å score', 'Første lag til å score', 'First team to score'), value: '+5', detail: language.text('laget som scorer kampens første mål', 'laget som scorar kampens første mål', 'the team that scores the first goal of the match') },
		{ label: language.text('Første spiller til å score', 'Første spelar til å score', 'First player to score'), value: '+10', detail: language.text('spilleren som scorer kampens første mål', 'spelaren som scorar kampens første mål', 'the player who scores the first goal of the match') }
	]);

	let forecastPoints = $derived.by(() => [
		{ label: language.text('Rett gruppeplassering', 'Rett gruppeplassering', 'Correct group placement'), value: '1' },
		{ label: language.text('Perfekt gruppe', 'Perfekt gruppe', 'Perfect group'), value: '+2' },
		{ label: language.text('Rett lag videre', 'Rett lag vidare', 'Correct team through'), value: '+1' },
		{ label: language.text('R32 / R16 / kvart', '32-del / 16-del / kvart', 'R32 / R16 / QF'), value: '1 / 2 / 3' },
		{ label: language.text('Semi / finale / vinner', 'Semi / finale / vinnar', 'SF / Final / Winner'), value: '5 / 8 / 13' },
		{ label: language.text('Rett toppscorer', 'Rett toppscorar', 'Correct Golden Boot winner'), value: '15' }
	]);

	let appFacts = $derived.by(() => [
		{ icon: Users, title: language.text('Ligaer', 'Ligaer', 'Leagues'), text: language.text('Opprett private ligaer, del invitasjon og følg tabellen sammen.', 'Opprett private ligaer, del invitasjon og følg tabellen saman.', 'Create private leagues, share an invite, and follow the table together.') },
		{ icon: Network, title: language.text('Turnering', 'Turnering', 'Tournament'), text: language.text('Se grupper, kamper og sluttspilltreet mens VM går.', 'Sjå grupper, kampar og sluttspelstreet medan VM går føre seg.', 'See groups, fixtures, and the knockout tree as the World Cup unfolds.') },
		{ icon: ListChecks, title: language.text('Oversikt', 'Oversikt', 'Overview'), text: language.text('Forsiden viser hva som mangler, neste frist og plasseringen din.', 'Framsida viser kva som manglar, neste frist og plasseringa di.', 'The home page shows what is missing, the next deadline, and your standing.') }
	]);

	function closeInfo() {
		if (browser && history.length > 1) {
			history.back();
			return;
		}
		void goto('/');
	}
</script>

<svelte:head>
	<title>{language.text('Info om spillet', 'Info om spelet', 'About the game')} · Cozinhámos Predictions</title>
</svelte:head>

<div class="info-page">
	<button class="close" type="button" aria-label={language.text('Lukk og gå tilbake', 'Lukk og gå tilbake', 'Close and go back')} onclick={closeInfo}>
		<X size={18} />
		<span>{language.text('Lukk', 'Lukk', 'Close')}</span>
	</button>

	<section class="hero" aria-labelledby="info-title">
		<div class="hero-copy">
			<p class="kicker">Info</p>
			<h1 id="info-title">{language.text('Slik fungerer Cozinhámos Predictions', 'Slik fungerer Cozinhámos Predictions', 'How Cozinhámos Predictions works')}</h1>
			<p class="lead">
				{language.text(
					'Velg toppscorer og første lag og spiller til å score i hver kamp. Grupper og sluttspilltre utledes automatisk fra kamptipsene dine. Konkurrer med venner i ligaer gjennom hele turneringen.',
					'Vel toppscorar og første lag og spelar til å score i kvar kamp. Grupper og sluttspeltre vert utleidde automatisk frå kamptipsa dine. Konkurrer med vener i ligaer gjennom heile turneringa.',
					'Pick a Golden Boot winner, and a first team and player to score in every match. Group standings and the bracket are auto-derived from your match tips. Compete with friends in leagues throughout the tournament.'
				)}
			</p>
		</div>
		<div class="scoreboard" aria-label={language.text('Kort oversikt', 'Kort oversikt', 'Quick overview')}>
			<div><strong>104</strong><span>{language.text('kamper', 'kampar', 'matches')}</span></div>
			<div><strong>1</strong><span>{language.text('toppscorervalg', 'toppscorarval', 'Golden Boot pick')}</span></div>
			<div><strong>45</strong><span>{language.text('maks per kamp', 'maks per kamp', 'max per game')}</span></div>
			<div><strong>15</strong><span>{language.text('toppscorer-poeng', 'toppscorar-poeng', 'Golden Boot pts')}</span></div>
		</div>
	</section>

	<section class="section-block" aria-labelledby="journey-title">
		<div class="section-head">
			<Info size={18} />
			<h2 id="journey-title">{language.text('Slik fungerer det', 'Slik går det føre seg', 'How it flows')}</h2>
		</div>
		<div class="flow-grid">
			{#each flow as step, index}
				{@const Icon = step.icon}
				<article class="card flow-card">
					<div class="step-mark"><span>{index + 1}</span><Icon size={22} /></div>
					<h3>{step.title}</h3>
					<p>{step.text}</p>
				</article>
			{/each}
		</div>
	</section>

	<section class="section-block" aria-labelledby="app-title">
		<div class="section-head">
			<CheckCircle2 size={18} />
			<h2 id="app-title">{language.text('Appen og spillet', 'Appen og spelet', 'The app and the game')}</h2>
		</div>
		<div class="facts-grid">
			{#each appFacts as fact}
				{@const Icon = fact.icon}
				<article class="card fact-card">
					<Icon size={22} />
					<div>
						<h3>{fact.title}</h3>
						<p>{fact.text}</p>
					</div>
				</article>
			{/each}
		</div>
	</section>

	<section class="section-block scoring" aria-labelledby="score-title">
		<div class="section-head">
			<Medal size={18} />
			<h2 id="score-title">{language.text('Poengsystem', 'Poengsystem', 'Scoring system')}</h2>
		</div>

		<div class="score-layout">
			<article class="card score-panel match-panel">
				<div class="panel-title">
					<Volleyball size={20} />
					<h3>{language.text('Kamptips', 'Kamptips', 'Match tips')}</h3>
				</div>
				<p>{language.text('Gruppespill: maks 30 poeng + førstemålscorer (15). Sluttspill: maks 45 poeng (FT 20 + ET 20 + videre 5) + førstemålscorer (15).', 'Gruppespel: maks 30 poeng + førstemålscorar (15). Sluttspel: maks 45 poeng (FT 20 + ET 20 + vidare 5) + førstemålscorar (15).', 'Group: max 30 pts + first scorer (15). Knockout: max 45 pts (FT 20 + ET 20 + advancer 5) + first scorer (15).')}</p>
				<div class="point-list">
					{#each matchPoints as point}
						<div class="point-row">
							<strong>{point.value}</strong>
							<div>
								<span>{point.label}</span>
								<small>{point.detail}</small>
							</div>
						</div>
					{/each}
				</div>
			</article>

			<article class="card score-panel forecast-panel">
				<div class="panel-title">
					<Telescope size={20} />
					<h3>{language.text('Turneringstips', 'Turneringstips', 'Tournament forecast')}</h3>
				</div>
				<p>{language.text('Grupper og sluttspilltre utledes automatisk fra kamptipsene dine og gir poeng etter hvert som resultatene faller. Toppscorer-valget låses ved første kamp.', 'Grupper og sluttspeltre vert utleidde automatisk frå kamptipsa dine og gir poeng etter kvart som resultata fell. Toppscorarval låser seg ved første kamp.', 'Group standings and the bracket are auto-derived from your match tips and score as results come in. The Golden Boot pick locks at the first match.')}</p>
				<div class="forecast-grid">
					{#each forecastPoints as point}
						<div>
							<span>{point.label}</span>
							<strong>{point.value}</strong>
						</div>
					{/each}
				</div>
			</article>
		</div>

		<div class="card tie-break">
			<Medal size={18} />
			<p>
				{language.text(
					'Ved poenglikhet sorteres tabellen etter flest eksakte resultater, flest rette vinnere, lavest målforskjell-feil, færrest leverte tips og tidligste levering.',
					'Ved poenglikskap blir tabellen sortert etter flest eksakte resultat, flest rette vinnarar, lågaste målforskjell-feil, færrast leverte tips og tidlegaste levering.',
					'If points are tied, the table sorts by most exact scores, most correct winners, lowest goal-difference error, fewest submitted tips, and earliest submission.'
				)}
			</p>
		</div>
	</section>

	<button class="back-bottom" type="button" onclick={closeInfo}>
		<ArrowLeft size={18} />
		{language.text('Tilbake', 'Tilbake', 'Back')}
	</button>

	<footer class="copyright">
		<p>© 2026 Øyvind Hovden · <a href="mailto:oyvhov@gmail.com">oyvhov@gmail.com</a></p>
	</footer>
</div>

<style>
	.info-page {
		max-width: 1080px;
		margin: 0 auto;
		padding: 0 0 2rem;
	}
	:global(.info-page .card) {
		border-color: color-mix(in srgb, var(--border) 55%, transparent);
	}
	.info-page h1,
	.info-page h2,
	.info-page h3 {
		letter-spacing: 0;
	}
	.close {
		position: sticky;
		top: calc(var(--topbar-h) + 0.75rem);
		z-index: 8;
		margin-left: auto;
		display: flex;
		align-items: center;
		gap: 0.4rem;
		width: fit-content;
		padding: 0.55rem 0.8rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--surface) 86%, transparent);
		color: var(--text);
		font: inherit;
		font-weight: 800;
		box-shadow: var(--shadow-pop);
		backdrop-filter: blur(14px);
		cursor: pointer;
	}
	.hero {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		gap: 1rem;
		padding: 1.2rem 0 0.9rem;
	}
	.hero-copy {
		padding: 1rem 0 0;
	}
	.kicker {
		margin: 0 0 0.55rem;
	}
	h1 {
		font-size: 2rem;
		line-height: 1.05;
	}
	.lead {
		max-width: 680px;
		margin: 0.8rem 0 0;
		font-size: 1.02rem;
		line-height: 1.55;
		color: var(--muted);
	}
	.scoreboard {
		display: flex;
		flex-wrap: wrap;
		gap: 0.65rem;
		margin-top: 0.6rem;
	}
	.scoreboard div {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.45rem 1rem;
		background: var(--surface-2);
		border-radius: var(--radius-pill);
	}
	.scoreboard strong {
		font-size: 1.15rem;
		line-height: 1;
		color: var(--text);
	}
	.scoreboard span {
		font-size: 0.85rem;
		font-weight: 700;
		color: var(--muted);
	}
	.section-block {
		margin-top: 1.35rem;
	}
	.section-head {
		display: flex;
		align-items: center;
		gap: 0.55rem;
		margin-bottom: 0.75rem;
		color: var(--text);
	}
	:global(.section-head svg) {
		color: var(--accent-2);
	}
	.section-head h2 {
		font-size: 1.35rem;
	}
	.flow-grid,
	.facts-grid,
	.score-layout {
		display: grid;
		gap: 0.75rem;
	}
	.step-mark {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
		color: var(--accent);
	}
	.step-mark span {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 50%;
		background: color-mix(in srgb, var(--accent) 14%, transparent);
		font-weight: 900;
		color: var(--text);
	}
	.flow-card h3,
	.fact-card h3,
	.score-panel h3 {
		font-size: 1rem;
	}
	.flow-card p,
	.fact-card p,
	.score-panel p,
	.tie-break p {
		margin: 0.45rem 0 0;
		line-height: 1.48;
		color: var(--muted);
	}
	.fact-card {
		display: flex;
		gap: 0.75rem;
	}
	:global(.fact-card svg) {
		flex: none;
		color: var(--accent);
	}
	.panel-title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	:global(.panel-title svg) {
		color: var(--accent-2);
	}
	.point-list {
		display: grid;
		gap: 0.6rem;
		margin-top: 0.9rem;
	}
	.point-row {
		display: grid;
		grid-template-columns: 3.25rem minmax(0, 1fr);
		gap: 0.75rem;
		align-items: center;
		padding: 0.72rem;
		border-radius: var(--radius-sm);
		background: var(--surface-2);
	}
	.point-row strong {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 3.25rem;
		height: 3.25rem;
		border-radius: 50%;
		background: var(--text);
		color: var(--bg);
		font-size: 1.05rem;
	}
	.point-row span,
	.forecast-grid span {
		display: block;
		font-weight: 900;
	}
	.point-row small {
		display: block;
		margin-top: 0.18rem;
		line-height: 1.35;
		color: var(--muted);
	}
	.forecast-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.55rem;
		margin-top: 0.9rem;
	}
	.forecast-grid div {
		min-height: 88px;
		padding: 0.75rem;
		border-radius: var(--radius-sm);
		background: var(--surface-2);
	}
	.forecast-grid strong {
		display: block;
		margin-top: 0.5rem;
		font-size: 1.35rem;
		color: var(--accent-2);
	}
	.tie-break {
		display: flex;
		gap: 0.7rem;
		align-items: flex-start;
		margin-top: 0.75rem;
		padding: 0.9rem 1rem;
	}
	:global(.tie-break svg) {
		flex: none;
		margin-top: 0.15rem;
		color: var(--gold);
	}
	.tie-break p {
		margin: 0;
	}
	.back-bottom {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		width: 100%;
		margin-top: 1.35rem;
		padding: 0.85rem 1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-weight: 900;
		cursor: pointer;
	}
	@media (min-width: 560px) {
		.flow-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (min-width: 760px) {
		.info-page {
			padding-bottom: 3rem;
		}
		h1 {
			font-size: 2.75rem;
		}
		.hero {
			grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.8fr);
			align-items: end;
			gap: 1.5rem;
			padding-top: 2rem;
		}
		.facts-grid {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
		.score-layout {
			grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		}
		.back-bottom {
			width: fit-content;
			padding-inline: 1.2rem;
		}
	}
	@media (min-width: 1020px) {
		.flow-grid {
			grid-template-columns: repeat(4, minmax(0, 1fr));
		}
	}
	@media (max-width: 759px) {
		.forecast-grid {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	.copyright {
		margin-top: 2rem;
		text-align: center;
		font-size: 0.82rem;
		color: var(--muted);
	}
	.copyright a {
		color: var(--muted);
		text-decoration: underline;
		text-underline-offset: 3px;
	}
</style>
