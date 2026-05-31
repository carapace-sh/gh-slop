# gh-slop

Experimental [GitHub CLI] extension that leverages [Crush] to handle slop contributions and serves as playground for [Slopware].

[![asciicast](https://asciinema.org/a/1168742.svg)](https://asciinema.org/a/1168742)

## Core Concepts

### Crush Integration

`gh-slop` is built around [Crush], an AI-powered CLI assistant. Running `gh slop` without a subcommand launches Crush with a preconfigured `crush.json` that registers the extension's MCP server and deploys a `slop-detect` skill. This allows Crush to autonomously discover repos, scan for slop PRs, profile authors, and classify contributions — all through the MCP tools below.

### MCP Server

The extension provides a stdio-based [MCP] server (`gh slop mcp`) with four tools:

| Tool | Description |
|------|-------------|
| `list-repos` | Lists the user's writable GitHub repositories |
| `list-sloppers` | Lists open PRs from new/low-contribution authors |
| `profile-sloppers` | Fetches detailed GitHub profiles for given usernames (account age, commit count, PR distribution, merge rate, recent PRs) |
| `slop-prs` | Fetches title, body, author, and metadata for a list of PRs in `OWNER/REPO#NUMBER` format |

The server is hand-rolled JSON-RPC 2.0 over stdio — no SDK dependency.

### Slop Detection Skill

The embedded `slop-detect` skill teaches Crush how to analyze slop. It provides a structured workflow:

1. **Discover** repos via `list-repos`
2. **Scan** for new contributors via `list-sloppers`
3. **Profile** each author via `profile-sloppers` (batched)
4. **Classify** each PR via `slop-prs` (batched) — checks for duplicates, AI agent markers, bounty claims, self-promo
5. **Identify patterns** — issue racing, burst spraying, coordinated accounts, AI agent slop, etc.
6. **Classify authors** — slop, likely slop, not slop, AI-assisted but legitimate
7. **Present results** — summary statistics, author table, pattern breakdown, recommendations

### Updating the Skill

The `slop-detect` skill evolves over time as new slop patterns emerge. To update it:

1. Run a detection pass (e.g. `gh slop detect`)
2. After reviewing results, ask Crush to identify any **new or recurring patterns** not yet covered by the skill
3. Crush includes a built-in skill (`crush-config`) that can update skill files — use it to incorporate the new patterns into `slop-detect`

Patterns that are specific to a particular org, tool, or repo should **not** go into the general `slop-detect` skill. Instead, they belong in separate `slop-patterns-*` skill files:

| Skill | Scope |
|-------|-------|
| `slop-patterns-carapace` | carapace-sh specific patterns (completer racing, template spraying) |
| `slop-patterns-awesome` | awesome-list specific patterns (self-promo spam) |
| `slop-patterns-rust` | Rust ecosystem specific patterns |

These project-specific skills contain org-specific signals, known offender classifications, extra analysis steps, and concrete examples. The `slop-detect` skill automatically loads any matching `slop-patterns-*` skill when triggered for a relevant project.

## Macro Export

`gh-slop` exports two [carapace] macros via `carapace-spec`, making its completion actions available in YAML user specs:

| Macro | Type | Description |
|-------|------|-------------|
| `gh-slop.Repos` | `MacroN` | Completes writable repositories in `owner/repo` format (24h cache) |
| `gh-slop.Sloppers` | `MacroV` | Completes usernames of low-contribution authors with slop counts and styling (15min cache, accepts repo list) |

### Usage in Specs

```yaml
# yaml-language-server: $schema=https://carapace.sh/schemas/command.json
name: slop-pr
run: "$(gh pr list --repo \"${C_ARG0}\" --author \"${C_ARG1}\")"
completion:
  positional:
    - ["$gh-slop.Repos ||| $multiparts([/])"]
    - ["$gh-slop.Sloppers([${C_ARG0}])"]
```

[Crush]: https://github.com/charmbracelet/crush
[GitHub CLI]: https://cli.github.com
[Slopware]: https://carapace-sh.github.io/carapace-bin/slopware.html
[MCP]: https://modelcontextprotocol.io
[carapace]: https://github.com/carapace-sh/carapace
