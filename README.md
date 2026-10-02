# _Résonance_

A personal mood intelligence platform that combines Spotify listening data with intentional CFT-style journaling to surface emotional patterns over time.

You start a journal entry by picking one of the songs you've had on repeat lately, then reflect on why you chose it. Over time, entries are meant to grow into structured CFT reflections (situation, self-critical thought, compassionate reframe) that, alongside your listening, build a longitudinal picture of your emotional state.

## Architecture

A Vue PWA and a single Go API, deployed to Kubernetes. The Ingress sends `/api` to the Go service and everything else to the frontend. The Go service handles the Spotify OAuth flow server-side, so the browser only ever holds an httpOnly session cookie.

The API starts as one service and gets split only once journaling brings a real reason to deploy parts independently.

See [PLAN.md](PLAN.md) for scope, Spotify API constraints, and milestones.

## Development

Requires Go, Node.js, and npm.

```sh
make dev    # Go API on :8080 and Vite on http://127.0.0.1:5173
make help   # list all targets
```
