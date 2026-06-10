# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

World Cup Pool is a self-hosted WC 2026 prediction game. The entire app ships as a **single binary / single Docker image**: a Go/PocketBase backend embeds a compiled SvelteKit SPA and serves both the REST API and the frontend from the same port (`:8090` in prod, `:8091` in dev/test).

## Commands

```sh
make install          # Install frontend npm deps (run once after clone)
make dev-backend      # Go backend on :8091 with disposable pb_data_dev
make dev-frontend     # SvelteKit dev server on :5173, proxies /api to :8091
make test             # Go tests
make build            # Build frontend then embed + compile single binary

cd frontend && npm run check      # Svelte type-check
cd frontend && npm test -- --run  # Vitest unit tests (single run)
cd frontend && npm run test:e2e   # Playwright e2e tests

make reset            # Wipe pb_data_dev for a clean local DB
make build-frontend   # Rebuild frontend embed only (internal/web/build)
```

## Architecture

### Backend (`main.go` + `internal/`)

- **Runtime**: [PocketBase](https://pocketbase.io/) provides auth, SQLite, REST, admin UI, and hooks. All custom logic registers on `app.OnServe()`.
- **Database migrations** live in `migrations/` as numbered Go files. PocketBase runs them automatically on `app.Start()` via `migratecmd`.
- **Seeding**: `internal/seed` populates teams, groups, and fixtures from the embedded openfootball dataset on first boot (idempotent).
- **Custom API packages** each expose a `Register(app, event)` function wired in `main.go`:
  - `internal/scoring` — match tip scoring (up to 6 pts) and tournament forecast scoring; triggered on every result change; scoring weights are stored as PocketBase `scoring_configs` records and editable without redeploy
  - `internal/tips` — match tip submission/locking; tips lock at kickoff via the `internal/clock` server clock
  - `internal/forecast` — full-tournament bracket predictions (groups → knockout → winner) + Golden Boot picks
  - `internal/sync` — cron (`*/30 * * * *`) + superuser endpoint to pull live results from API-Football or openfootball; `RESULTS_SOURCE` env var selects the provider
  - `internal/odds` — syncs bookmaker odds from The Odds API; falls back to FIFA-ranking-based probabilities when no key is configured
  - `internal/leagues` — private leagues with invite codes, invitations, and member management
  - `internal/standings` — group stage table calculations
  - `internal/bracket` — knockout bracket state
  - `internal/topscorer` — Golden Boot (top scorer) tracking
  - `internal/chat` — per-league chat
  - `internal/account` — signup alerts, user stats
  - `internal/oauth` — optional Google OAuth
  - `internal/web` — embeds the compiled SvelteKit build (`internal/web/build/`) and serves it with SPA fallback

### Frontend (`frontend/`)

SvelteKit 2 + Svelte 5 SPA built with `@sveltejs/adapter-static`. The compiled output is embedded into the Go binary via `internal/web/embed.go`.

- **API layer**: `src/lib/pb.ts` — PocketBase JS SDK instance (same-origin; `autoCancellation(false)` to allow parallel requests). `src/lib/api.ts` wraps custom Go endpoints via `pb.send()`.
- **State**: Svelte 5 runes (`*.svelte.ts` files in `src/lib/`). Auth state is in `auth.svelte.ts`, tips in `tips.svelte.ts`, forecast in `forecast.svelte.ts`, chat in `chat.svelte.ts`.
- **Routes**: file-based. Key routes: `/` home, `/tips` match tips, `/forecast` tournament predictions, `/leagues` / `/leagues/[id]`, `/tournament` bracket view, `/settings`.
- **i18n**: Bokmål / Nynorsk / English — all user-facing strings must be available in all three. String tables live in `src/lib/strings.ts` and `src/lib/language.svelte.ts`.
- **Dev proxy**: `vite.config.ts` proxies `/api` and `/_` to `VITE_API_ORIGIN` (default `:8091`) so prod is never touched during development.

### Data flow for a match result update

1. Admin posts result (manual endpoint) **or** sync cron fires.
2. `internal/sync` updates the `matches` record.
3. PocketBase record hooks in `internal/scoring` trigger a full idempotent recompute of all `tips` and `forecasts` for every league that owns a `scoring_config`.
4. Frontend receives updated leaderboard/standings via PocketBase realtime subscriptions or next page load.

## Key Conventions

- **Dev data is isolated**: `pb_data_dev` (local) and the `:8091` test Docker container are completely separate from prod (`:8090` / `pb_data`). Never run against `:8090` during development.
- **Frontend build must be current** before `go build` — run `make build-frontend` if only Go changes are needed but the embed is stale.
- **Migrations are code**: add new `migrations/NNNN_description.go` files; do not edit existing ones.
- **Scoring weights are data**: change point values in PocketBase admin UI (`scoring_configs`), not in Go code.
- **Language parity**: any new user-facing string needs Bokmål, Nynorsk, and English entries.
- **No local Go**: Go is not installed on the host. Always use `docker compose build` or `docker compose up --build` to compile backend changes — never `go build` directly.
- **Language is always English**: `language.svelte.ts` is simplified to always return the English argument. All `language.text(nb, nn, en)` call sites work unchanged.

## In-progress work (session checkpoint)

### What was completed (all verified via `docker compose build`)

**Backend:**
- `migrations/0020_first_scorer_fields.go` — adds `firstTeam`/`firstPlayer` to `tips`, `firstTeamScorer`/`firstPlayerScorer` to `matches`, updates `scoring_configs`
- `migrations/0021_players.go` — creates `players` collection (name, position, teamId where teamId is the PocketBase record ID of the team)
- `internal/seed/seed.go` — `SeedPlayers()` reads `internal/seed/data/players2026.csv`; maps 48 Portuguese CSV headers → English DB team names → team record IDs. **Fixed**: the early-return path (when teams already exist) now also calls `SeedPlayers()`, so players are seeded on existing instances too.
- `internal/scoring/scoring.go` — first scorer scoring (+5/+10), `deriveGroupOrder()`, rewritten `scoreForecast()` auto-derives group/thirds/KO from user's match tips; golden boot still read from the forecast record directly
- `internal/scoring/recompute.go` — scores every user who has tips (even without a forecast record)
- `internal/tips/tips.go` — `/api/tips/others/{matchId}` returns `firstTeam` and `firstPlayer`

**Frontend:**
- `frontend/src/lib/tips.svelte.ts` — `Player` interface, `firstTeam`/`firstPlayer` on `Tip`/`FriendTip`, `playersForTeams()` method (fetches from `players` collection filtered by team record ID)
- `frontend/src/lib/components/TipCard.svelte` — "First team to score" toggle + "First player to score" searchable dropdown; locked view shows saved picks
- `frontend/src/lib/language.svelte.ts` — always returns English; `set()`/`toggle()` are no-ops
- `frontend/src/routes/+layout.svelte` — removed LanguageToggle
- `frontend/src/app.html`, `manifest.webmanifest`, `Logo.svelte`, `strings.ts`, join/info/PWA pages — renamed to "Cozinhámos Predictions", lang=en
- `frontend/src/lib/strings.ts` — nav label `worldCupTips` changed to `'Golden Boot'` (EN) / `'Toppscorer'` (NB/NN)
- `frontend/src/routes/forecast/+page.svelte` — rewritten to Golden Boot only (~280 lines, was 1199); groups/thirds/bracket manual forms removed
- `frontend/src/routes/info/+page.svelte` — updated: flow step 1 explains auto-derive, match scoring table adds first scorer rows (+5/+10), max per match updated to 21, forecast panel renamed and description updated

### Next task: update scoring rules

The user wants these new point values for match tips:

| Component | New points |
|---|---|
| Correct result (win / draw / loss) | 5 |
| Exact goals (home or away — either team's goal count exactly right) | 5 |
| Exact goal difference | 5 |
| Exact result (both goals exactly right) | 10 |
| First team to score | 5 |
| First player to score | 10 |

**What needs to change:**

1. **`internal/scoring/scoring.go`** — the `scoreValues()` function currently computes:
   - `Tendency` (correct 1/X/2 outcome) — keep concept, default value 3 → 5
   - `TotalGoals` (home+away total matches) — **semantics change**: becomes "exact goals home OR away" (award if `tipHome == resHome || tipAway == resAway`)
   - `GoalDiff` (correct goal difference) — keep concept, default value 1 → 5
   - `Exact` (both goals exact) — keep concept, default value 1 → 10
   - `FirstTeamScorer` — already implemented at 5 (no change)
   - `FirstPlayerScorer` — already implemented at 10 (no change)
   The `Config.Match` struct field names stay the same; only the computation of `TotalGoals` changes.

2. **`migrations/0022_scoring_update.go`** — new migration to update default values in all existing `scoring_configs` records:
   - `match.tendency`: 3 → 5
   - `match.totalGoals`: 1 → 5  (also rename display label if shown anywhere)
   - `match.goalDiff`: 1 → 5
   - `match.exact`: 1 → 10
   - `match.firstTeamScorer`: already 5 (leave as-is)
   - `match.firstPlayerScorer`: already 10 (leave as-is)
   Pattern to follow: see `patchFirstScorerConfig()` in `migrations/0020_first_scorer_fields.go` for how to patch all existing `scoring_configs` JSON blobs.

3. **`internal/seed/seed.go`** — update `DefaultScoringConfig` map with new default values (for fresh installs).

4. **`frontend/src/routes/info/+page.svelte`** — update `matchPoints` array:
   - "Correct outcome" value `'3'` → `'5'`
   - "Exact score" value `'+1'` → `'+10'`
   - "Total goals" label → "Exact goals (home or away)", value `'+1'` → `'+5'`
   - "Correct goal difference" value `'+1'` → `'+5'`
   - First team/player rows already correct at +5/+10
   - Match panel description: already says "Max 21 points" (correct: 5+5+5+10+5+10 = 40... need to recalculate. With all stacking: 5+5+5+10+5+10 = 40 max)

5. **`frontend/src/lib/components/TipCard.svelte`** — if any point values are hardcoded in the UI, update them. (Currently no point values are shown in TipCard so likely no change needed here.)

### Next task (UI): compact TipCard first-scorer section

The first-scorer section in `frontend/src/lib/components/TipCard.svelte` is too visually crowded. The player dropdown in particular feels packed. Goals:
- Tighten spacing/padding in the `.first-scorer-section` and `.player-picker` area
- The "First team to score" buttons and "First player to score" input should feel like a natural extension of the card, not a bolted-on block
- Consider making the team toggle buttons smaller/inline rather than full-width
- The player search input + dropdown should feel lightweight — less padding, smaller font if needed
- Test on mobile (narrow) and desktop widths
- No functional changes, layout/spacing only

### Optional follow-up (lower priority)

- Clean up dead code in `scoring.go`: `fcResolver`, `assignThirds`, `resolve` functions are unreachable after the `scoreForecast` rewrite
- Simplify `forecast.svelte.ts` store: `groupOrder`, `thirds`, `bracket` state and their load/save logic can be stripped (still harmlessly saved/loaded but never shown in UI)
- Check if `forecast.svelte.ts` exports `koKey`, `KOMatch` are still used elsewhere before removing
