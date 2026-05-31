# AGENTS.md

## Project Overview

`gh-slop` is a GitHub CLI (`gh`) extension that identifies "slop" contributions — open pull requests from contributors with few prior merged PRs. It helps maintainers spot low-effort or drive-by contributions. The extension uses GitHub's GraphQL and REST APIs via the `go-gh/v2` library.

## Commands

```bash
# Build
go build -o gh-slop .

# Install as a gh extension (from project root)
gh extension install .

# Run
gh slop                    # launches Crush (AI assistant) with gh-slop MCP configured
gh slop list               # lists PRs from new/low-contribution authors
gh slop list -m 3          # only show authors with fewer than 3 merged PRs
gh slop -R owner/repo list # target a specific repository
gh slop mcp                # starts the MCP stdio server (hidden command)

# Test
go test ./...
```

There is no linter config or Makefile in the project. There are no tests currently.

## Architecture

```
main.go                → entry point, calls cmd.Execute()
cmd/root.go            → root cobra command, --repo flag (comma-separated multi-repo), ResolveRepos(), carapace/spec setup
cmd/list.go            → `list` subcommand, renders PRs via pkg/render
cmd/mcp.go             → `mcp` subcommand (hidden), starts MCP stdio server
pkg/slop/slop.go       → core logic: fetch PRs, fetch contribution counts, filter new contributors
pkg/slop/profile.go   → batch user profile fetching (account age, commits, PR stats) via GraphQL
pkg/slop/repos.go      → ResolveRepos() (flag→Repository parsing), AccessibleRepos() (REST API for user's writable repos)
pkg/render/render.go   → terminal output: groups by author, time-cluster coloring, lipgloss styling
pkg/mcp/server.go      → hand-rolled MCP stdio server (JSON-RPC 2.0 with Content-Length framing)
pkg/actions/repo.go     → carapace completion action for repositories (ActionRepos)
pkg/actions/slopper.go  → carapace completion action for slopper usernames (ActionSloppers, MacroV)
pkg/crush/crush.go      → Crush integration: deploys embedded crush.json config, launches `crush` CLI
pkg/crush/crush.json    → embedded config: registers gh-slop as MCP server for Crush
```

**Data flow**: `cmd/list.go` calls `slop.ListNewContributors(repos, minContributions)` which:
1. Fetches all open PRs via paginated GraphQL (`fetchPullRequests`) — concurrently across multiple repos
2. For each unique author, concurrently fetches their merged PR count via GraphQL search (`filterNewContributors`, semaphore-limited to 5 concurrent requests)
3. Filters PRs where the author's merged count is below the threshold
4. Results are rendered by `render.Render()` which groups by author, sorts chronologically, and applies time-cluster coloring

**Multi-repo flow**: The `--repo`/`-R` flag accepts comma-separated repos (`StringSliceVarP`). `ListNewContributors` processes all repos concurrently (also semaphore-limited to 5). When multiple repos are targeted, each PR is prefixed with `owner/repo#` in output.

**MCP server**: `cmd/mcp.go` exposes three tools over stdio JSON-RPC:
- `list-repos` — returns user's writable repositories
- `list-sloppers` — returns PRs from new contributors (accepts `repositories` and `min_contributions` args)
- `profile-sloppers` — batch-fetches GitHub user profiles for deep slop analysis (accepts `sloppers` list)

**Crush integration**: Running `gh slop` without a subcommand deploys an embedded `crush.json` config (which registers the MCP server) to `$XDG_CONFIG_HOME/gh-slop/crush/` and launches the `crush` CLI binary.

**Output styling**: `pkg/render/render.go` groups PRs by author and uses `lipgloss` for terminal styling. PRs are time-clustered (within 1-hour windows) and color-coded: white for the first in a cluster, yellow for the second, red for third+.

## Key Conventions

- **Module path**: `github.com/rsteube/gh-slop`
- **Go version**: 1.26.3 (specified in go.mod)
- **Cobra + carapace**: Commands use `spf13/cobra` for CLI structure and `carapace-sh/carapace` for shell completions. Every command calls `carapace.Gen(cmd)` in its `init()` to initialize carapace.
- **carapace-spec**: The root command registers with `spec.Register(rootCmd)` and exposes macros via `spec.AddMacro`. This enables YAML-based spec generation for carapace user specs.
- **gh API access**: Uses `go-gh/v2` (`api` package) for both REST and GraphQL. REST client from `api.DefaultRESTClient()`, GraphQL client from `api.NewGraphQLClient()`. The `--repo`/`-R` flag uses `repository.Parse()`/`repository.Current()` from go-gh. When a specific host is in the repo string, a GraphQL client is created for that host via `api.NewGraphQLClient(api.ClientOptions{Host: r.Host})`.
- **GraphQL pagination**: `fetchPullRequests` paginates with cursors (100 per page). New queries should follow this pattern.
- **Concurrent API calls**: Both `ListNewContributors` (multi-repo) and `filterNewContributors` (per-author counts) use a semaphore (`chan struct{}, 5`) to limit concurrency. Follow this pattern for any batched GitHub API calls.
- **Interface-based testing**: `graphqlDoer` interface in `pkg/slop/slop.go` abstracts the GraphQL client, enabling mock injection for tests. Note: `graphqlDoer` is unexported, so tests must reside in the `slop` package.
- **Flag naming**: Uses `pflag` conventions via cobra — `StringVarP`/`IntVarP` with short flags.
- **Color palette**: Use the [Charm color palette](https://github.com/charmbracelet/x/tree/main/colors) (`github.com/charmbracelet/x/colors`) for terminal colors. This package provides `lipgloss.AdaptiveColor` presets with light/dark variants (e.g., `colors.Indigo`, `colors.Green`, `colors.Fuschia`, `colors.Gray`). Prefer these named colors over hardcoded hex values. Note: `pkg/render/render.go` currently uses hardcoded `AdaptiveColor` values rather than the `colors` package — new code should prefer the `colors` package.
- **MCP server is hand-rolled**: The MCP server in `pkg/mcp/server.go` is a minimal JSON-RPC 2.0 implementation with Content-Length header framing, not using any MCP SDK. It only supports `initialize`, `tools/list`, and `tools/call` methods. If adding MCP capabilities, extend this server directly rather than introducing an SDK.
- **Embedded config**: `pkg/crush/crush.go` uses `//go:embed crush.json` to embed the Crush configuration. The `EnsureConfig()` function deploys it on first run but won't overwrite an existing config.

## Release

Releases are automated via `.github/workflows/release.yml` using `cli/gh-extension-precompile@v2`. Push a `v*` tag to trigger a release that cross-compiles for all platforms and generates provenance attestations.

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Carapace Skills

This project uses `carapace-sh/carapace` and `carapace-sh/carapace-spec` for shell completions. When working on this project, the following carapace-specific skills should be consulted:

- **carapace-action** — For creating, modifying, or structuring custom carapace actions (naming, Opts, caching, tags, UIDs, combining actions, ActionMultiParts, etc.)
- **carapace-integrate** — For integrating carapace into cobra-based CLIs: PreRun/PreInvoke hooks, flag/positional completions, action composition, carapace-bridge for external completions, carapace-spec registration, and carapace-pflag for non-POSIX flag modes
- **carapace-spec** — For creating or editing YAML completion spec files
- **carapace-macro** — For looking up available macros, formatting macro arguments in YAML/Go, and understanding MacroN/MacroI/MacroV types

Skill source code: https://github.com/carapace-sh/carapace-bin/tree/master/skills

### Carapace MCP (stdio)

The `carapace` binary provides a stdio MCP server (`carapace --mcp`) with a `list_macros` tool that lists all available macros (from carapace-bin and any registered specs). This enables loose coupling — instead of importing Go packages directly, you can reference macros by name (e.g., `carapace.tools.gh.Repositories`) via `spec.ActionMacro`. Use the `list_macros` MCP tool to discover available macros at design time.

Current carapace integration in this project:

- `cmd/root.go` registers two macros:
  - `Repos` via `spec.AddMacro("Repos", spec.MacroN(actions.ActionRepos))` — no arguments
  - `Sloppers` via `spec.AddMacro("Sloppers", spec.MacroV(actions.ActionSloppers))` — variadic arguments (repo list)
- `pkg/actions/repo.go` defines `ActionRepos()` as a public action with doc comment, 24-hour cache, and `"repositories"` tag — following the carapace-action skill conventions
- `pkg/actions/slopper.go` defines `ActionSloppers(repos ...string)` as a MacroV action with 15-minute cache (keyed on sorted repo list), styled values (negative/warning/usage based on slop count), and `"sloppers"` tag
- Every cobra command with flags calls `carapace.Gen(cmd)` in its `init()` function
- Flag completions are registered via `carapace.Gen(cmd).FlagCompletion(carapace.ActionMap{...})`
- The `--repo` flag completion uses `actions.ActionRepos().MultiParts("/").UniqueList(",")` for `owner/repo` format with comma separation

When adding new completion actions, follow the patterns established in `pkg/actions/repo.go` and consult the **carapace-action** skill for naming conventions, documentation format, caching, and macro registration.

For loosely coupled completions (e.g., reusing `carapace.tools.gh.*` macros from carapace-bin), use `spec.ActionMacro` to reference them by name rather than importing Go packages directly. Check available macros with the `list_macros` MCP tool.
