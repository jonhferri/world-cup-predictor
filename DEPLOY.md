# Deploy

World Cup Pool ships as **one self-contained Docker image**: the Go binary
serves the API and the embedded SvelteKit SPA from a single port, with SQLite
data on a mounted volume.

For backwards compatibility, the Docker image still exposes the `wm-pickems`
binary name and the default compose file still uses the `fhun_tips` container
name. Do not change those names in an existing production instance unless you
also plan a controlled migration.

## 1. Configure

```sh
cp .env.example .env
```

| Var | Needed | Notes |
|-----|--------|-------|
| `HTTP_PORT` | no | Host port (default `8090`). |
| `PUBLIC_APP_URL` | recommended | Public origin used for invite Open Graph images and URLs, for example `https://vm.midttunet.no`. If omitted, the app derives it from forwarded headers / host. |
| `API_FOOTBALL_KEY` | optional | Leave empty unless you have your own API-Football key. |
| `RESULTS_SOURCE` | no | `auto` (default): API-Football if its key reaches WC2026, else the free **openfootball** JSON. Force with `apifootball` / `openfootball`. Manual override always works. openfootball is community-updated (hours, not real-time). |
| `PB_ADMIN_EMAIL` / `PB_ADMIN_PASSWORD` | optional | Convenience only. Use your own values, never the example values. See superuser step below. |

## 2. Run

```sh
docker compose up --build -d
```

App + API: `http://<host>:${HTTP_PORT}`. Data persists in the `pb_data`
Docker volume (SQLite DB, uploaded files, logs). First boot auto-runs
migrations and seeds 48 teams / 12 groups / 104 fixtures.

## 3. Create an admin (superuser)

The PocketBase admin UI (`/_/`) and the admin endpoints
(`/api/sync/refresh`, `/api/admin/matches/{id}/result`,
`/api/admin/recompute`) require a superuser:

```sh
docker compose exec app wm-pickems superuser create you@example.com 'a-strong-pass' --dir=/pb_data
```

## 4. Configure mail / password reset

Password-reset requests use PocketBase's built-in auth email flow. The app
route and API can work without SMTP, but a real reset email is only delivered
after mail is configured:

1. Open `http://<host>:${HTTP_PORT}/_/` and sign in as the superuser.
2. Go to **Settings → Application** and set the public Application URL.
3. Go to **Settings → Mail**, set sender name/address, enable SMTP, and enter
  your SMTP host, port, username/password, TLS and auth method.
4. Use **Send test email** in PocketBase. Treat this as the delivery proof.
5. Then test the app flow from `/forgot-password`: the API should return `204`,
  the inbox should receive a link to `/confirm-password-reset/<token>`, and
  that page should accept a new password.

Without SMTP, a `/forgot-password` request can still return `204` because
PocketBase intentionally does not reveal whether an address exists, but no
external email delivery has been proven.

### Signup email alerts

When SMTP is configured, the backend sends a notification email to the admin
whenever a new user account is created (both email/password and Google OAuth).

Set the recipient in `.env`:

```env
SIGNUP_ALERT_EMAIL=you@example.com
```

If `SIGNUP_ALERT_EMAIL` is not set, it falls back to `PB_ADMIN_EMAIL`. If
neither is set, no alert is sent. Failed delivery is logged but never blocks
the signup. Dev-bot accounts (suffix `@dev.local`) are skipped.

## 5. Operating

- **Results**: synced every 30 min from the active source (openfootball by
  default, or a paid API-Football). Force one: `POST /api/sync/refresh`
  (superuser) — returns the source used.
- **Odds**: when `ODDS_API_KEY` is set, bookmaker odds sync at startup and
  then daily at `07:00 UTC` and `18:30 UTC` (`20:30` CEST during the
  tournament). Without a key, the tips UI falls back to FIFA-ranking-based
  probabilities.
- **Manual override / fix a result**: `POST /api/admin/matches/{id}/result`
  with `{ "FTHome":2, "FTAway":1, "Status":"finished" }` (also `ETHome/ETAway`,
  `PenHome/PenAway` for knockout). Scores recompute automatically.
- **Recompute everything** (after changing a scoring config):
  `POST /api/admin/recompute` (superuser).
- **Scoring config**: edit the `scoring_configs` "Default" record in `/_/`
  (or a per-League override) — no redeploy. Note: a config change (or a
  schema migration that rewrites it) does **not** retro-rescore matches that
  are already finished until you call `POST /api/admin/recompute` (or the
  next result comes in, which recomputes automatically).

## 6. Backup

The whole app state is the `/pb_data` volume. Snapshot it while running:

```sh
docker run --rm --volumes-from fhun_tips -v "$PWD":/backup alpine \
  tar czf /backup/pb_data-backup.tgz -C /pb_data .
```

Restore by extracting the archive back into an empty data volume before `up`.
Keep backups outside git. They can contain users, emails, league data, chat,
avatars, and secrets.

## 7. TLS / reverse proxy

Terminate TLS at a proxy (Caddy/Traefik/nginx) and forward to the container
port. Example Caddy:

```
pickems.example.com {
    reverse_proxy localhost:8090
}
```

## 8. Updating

```sh
git pull
docker compose up --build -d   # migrations run automatically on boot
```

## Health

`GET /api/health` returns 200 when up — use it for container/proxy health
checks.
