# Documentation

User-facing documentation lives under `docs/` and is published as the mkdocs site. **Only update it when a change affects configuration or user-visible behavior.** Pure refactors, internal renames, test changes, and implementation tweaks that leave the public surface unchanged do not require doc updates.

Update `docs/` when a change:
- adds, removes, or renames a config option (CLI flag, TOML key, env var, default value)
- changes the runtime behavior a user can observe (proxy modes, DNS resolution, fake-packet behavior, TUI interactions, exit codes, log output users rely on)
- changes install, build, or run instructions

Place updates in the section that matches the change:
- `docs/user-guide/` — config options and runtime behavior (`app.md`, `connection.md`, `dns.md`, `https.md`, `udp.md`, `policy.md`, `configuration-recipes/`, etc.)
- `docs/getting-started/` — install, quick-start, introduction
- `docs/developer-guide/` — build/test/lint workflow, commit conventions

If unsure whether a change is user-visible, default to **no doc update** and mention it so the user can decide.
