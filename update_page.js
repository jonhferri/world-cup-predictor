const fs = require('fs');
let code = fs.readFileSync('frontend/src/routes/+page.svelte', 'utf8');

// 1. Add Topscorer script states
const scriptAddIndex = code.indexOf('let miniLeaderboard');
if (scriptAddIndex !== -1 && code.indexOf('myGoldenBootPlayerId') === -1) {
  const injection = `
	let myGoldenBootPlayerId = $derived(fs.loaded ? fs.goldenBootPlayer : null);
	let myGoldenBootPickName = $derived.by(() => {
		if (!myGoldenBootPlayerId || !activeLeague) return null;
		const gb = goldenBoots[activeLeague.id];
		if (!gb) return null;
		return gb.players.find(p => p.id === myGoldenBootPlayerId)?.name;
	});
	let myGoldenBootPickIsWinner = $derived.by(() => {
		if (!myGoldenBootPlayerId || !activeLeague) return false;
		const gb = goldenBoots[activeLeague.id];
		if (!gb) return false;
		const me = gb.players.find(p => p.id === myGoldenBootPlayerId);
		return me ? me.rank === 1 && me.goals > 0 : false;
	});

`;
  code = code.slice(0, scriptAddIndex) + injection + code.slice(scriptAddIndex);
}

// 2. update nowHero payload
code = code.replace(/if \(tournamentFinished\) \{[\s\S]*?tone: 'done',[\s\S]*?\};?\r?\n\t\t\}/m, 
`if (tournamentFinished) {
			const winnerCount = wonLeaguePlacements.length;
			const isGold = winnerCount > 0;
			return {
				tone: isGold ? 'gold' : 'done',
				kicker: isEnglish ? 'Tournament over' : 'Turneringa er over',
				title: isGold
					? isEnglish ? \`You won \${winnerCount} \${winnerCount === 1 ? 'league' : 'leagues'} 🥇🏆\` : \`Du vann \${winnerCount} \${winnerCount === 1 ? 'liga' : 'ligaer'} 🥇🏆\`
					: isEnglish ? 'World Cup is over 🎊🏆' : 'VM er over 🎊🏆',
				body: isGold 
					? isEnglish ? 'Fantastic predictions! Your final standings are below.' : 'Fantastisk tipping! Sjå sluttresultata dine under.'
					: activeLeagueRow
					? isEnglish
						? 'Your final league standings are ready below.'
						: 'Sluttresultata dine i ligaene ligg klare under.'
					: isEnglish
						? 'Thanks for playing. Your final standings are ready below.'
						: 'Takk for at du spelte. Sluttresultata dine ligg klare under.',
				href: leagueHref,
				label: isEnglish ? 'View standings' : 'Sjå sluttresultat'
			};
		}`);

// 3. update class:tourney-over -> class:tourney-gold
code = code.replace(/class:tourney-over=\{nowHero\.tone === 'done'\}/, 
	`class:tourney-over={nowHero.tone === 'done'}\n\t\tclass:tourney-gold={nowHero.tone === 'gold'}`);

// 4. update icon and colors
code = code.replace(/class:done=\{nowHero\.tone === 'done'\}/, 
	`class:done={nowHero.tone === 'done'} class:gold={nowHero.tone === 'gold'}`);
code = code.replace(/\{:else if nowHero\.tone === 'done'\}<Trophy size=\{18\} \/>/, 
	`{:else if nowHero.tone === 'done' || nowHero.tone === 'gold'}<Trophy size={18} />`);

// 5. top scorer in UI
const uiAddIndex = code.indexOf('<div class="league-results-list">');
if (uiAddIndex !== -1) {
  const htmlInject = `				{#if myGoldenBootPickName}
					<div class="topscorer-pick" style="padding:0.75rem; background:var(--surface-2); border-radius:12px; margin-bottom:1rem; display:flex; align-items:center; gap:0.75rem;">
						<div style="font-size:1.5rem;">{myGoldenBootPickIsWinner ? '🥇' : '⚽'}</div>
						<div>
							<div style="font-size:0.85rem; color:var(--muted)">{isEnglish ? 'Your Topscorer' : 'Din Toppscorer'}</div>
							<div style="font-weight:600; color:var(--fg)">{myGoldenBootPickName} 
								<span style="color:var(--gold); font-weight:700;">{myGoldenBootPickIsWinner ? (isEnglish ? '(Winner!)' : '(Vinnar!)') : ''}</span>
							</div>
						</div>
					</div>
				{/if}

				<div class="league-results-list" style="max-height: 250px; overflow-y: auto; margin-right:-0.5rem; padding-right:0.5rem;">`;
  code = code.replace('<div class="league-results-list">', htmlInject);
}

// 6. CSS inject
const cssInject = `
	.action-card.tourney-gold {
		border-color: color-mix(in srgb, var(--gold) 60%, var(--border));
		background: radial-gradient(circle at top right, color-mix(in srgb, var(--gold) 20%, var(--surface)) 0%, var(--surface) 120%);
		position: relative;
		overflow: hidden;
	}
	.action-card.tourney-gold::after {
		content: '🎊';
		position: absolute;
		right: 1.5rem;
		top: -0.5rem;
		font-size: 6rem;
		opacity: 0.15;
		transform: rotate(15deg);
		pointer-events: none;
	}
	.now-icon.gold {
		background: color-mix(in srgb, var(--gold) 15%, transparent);
		color: var(--gold);
	}
`;
if (code.indexOf('.action-card.tourney-gold') === -1) {
  code = code.replace('.action-card.tourney-over {', cssInject + '\n\t.action-card.tourney-over {');
}

fs.writeFileSync('frontend/src/routes/+page.svelte', code);
console.log('done');
