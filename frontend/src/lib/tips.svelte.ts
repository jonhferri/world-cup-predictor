import { pb } from './pb';
import { auth } from './auth.svelte';
import { serverClock } from './serverclock.svelte';

export interface Team {
	id: string;
	name: string;
	iso2: string;
	fifaCode: string;
}

export interface Match {
	id: string;
	stage: string; // group | R32 | R16 | QF | SF | 3RD | FINAL
	groupLetter: string;
	roundLabel: string;
	num: number;
	kickoff: string;
	tvChannel: string;
	status: string;
	homeTeam: string;
	awayTeam: string;
	homeLabel: string;
	awayLabel: string;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penHome: number;
	penAway: number;
	advancer: string;
	finalizedAt: string;
}

export interface Tip {
	id?: string;
	match: string;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penWinner: string;
	advancer: string;
	firstTeam: string;
	firstPlayer: string;
	turbo?: boolean;
}

export interface Player {
	id: string;
	name: string;
	position: string;
	teamId: string;
}

export interface MatchOdds {
	matchId: string;
	pHome: number;
	pDraw: number;
	pAway: number;
	homeOdds: number;
	drawOdds: number;
	awayOdds: number;
}

export interface TipComponents {
	tendency: number;
	exact: number;
	totalGoals: number;
	goalDiff: number;
	firstTeamScorer: number;
	firstPlayerScorer: number;
	turbo: boolean;
}

export interface FriendTip {
	userId: string;
	name: string;
	isMe: boolean;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penWinner: string;
	advancer: string;
	firstTeam: string;
	firstPlayer: string;
	turbo?: boolean;
	points: number; // -1 = no tip submitted
	components?: TipComponents;
}

class TipsStore {
	teams = $state<Record<string, Team>>({});
	matches = $state<Match[]>([]);
	tips = $state<Record<string, Tip>>({}); // keyed by matchId
	scores = $state<Record<string, number>>({}); // matchId -> points (default cfg)
	tournamentGroups = $state<Record<string, string[]>>({}); // letter -> teamIds
	odds = $state<Record<string, MatchOdds>>({}); // keyed by matchId
	oddsSource = $state<'odds_api' | 'rankings' | 'none'>('none');
	loaded = $state(false);
	private loadPromise: Promise<void> | null = null;
	private playerCache = new Map<string, Player[]>(); // teamId -> players

	async load() {
		if (this.loaded) return;
		if (this.loadPromise) return this.loadPromise;
		this.loadPromise = this.loadInner().finally(() => {
			this.loadPromise = null;
		});
		return this.loadPromise;
	}

	private async loadInner() {
		const [teams, matches, mine, tgroups] = await Promise.all([
			pb.collection('teams').getFullList({ sort: 'name' }),
			pb.collection('matches').getFullList({ sort: 'kickoff' }),
			pb
				.collection('tips')
				.getFullList({ filter: `user = "${auth.user?.id}"` }),
			pb.collection('tournament_groups').getFullList({ sort: 'letter' }),
			serverClock.refresh(),
			pb
				.send('/api/tips/scores', { method: 'GET' })
				.then((r) => (this.scores = r.scores ?? {}))
				.catch(() => {})
		]);
		const gmap: Record<string, string[]> = {};
		for (const g of tgroups) gmap[g.letter] = g.teams ?? [];
		this.tournamentGroups = gmap;
		const tmap: Record<string, Team> = {};
		for (const t of teams)
			tmap[t.id] = {
				id: t.id,
				name: t.name,
				iso2: t.iso2,
				fifaCode: t.fifaCode
			};
		this.teams = tmap;
		this.matches = matches as unknown as Match[];
		const tip: Record<string, Tip> = {};
		for (const r of mine)
			tip[r.match] = {
				id: r.id,
				match: r.match,
				ftHome: r.ftHome,
				ftAway: r.ftAway,
				etHome: r.etHome,
				etAway: r.etAway,
				penWinner: r.penWinner,
				advancer: r.advancer,
				firstTeam: r.firstTeam ?? '',
				firstPlayer: r.firstPlayer ?? '',
				turbo: r.turbo ?? false
			};
		this.tips = tip;
		this.loaded = true;

		// Odds are non-critical — load in background, silently skip on failure.
		pb.send('/api/odds', { method: 'GET' })
			.then((r) => {
				this.oddsSource = r.source ?? 'none';
				const map: Record<string, MatchOdds> = {};
				for (const o of r.odds ?? []) map[o.matchId] = o;
				this.odds = map;
			})
			.catch(() => {});
	}

	team(id: string): Team | undefined {
		return this.teams[id];
	}

	/** Returns the set of stage-group keys that already have a turbo tip saved.
	 *  Group matches key on roundLabel ("Matchday 1/2/3"); KO on stage ("R32"…).
	 *  Call from a reactive context so Svelte tracks tips + matches as deps. */
	turboedStageGroups(): Set<string> {
		const result = new Set<string>();
		for (const [matchId, tip] of Object.entries(this.tips)) {
			if (!tip.turbo) continue;
			const m = this.matches.find((x) => x.id === matchId);
			if (!m) continue;
			result.add(m.stage === 'group' ? groupStageBucket(m.roundLabel) : m.stage);
		}
		return result;
	}

	/** Save (create or update) a tip; throws with the server message on a
	 *  rule/validation failure so the UI can show it. */
	async save(t: Tip): Promise<void> {
		const user = auth.user?.id;
		if (!user) throw new Error('You must be signed in to save tips.');
		const data = {
			user,
			match: t.match,
			ftHome: t.ftHome,
			ftAway: t.ftAway,
			etHome: t.etHome,
			etAway: t.etAway,
			penWinner: t.penWinner || '',
			firstTeam: t.firstTeam || '',
			firstPlayer: t.firstPlayer || '',
			turbo: t.turbo ?? false
		};
		let rec;
		if (t.id) {
			rec = await pb.collection('tips').update(t.id, data);
		} else {
			try {
				rec = await pb.collection('tips').create(data);
			} catch (createError) {
				try {
					const existing = await pb
						.collection('tips')
						.getFirstListItem(`user = "${user}" && match = "${t.match}"`);
					rec = await pb.collection('tips').update(existing.id, data);
				} catch {
					throw createError;
				}
			}
		}
		this.tips[t.match] = {
			id: rec.id,
			match: rec.match,
			ftHome: rec.ftHome,
			ftAway: rec.ftAway,
			etHome: rec.etHome,
			etAway: rec.etAway,
			penWinner: rec.penWinner,
			advancer: rec.advancer,
			firstTeam: rec.firstTeam ?? '',
			firstPlayer: rec.firstPlayer ?? '',
			turbo: rec.turbo ?? false
		};
	}

	async friends(matchId: string): Promise<FriendTip[]> {
		const r = await pb.send(`/api/tips/others/${matchId}`, {
			method: 'GET'
		});
		return (r.tips ?? []).map((t: FriendTip & { components?: string }) => ({
			...t,
			components: typeof t.components === 'string' && t.components
				? JSON.parse(t.components)
				: t.components
		}));
	}

	async playersForTeams(teamIds: string[]): Promise<Player[]> {
		const missing = teamIds.filter((id) => !this.playerCache.has(id));
		if (missing.length > 0) {
			const filter = missing.map((id) => `teamId = "${id}"`).join(' || ');
			const rows = await pb
				.collection('players')
				.getFullList({ filter, sort: 'name' });
			// Group by teamId and populate cache.
			const byTeam = new Map<string, Player[]>();
			for (const r of rows) {
				const entry: Player = {
					id: r.id,
					name: r.name,
					position: r.position,
					teamId: r.teamId
				};
				if (!byTeam.has(r.teamId)) byTeam.set(r.teamId, []);
				byTeam.get(r.teamId)!.push(entry);
			}
			for (const id of missing) {
				this.playerCache.set(id, byTeam.get(id) ?? []);
			}
		}
		return teamIds.flatMap((id) => this.playerCache.get(id) ?? []);
	}
}

export const tipsStore = new TipsStore();

// groupStageBucket maps "Matchday N" → a stable turbo-slot key.
// WC 2026: 1-7 = group-1, 8-13 = group-2, 14-17 = group-3.
export function groupStageBucket(roundLabel: string): string {
	const n = parseInt(roundLabel.replace('Matchday ', ''), 10);
	if (n >= 14) return 'group-3';
	if (n >= 8) return 'group-2';
	return 'group-1';
}

export function isLocked(m: Match): boolean {
	return serverClock.now() >= new Date(m.kickoff).getTime();
}
export function teamsResolved(m: Match): boolean {
	return !!m.homeTeam && !!m.awayTeam;
}
