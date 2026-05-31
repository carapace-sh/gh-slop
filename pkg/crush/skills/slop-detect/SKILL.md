---
name: slop-detect
description: Use when the user wants to find, list, or analyze slop PRs — open pull requests from new or low-contribution authors. Triggers on mentions of "slop", "drive-by PRs", "low-effort contributions", "new contributor PRs", or reviewing unfamiliar authors' pull requests.
user-invocable: true
---

# Slop Detection

**When updating this skill, never include real usernames, account names, or other personally identifiable information in this file.** Use generic descriptions (e.g., "Author A", "a slop account") in all examples and pattern descriptions.

Use the `gh-slop` MCP server to identify open pull requests from contributors with few prior merged PRs ("sloppers"), then perform deep analysis to classify which PRs and authors are genuinely slop vs. potentially legitimate new contributors.

## Available MCP Tools

### `mcp_gh-slop_list-repos`

Lists the user's writable GitHub repositories. Use this first to discover which repos to scan.

No parameters required.

### `mcp_gh-slop_list-sloppers`

Lists open PRs from new/low-contribution authors. Accepts:

|| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `repositories` | `array<string>` | no | current repo | List of `owner/repo` repositories to scan |
| `min_contributions` | `integer` | no | `3` | Authors with fewer than this many merged PRs are considered sloppers |

## Workflow

### Step 1: Discover repos

Call `mcp_gh-slop_list-repos` to get the user's writable repositories. Group them by org/user for presentation.

### Step 2: Scan for slop

Call `mcp_gh-slop_list-sloppers` with the desired repositories and `min_contributions` threshold. When the user names an org or project, pass all repos matching that org prefix. When no repo is specified, use the current repository or ask the user which repos to target.

### Step 3: Deep analysis — Profile each author

For each unique author returned, use the GitHub GraphQL API to gather:

```
gh api graphql -f query='
query {
  user(login: "USERNAME") {
    createdAt
    contributionsCollection {
      totalCommitContributions
    }
    pullRequests(first: 50, orderBy: {field: CREATED_AT, direction: DESC}) {
      totalCount
      nodes {
        repository { nameWithOwner }
        title
        createdAt
        state
      }
    }
  }
}'
```

Extract the following signals for each author:

1. **Account age** — `createdAt` date. Very new accounts (< 6 months) are a strong slop signal.
2. **Total commits** — `totalCommitContributions`. Accounts with < 50 total commits and many PRs across many repos are likely slop.
3. **PR distribution** — How many different repos they target. Legitimate contributors focus on few repos; slop authors spray across many unrelated repos.
4. **PR success rate** — Ratio of MERGED to total PRs. Slop authors have very low merge rates (mostly OPEN or CLOSED).
5. **Burst pattern** — How many PRs filed per day. Filing 5+ PRs across different repos in a single day is a strong slop signal.
6. **Repo overlap with other slop authors** — Whether they target the same repos as other flagged authors, especially within the same time window.

### Step 4: Deep analysis — Classify each PR

For each PR, check for these slop signals:

#### Duplicate / overlapping PRs

Use `gh pr view PR_NUMBER --repo OWNER/REPO --json title,body,author,createdAt` to compare PRs that:
- Target the same issue (check for "Closes #", "Fixes #", "Refs #" in the body)
- Implement the same feature (e.g., multiple "add zig completer" PRs)
- Were filed within days of each other by different authors

When multiple PRs target the same issue, only the first (or best) is legitimate; the rest are slop.

#### AI-generated PRs

Check for these AI slop markers in PR bodies:
- "Floyd Autonomous Fix" or "Floyd: Fix GitHub issue" — indicates automated AI agent
- "[codex]" prefix in title — indicates Codex AI agent
- "/claim" in body — bounty-hunting behavior
- Body contains the full issue text copied verbatim as "context"
- Body has a rigid template structure: "Task:", "Request:", "Proposed solution:" (AI agent format)
- Body mentions AI tools by name (e.g., "codex", "Floyd", "OpenClaw", "Remote OpenClaw")

#### Bounty-driven PRs

PRs targeting repos with bounty labels ("fund", "bounty", "bounty-hunters", "reward") or repos in bounty platforms (UnsafeLabs, Scottcjn/Rustchain, gitcoinco, etc.) are often slop. Check:
- Does the repo have a "fund" or "bounty" label on the targeted issue?
- Is the PR body structured as a bounty claim ("/claim #ISSUE")?
- Does the repo name contain "bounty", "reward", "hackathon", or similar?

#### Self-promotion / spam PRs

PRs that add the author's own project/tool to "awesome-*" lists or similar directories:
- Multiple identical PRs to different "awesome-*" repos with the same content
- PR adds a link to the author's own service/tool
- No code changes, only markdown/list additions

### Step 5: Identify slop patterns

Group findings by pattern type. Common patterns observed in the wild:

| Pattern | Description | Example |
|---------|-------------|---------|
| **Issue racing** | Multiple authors submit PRs for the same bounty-labeled issue within days | 3 PRs for the same completer in one week |
| **Burst spraying** | Author files 10+ PRs across many repos in a single day | 42 PRs filed on a single day |
| **Coordinated accounts** | Two accounts file near-identical PRs to the same repos | Two accounts submit the same feature within 24h |
| **AI agent slop** | PR body contains AI agent markers ("Floyd:", "[codex]") | PR titled "Floyd Autonomous Fix" |
| **Self-promo spam** | Same link submission across many "awesome-*" repos | Same tool link submitted to 10+ awesome lists |
| **Template completers** | Identically structured PRs following the same pattern, with no domain-specific adaptation | Mass completer submissions following a cookie-cutter template |
| **Feature duplication** | Multiple authors implement the same feature independently | 4 PRs for the same feature in one week |
| **Bounty farming** | Author targets repos with bounty/fund labels across platforms | Bounty repos, /claim markers, multiple bounty platforms |
| **Codex toolchain** | PR validation references Codex-specific paths | `/tmp/codex-go-toolchain/` in validation steps |
| **Stale title prefix** | PR title contains "PR Title:" prefix | Floyd agent artifact, title not cleaned up |

### Step 6: Classify authors

Based on the analysis, classify each author into one of these categories:

**Slop Author** — Account exists primarily to generate low-effort or AI-generated PRs, often targeting bounties:
- Account age < 6 months AND burst spraying pattern
- Majority of PRs to bounty/fund-labeled issues
- AI agent markers in PR bodies or titles
- Low merge rate (< 20% of PRs merged)
- PRs across many unrelated repos with no domain expertise signal
- Coordinated with other slop accounts (same repos, same time window)

**Likely Slop Author** — Shows some slop signals but may have some legitimate contributions:
- New account with burst pattern but some PRs show domain knowledge
- Mix of bounty-targeted and genuine contributions
- Some PRs are well-structured while others are AI-generated

**Not Slop (False Positive)** — New contributor with legitimate contributions:
- Account may be new but PRs are focused on repos they actually use
- PRs show understanding of the codebase and domain
- No AI agent markers, no bounty claims
- PR success rate is reasonable
- Contributions are to few, related repos

**AI-Assisted but Legitimate** — Uses AI tools (like Codex) but contributes meaningfully:
- Has substantial commit history and merge rate
- PRs to repos they maintain or actively contribute to
- AI tool used as assistive, not autonomous
- Marked with `[codex]` or similar but PRs are well-scoped

### Step 7: Present results

Summarize findings as:

1. **Summary statistics** — Total slop PRs, total slop authors, repos affected
2. **Slop author table** — Author, category (slop/likely slop), key signals, PR count, repos affected
3. **Pattern breakdown** — Group PRs by pattern type with specific examples
4. **Duplicate PR map** — Table of issues/features with multiple competing PRs, showing which PR was first and which are duplicates
5. **Recommendations** — Which PRs to close, which authors to watch, suggested `min_contributions` threshold adjustment

## Slop Signals Reference

### Author-level signals (weighted by importance)

| Signal | Weight | How to detect |
|--------|--------|---------------|
| Account age < 6 months | High | Check `createdAt` from GraphQL |
| Burst pattern (5+ PRs/day) | High | Group PRs by date, count per day |
| Low merge rate (< 20%) | High | Count MERGED vs total from PR states |
| PRs across 5+ unrelated repos | Medium | Count unique repos from PR list |
| Bounty repo targeting | Medium | Check if repos have "fund"/"bounty" labels |
| AI agent markers in PRs | High | Search body for "Floyd", "[codex]", "/claim" |
| Coordinated with other flagged authors | High | Check repo overlap + timing overlap |
| Self-promotion pattern | Medium | Same content across many "awesome-*" repos |
| Very low total commits (< 50) | Medium | `totalCommitContributions` from GraphQL |

### PR-level signals

| Signal | Weight | How to detect |
|--------|--------|---------------|
| Duplicate of earlier PR for same issue | High | Compare issue references and titles |
| AI agent markers in body/title | High | Search for "Floyd:", "[codex]", "Autonomous Fix" |
| Bounty claim ("/claim") | Medium | Search body for "/claim" |
| Very small diff (1-5 lines) with templated body | Low | Check `additions`/`deletions` count |
| Targets "fund"/"bounty"-labeled issue | Medium | Check issue labels on the target repo |
| Self-promotional content | Medium | Adds link to author's own project in list |
| Identical structure to other PRs by same author | Medium | Compare titles/body format across author's PRs |

## Organization-Specific Patterns

When deep analysis reveals slop patterns that are unique to a specific project, organization, or tool — not generalizable across all repos — write a separate skill with the prefix `slop-patterns-*` (replacing `*` with the organization, repo, or tool name). For example:

- `slop-patterns-carapace` for carapace-sh specific patterns (completer racing, template completer spraying)
- `slop-patterns-awesome` for awesome-list specific patterns (self-promo spam)
- `slop-patterns-rust` for Rust ecosystem specific patterns

These project-specific skills should contain:
1. Project-specific slop signals and detection methods
2. Known offender classifications with evidence
3. Project-specific analysis steps to add to the general workflow
4. Concrete examples of observed patterns

When the `slop-detect` skill is triggered for a project that has a `slop-patterns-*` skill, load and follow that skill as well.

## Tips

- The `repositories` parameter is optional; omitting it scans the current repository only.
- Pass multiple repos to scan across an organization.
- Some repos returned by `list-repos` may be archived or renamed — errors are reported per-repo and can be safely retried without the failing repo.
- `dependabot` and similar bots may appear as sloppers; note them separately when summarizing.
- Not every PR from a new contributor is slop. Always perform deep analysis before labeling.
- When PRs are duplicates, note which was filed first — the earliest PR has the strongest legitimacy claim.
- Coordinated accounts are a strong slop signal: if two authors file near-identical PRs to the same repos within days, they are likely related.
- Repos with "fund" or "bounty" labels attract disproportionate slop. Consider raising `min_contributions` when scanning such repos.
- AI agent tools (Floyd, Codex, etc.) leave distinctive markers in PR bodies — always check for these.
