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
gh slop                    # default: verifies auth by fetching current user
gh slop list               # lists PRs from new/low-contribution authors
gh slop list -m 3          # only show authors with fewer than 3 merged PRs
gh slop -R owner/repo list # target a specific repository

# Test
go test ./...
```

There is no linter config or Makefile in the project. There are no tests currently.

## Architecture

```
main.go              → entry point, calls cmd.Execute()
cmd/root.go          → root cobra command, --repo flag, resolveRepo(), carapace/spec setup
cmd/list.go          → `list` subcommand, PR display with time-cluster highlighting
pkg/slop/slop.go     → core logic: fetch PRs, fetch contribution counts, filter new contributors
pkg/actions/repo.go  → carapace completion action for repositories
```

**Data flow**: `cmd/list.go` calls `slop.ListNewContributors(repo, minContributions)` which:
1. Fetches all open PRs via paginated GraphQL (`fetchPullRequests`)
2. For each unique author, concurrently fetches their merged PR count via GraphQL search (`fetchContributionCounts`, semaphore-limited to 5 concurrent requests)
3. Filters PRs where the author's merged count is below the threshold

**Output styling**: `cmd/list.go` groups PRs by author and uses `lipgloss` for terminal styling. PRs are time-clustered (within 1-hour windows) and color-coded: white for the first in a cluster, yellow for the second, red for third+.

## Key Conventions

- **Module path**: `github.com/rsteube/gh-slop`
- **Go version**: 1.26.3 (specified in go.mod)
- **Cobra + carapace**: Commands use `spf13/cobra` for CLI structure and `carapace-sh/carapace` for shell completions. Every command with flags calls `carapace.Gen(cmd)` in its `init()`.
- **carapace-spec**: The root command registers with `spec.Register(rootCmd)` and exposes a `Repos` macro via `spec.AddMacro`. This enables YAML-based spec generation for carapace user specs.
- **gh API access**: Uses `go-gh/v2` (`api` package) for both REST and GraphQL. REST client from `api.DefaultRESTClient()`, GraphQL client from `api.NewGraphQLClient()`. The `--repo`/`-R` flag uses `repository.Parse()`/`repository.Current()` from go-gh.
- **GraphQL pagination**: `fetchPullRequests` paginates with cursors (100 per page). New queries should follow this pattern.
- **Concurrent API calls**: `fetchContributionCounts` uses a semaphore (`chan struct{}, 5`) to limit concurrency. Follow this pattern for any batched GitHub API calls.
- **Interface-based testing**: `graphqlDoer` interface in `pkg/slop/slop.go` abstracts the GraphQL client, enabling mock injection for tests.
- **Flag naming**: Uses `pflag` conventions via cobra — `StringVarP`/`IntVarP` with short flags.

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

- `cmd/root.go` registers the `Repos` macro via `spec.AddMacro("Repos", spec.MacroN(actions.ActionRepos))` and calls `spec.Register(rootCmd)`
- `pkg/actions/repo.go` defines `ActionRepos()` as a public action with doc comment, 24-hour cache, and `"repositories"` tag — following the carapace-action skill conventions
- Every cobra command with flags calls `carapace.Gen(cmd)` in its `init()` function
- Flag completions are registered via `carapace.Gen(cmd).FlagCompletion(carapace.ActionMap{...})`

When adding new completion actions, follow the patterns established in `pkg/actions/repo.go` and consult the **carapace-action** skill for naming conventions, documentation format, caching, and macro registration.

For loosely coupled completions (e.g., reusing `carapace.tools.gh.*` macros from carapace-bin), use `spec.ActionMacro` to reference them by name rather than importing Go packages directly. Check available macros with the `list_macros` MCP tool.

## Gotchas

- The `graphqlDoer` interface in `pkg/slop/slop.go` is unexported. To test `ListNewContributors` from external packages, you'd need to either export it or test from within the `slop` package.
- The root command (no subcommand) hits the REST API to verify auth and print the current user. The `list` subcommand is the actual feature — this is intentional for a `gh` extension.
- `go-gh` relies on the `gh` CLI's auth config. Running locally requires being authenticated via `gh auth login`.
- The `--repo` flag is persistent (`PersistentFlags`), so it applies to all subcommands.
- `ActionRepos` in `pkg/actions/repo.go` caches results for 24 hours. It filters to repos where the user has push or admin permissions.