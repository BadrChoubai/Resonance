# Plan

## Goal

A guided journaling app that uses your Spotify listening to prompt reflection on mood. The user picks one of their top three songs and writes about why they chose it.

## Spotify constraints (checked October 2026)

These shape v1. Re-verify against Spotify's docs when registering the app, since the rules have changed several times.

- **No one-week window.** `GET /me/top/tracks` supports `time_range` of `short_term` (~4 weeks), `medium_term` (~6 months), and `long_term` (~1 year). `GET /me/player/recently-played` returns at most 50 plays, so a week can't be reconstructed from it without storing play history ourselves.
- **Development mode only.** Max five allowlisted users per app, and the app owner needs Premium. Extended quota is only open to registered organizations with 250k+ monthly active users. v1 is therefore a personal app for the owner plus up to four others, added by hand in the Spotify dashboard.
- **Removed endpoints.** The February 2026 migration removed batch lookups (`GET /tracks`, `/albums`, `/artists`), artist top-tracks, and others. Top items and recently-played were not on the removed lists I found; confirm against the changelog.

References:

- Top items: https://developer.spotify.com/documentation/web-api/reference/get-users-top-artists-and-tracks
- February 2026 changelog: https://developer.spotify.com/documentation/web-api/references/changes/february-2026
- Migration guide: https://developer.spotify.com/documentation/web-api/tutorials/february-2026-migration-guide

## v1 scope

**In:**

- Spotify login
- Fetch the user's top 3 tracks
- "New journal" screen where the user selects one of the three
- Installable Vue PWA
- Deployed to the Kubernetes cluster

**Out:**

- Prompts / guided questions
- Writing and saved entries
- Mood or valence analysis
- Non-Spotify accounts
- Public signups
- An API gateway

**Assumptions (change if wrong):**

- Time window is `short_term` (~4 weeks), labelled "your top songs lately" in the UI. The backend takes the window as a parameter so a true 7-day version can be added later.
- Selecting a song persists nothing in v1. "Open a new journal" means the entry screen, not storage.
- The app is for the owner and a handful of friends, not the public.

## Architecture

Vue PWA → Ingress → Go API

- **One Go API service**, with `auth` and `spotify` as separate packages inside it. No gateway: with one backend there is nothing to route between. Revisit only if a second service appears.
- **Auth:** authorization-code flow handled server-side. Go holds the client secret and refresh token. The browser gets only an httpOnly session cookie and never sees Spotify tokens.
- **Scope requested:** `user-top-read`.
- **Stored data (minimal):** Spotify user ID and an encrypted refresh token.
- **Routing:** the Go service registers its public routes under `/api` (`/api/auth/login`, `/api/me`, ...). The Ingress sends `/api` to the Go service unchanged and `/` to the frontend. In dev, the Vite proxy forwards `/api` to Go with no rewrite, so paths are identical in dev and prod. `/healthz` and `/metrics` live outside `/api`, so the Ingress never exposes them.
- **Redirect URI:** `http://127.0.0.1:5173/api/auth/callback` in dev (Spotify accepts loopback only as `127.0.0.1`), the cluster hostname's `/api/auth/callback` in prod.

## Milestones

Each milestone ends in something runnable.

1. **Scaffold.** Repo layout, hello-world Go server (`GET /api/hello`, `GET /healthz`), Vue PWA from Vite, Dockerfiles, Makefile.
   _Done when:_ `make dev` shows a page displaying a response from the API.
2. **Login.** Register the Spotify app; add `/api/auth/login`, `/api/auth/callback`, `/api/me`; login button and logged-in state in the UI.
   _Done when:_ you can log in and see your Spotify display name.
3. **Top tracks API.** `GET /api/top-tracks` returns three tracks (id, title, artist, art); refreshes expired tokens; handles 429s and users with fewer than three tracks.
   _Done when:_ it returns correct data for your account via curl.
4. **Song selection UI.** "New journal" shows three cards; selecting one shows a confirmation state.
   _Done when:_ the full flow works in the browser.
5. **PWA and deploy.** Manifest, service worker, k8s manifests, Ingress, secrets for the client secret and session key.
   _Done when:_ the app is installable and running in the cluster.

## Later (not v1)

- Guided prompts and written entries, with storage for them
- True 7-day top songs (requires storing play history)
- Mood features; check what audio-feature data is still available to new Spotify apps before designing around it
- Splitting the API into separate services (and only then, a gateway)
