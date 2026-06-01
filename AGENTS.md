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
gh slop close <slopper>    # close all open PRs from a given slopper (interactive confirmation)
gh slop close <slopper> OWNER/REPO#123 OWNER/REPO#456  # close only specific PRs from that slopper
gh slop detect             # detect slop using Crush AI analysis (auto-targets current repo)
gh slop detect -R o/r      # detect slop in specific repo(s)
gh slop mcp                # starts the MCP stdio server (hidden command)

# Test
go test ./...
```

There is no linter config or Makefile in the project. There are no tests currently (no `*_test.go` files exist).

## Architecture

```
main.go                     → entry point, calls cmd.Execute()
cmd/root.go                 → root cobra command, --repo flag (comma-separated multi-repo), ResolveRepos(), carapace/spec setup
cmd/list.go                 → `list` subcommand, renders PRs via pkg/render
cmd/close.go                → `close` subcommand, interactively confirms then closes PRs from a slopper (accepts optional PR refs to selectively close)
cmd/detect.go               → `detect` subcommand, launches Crush AI with slop-detect skill
cmd/mcp.go                  → `mcp` subcommand (hidden), starts MCP stdio server
pkg/slop/slop.go            → core logic: fetch PRs, fetch contribution counts, filter new contributors
pkg/slop/closeprs.go        → batch PR closing via REST API
pkg/slop/prdetails.go       → batch PR detail fetching (aliased GraphQL queries per repo), ParsePRRef()
pkg/slop/profile.go         → batch user profile fetching (account age, commits, PR stats) via GraphQL
pkg/slop/repos.go           → ResolveRepos() (flag→Repository parsing), AccessibleRepos() (REST API)
pkg/slop/parallel.go        → generic parallelMap[T,R] helper (replaces manual semaphore patterns)
pkg/slop/api/client.go      → GraphQL/REST client singletons (unexported, sync.Once), graphQLDoer/restDoer interfaces
pkg/slop/api/graphql.go     → GraphQL query constants and response type structs
pkg/slop/api/graphql_calls.go → GraphQL call implementations (pagination, profile, PR details, merged count)
pkg/slop/api/rest_calls.go  → REST call implementations (accessible repos, close PR)
pkg/render/render.go        → terminal output: groups by author, time-cluster coloring, lipgloss styling
pkg/mcp/server.go           → MCP server types (Server, Tool, Request/Response, NewServer)
pkg/mcp/handler.go           → MCP request routing (ServeStdio, processMessage, handleRequest, handleToolCall)
pkg/mcp/handlers.go          → MCP tool handler implementations (listReposHandler, listSloppersHandler, etc.)
pkg/mcp/formatter.go         → MCP response formatting (formatPRs, formatProfiles, formatPRDetails, htmlEscape)
pkg/actions/repo.go         → carapace completion action for repositories (ActionRepos)
pkg/actions/slopper.go      → carapace completion actions: ActionSloppers (MacroV), ActionSlopperPRs (MacroI)
pkg/crush/crush.go          → Crush integration: deploys embedded crush.json + slop-detect skill, launches `crush` CLI
pkg/crush/crush.json        → embedded config: registers gh-slop as MCP server for Crush
pkg/crush/skills/slop-detect/SKILL.md → embedded skill: 7-step slop detection workflow for AI agents
```

**Data flow**: `cmd/list.go` calls `slop.ListNewContributors(repos, minContributions)` which:
1. Fetches all open PRs via paginated GraphQL (`api.FetchOpenPullRequests`) — concurrently across multiple repos via `parallelMap`
2. For each unique author, concurrently fetches their merged PR count via GraphQL search (`filterNewContributors` → `api.FetchMergedPRCount`, concurrency 5)
3. Filters PRs where the author's merged count is below the threshold
4. Results are rendered by `render.Render()` which groups by author, sorts chronologically, and applies time-cluster coloring

**Multi-repo flow**: The `--repo`/`-R` flag accepts comma-separated repos (`StringSliceVarP`). `ListNewContributors` processes all repos concurrently via `parallelMap` (concurrency 5). When multiple repos are targeted, each PR is prefixed with `owner/repo#` in output.

**Close flow**: `cmd/close.go` calls `slop.FindPRsByAuthor` to find all open PRs by a username. Optional positional args after the slopper name are PR refs (`OWNER/REPO#NUMBER`) to selectively close specific PRs only. Shows PRs, prompts for `y/N` confirmation, then calls `slop.ClosePRs` which parses refs and closes each via REST API.

**Detect flow**: `cmd/detect.go` calls `crush.RunDetect` which resolves repos (defaults to current repo), builds a prompt like `"detect slop in owner/repo"`, and runs `crush run <prompt>` with the embedded MCP config and slop-detect skill.

**MCP server**: `cmd/mcp.go` exposes five tools over stdio JSON-RPC:
- `list-repos` — returns user's writable repositories
- `list-sloppers` — returns PRs from new contributors (accepts `repositories` and `min_contributions` args)
- `profile-sloppers` — batch-fetches GitHub user profiles for deep slop analysis (accepts `sloppers` list)
- `view-prs` — batch-fetches PR details (title, body, author, createdAt, URL) for a list of PRs in `OWNER/REPO#NUMBER` format, using aliased GraphQL queries per repo
- `close-prs` — closes pull requests by reference, accepts a list of PRs in `OWNER/REPO#NUMBER` format and closes each via the GitHub REST API (destructive — must only be invoked with explicit user authorization)

**Crush integration**: Running `gh slop` without a subcommand deploys an embedded `crush.json` config (which registers the MCP server) and the `slop-detect` skill to `$XDG_CONFIG_HOME/gh-slop/crush/`, then launches the `crush` CLI binary. Running `gh slop detect` does the same but passes `crush run "detect slop in ..."` to auto-trigger the skill. The `EnsureConfig()` function won't overwrite an existing `crush.json` (only deploys if missing), but the skill file is always overwritten.

**Output styling**: `pkg/render/render.go` groups PRs by author and uses `lipgloss` for terminal styling. PRs are time-clustered (within 1-hour windows) and color-coded: white for the first in a cluster, yellow for the second, red for third+.

## Key Conventions

- **Module path**: `github.com/rsteube/gh-slop`
- **Go version**: 1.26.3 (specified in go.mod)
- **Cobra + carapace**: Commands use `spf13/cobra` for CLI structure and `carapace-sh/carapace` for shell completions. Every command calls `carapace.Gen(cmd)` in its `init()` to initialize carapace.
- **carapace-spec**: The root command registers with `spec.Register(rootCmd)` and exposes macros via `spec.AddMacro`. This enables YAML-based spec generation for carapace user specs.
- **gh API access**: Uses `go-gh/v2` (`api` package) for both REST and GraphQL. Client singletons are in `pkg/slop/api/client.go` — `graphQLClient()` and `restClient()` (unexported, `sync.Once`-initialized). The `--repo`/`-R` flag uses `repository.Parse()`/`repository.Current()` from go-gh. When a specific host is in the repo string, a GraphQL client is created for that host via `api.NewGraphQLClient(api.ClientOptions{Host: r.Host})`.
- **GraphQL pagination**: `FetchOpenPullRequests` and `FetchPullRequestsByAuthor` paginate with cursors (100 per page). New queries should follow this pattern.
- **Concurrent API calls**: Use `parallelMap` from `pkg/slop/parallel.go` for all concurrent batch operations. It accepts a concurrency limit (typically 5), preserves order via index tracking, and fails fast on the first error. Do not create manual semaphore patterns — `parallelMap` replaces them.
- **API layer separation**: `pkg/slop/api/` contains all raw API interactions (GraphQL queries, REST calls, client management). The `pkg/slop/` package contains domain logic and types, calling into `api` for data. New API calls go in `pkg/slop/api/`; new domain logic goes in `pkg/slop/`.
- **Interface-based testing**: `graphQLDoer` and `restDoer` interfaces in `pkg/slop/api/client.go` abstract the respective clients, enabling mock injection for tests. Both are unexported, so tests must reside in the `api` package.
- **Flag naming**: Uses `pflag` conventions via cobra — `StringVarP`/`IntVarP` with short flags.
- **Color palette**: Use the [Charm color palette](https://github.com/charmbracelet/x/tree/main/colors) (`github.com/charmbracelet/x/colors`) for terminal colors. This package provides `lipgloss.AdaptiveColor` presets with light/dark variants (e.g., `colors.Indigo`, `colors.Green`, `colors.Fuschia`, `colors.Gray`). Prefer these named colors over hardcoded hex values. Note: `pkg/render/render.go` currently uses hardcoded `AdaptiveColor` values rather than the `colors` package — new code should prefer the `colors` package.
- **MCP server is hand-rolled**: The MCP server is a minimal JSON-RPC 2.0 implementation with newline-delimited JSON framing (not Content-Length), not using any MCP SDK. It only supports `initialize`, `notifications/initialized`, `tools/list`, and `tools/call` methods. If adding MCP capabilities, extend the existing files directly rather than introducing an SDK.
- **MCP server file split**: The MCP server is split across four files — `server.go` (types, Tool definitions, NewServer), `handler.go` (ServeStdio, processMessage, handleRequest, handleToolCall), `handlers.go` (per-tool handler functions), `formatter.go` (response formatting + htmlEscape). When adding a new MCP tool: (1) add Tool definition in `server.go`'s `NewServer()`, (2) add a case in `handleToolCall` in `handler.go`, (3) add a handler function in `handlers.go`, (4) add a formatter in `formatter.go` if needed.
- **Embedded config**: `pkg/crush/crush.go` uses `//go:embed` to embed both `crush.json` and `skills/slop-detect/SKILL.md`. The `EnsureConfig()` function deploys them on first run but won't overwrite an existing `crush.json` (skill is always overwritten).

## Gotchas

- **Merged PR count uses search, not direct query**: `FetchMergedPRCount` uses GitHub's search API (`search(query: ..., type: ISSUE, first: 1) { issueCount }`) to count merged PRs. The search query is `repo:OWNER/NAME is:pr is:closed author:LOGIN`. This counts *closed* PRs (which includes merged), not just merged — the naming is slightly misleading.
- **GraphQL client is a singleton**: `graphQLClient()` and `restClient()` are `sync.Once` singletons. If you need a client for a different host, create a new one directly with `ghapi.NewGraphQLClient(ghapi.ClientOptions{Host: ...})` rather than trying to modify the singleton.
- **Author can be empty string**: GraphQL responses may have `author: null` (e.g., deleted accounts), mapped to empty string. Code filters these out in `filterNewContributors` (`if pr.Author != ""`), but downstream code should be aware.
- **Sort order depends on string comparison**: `sortByCreatedAt` sorts by `CreatedAt` string comparison, which works because ISO 8601 strings sort lexicographically in chronological order.
- **MCP protocol version is hardcoded**: The MCP server declares protocol version `2025-11-25` and server version `0.0.0` in `handler.go`. Update these when bumping the MCP protocol version.
- **`listReposHandler` ignores its params argument**: The `list-repos` MCP tool accepts no parameters, but the handler still receives `params json.RawMessage`. The `unusedparams` lint warning is intentional — the handler signature must match the pattern used by other handlers.
- **`close` command re-declares `--repo` flag**: `cmd/close.go` has its own `StringSliceVarP` for `--repo` instead of using the root command's persistent flag. This means the flag works on the `close` subcommand directly, but the `repos` package-level variable in `cmd/root.go` is shadowed by a local one in `close.go`.
- **`ActionSloppers` hardcodes `minContributions=3`**: The slopper completion action uses a hardcoded threshold of 3 (with a TODO comment for config), while the `list` command defaults to 1. This means the completion may show different authors than the list output.
- **Carapace `--repo` flag on `close` has its own completion**: The `close` command registers its own `--repo` flag completion independently (not inherited from root's persistent flag), so any changes to the root's flag completion must be duplicated there.
- **PR reference format**: PRs are referenced throughout as `OWNER/REPO#NUMBER` (e.g., `cli/cli#1234`). `ParsePRRef()` in `pkg/slop/prdetails.go` parses this format. This format is used by MCP tools (`view-prs`, `close-prs`) and the `close` command.

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

- `cmd/root.go` registers three macros:
  - `Repos` via `spec.AddMacro("Repos", spec.MacroN(actions.ActionRepos))` — no arguments
  - `Sloppers` via `spec.AddMacro("Sloppers", spec.MacroV(actions.ActionSloppers))` — variadic arguments (repo list)
  - `SlopperPRs` via `spec.AddMacro("SlopperPRs", spec.MacroI(actions.ActionSlopperPRs))` — single argument (SlopperPROpts struct)
- `pkg/actions/repo.go` defines `ActionRepos()` as a public action with doc comment, 24-hour cache, and `"repositories"` tag — following the carapace-action skill conventions
- `pkg/actions/slopper.go` defines `ActionSloppers(repos ...string)` as a MacroV action with 15-minute cache (keyed on sorted repo list), styled values (negative/warning/usage based on slop count), and `"sloppers"` tag
- Every cobra command with flags calls `carapace.Gen(cmd)` in its `init()` function
- Flag completions are registered via `carapace.Gen(cmd).FlagCompletion(carapace.ActionMap{...})`
- The `--repo` flag completion uses `actions.ActionRepos().MultiParts("/").UniqueList(",")` for `owner/repo` format with comma separation
- The `close` command uses `actions.ActionSloppers(repos...)` for the first positional arg (slopper username) and `actions.ActionSlopperPRs` for subsequent args (PR refs)

When adding new completion actions, follow the patterns established in `pkg/actions/repo.go` and consult the **carapace-action** skill for naming conventions, documentation format, caching, and macro registration.

For loosely coupled completions (e.g., reusing `carapace.tools.gh.*` macros from carapace-bin), use `spec.ActionMacro` to reference them by name rather than importing Go packages directly. Check available macros with the `list_macros` MCP tool.
