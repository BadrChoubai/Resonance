# _Résonance_

A personal mood intelligence platform that combines passive Spotify listening data with intentional CFT-style journaling to surface emotional patterns over time.

The listening service syncs your Spotify history and derives a mood signal from each session's audio features. The journal service captures structured CFT reflections - situation, self-critical thought, compassionate reframe. Together they build a 9 longitudinal picture of your emotional state that neither could produce alone. The project is built as a production-style microservices architecture - two domain services behind an API gateway, each with its own database, orchestrated via Docker Compose.
