package api

const QueryOpenPullRequests = `
	query($owner: String!, $name: String!, $cursor: String) {
		repository(owner: $owner, name: $name) {
			pullRequests(first: 100, after: $cursor, states: [OPEN]) {
				pageInfo { hasNextPage endCursor }
				edges { node { number title createdAt author { login } } }
			}
		}
	}`

const QuerySearchPullRequests = `
	query($query: String!, $cursor: String) {
		search(query: $query, type: ISSUE, first: 100, after: $cursor) {
			pageInfo { hasNextPage endCursor }
			edges { node { ... on PullRequest { number title createdAt author { login } repository { nameWithOwner } } } }
		}
	}`

const QuerySearchIssues = `
	query($query: String!, $cursor: String) {
		search(query: $query, type: ISSUE, first: 100, after: $cursor) {
			pageInfo { hasNextPage endCursor }
			edges { node { ... on Issue { number title createdAt state author { login } repository { nameWithOwner } } } }
		}
	}`

const QueryMergedPRCount = `
	query($query: String!) {
		search(query: $query, type: ISSUE, first: 1) {
			issueCount
		}
	}`

const QueryUserProfile = `
	query($login: String!) {
		user(login: $login) {
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
	}`

type PullRequestsResponse struct {
	Repository struct {
		PullRequests struct {
			PageInfo struct {
				HasNextPage bool    `json:"hasNextPage"`
				EndCursor   *string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node PullRequestNode `json:"node"`
			} `json:"edges"`
		} `json:"pullRequests"`
	} `json:"repository"`
}

type MergedPRCountResponse struct {
	Search struct {
		IssueCount int `json:"issueCount"`
	} `json:"search"`
}

type UserProfileResponse struct {
	User struct {
		CreatedAt string `json:"createdAt"`
		ContributionsCollection struct {
			TotalCommitContributions int `json:"totalCommitContributions"`
		} `json:"contributionsCollection"`
		PullRequests struct {
			TotalCount int `json:"totalCount"`
			Nodes      []struct {
				Repository struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"repository"`
				Title     string `json:"title"`
				CreatedAt string `json:"createdAt"`
				State     string `json:"state"`
			} `json:"nodes"`
		} `json:"pullRequests"`
	} `json:"user"`
}

type SearchPullRequestsResponse struct {
	Search struct {
		PageInfo struct {
			HasNextPage bool    `json:"hasNextPage"`
			EndCursor   *string `json:"endCursor"`
		} `json:"pageInfo"`
		Edges []struct {
			Node PullRequestNode `json:"node"`
		} `json:"edges"`
	} `json:"search"`
}

type IssueNode struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	State     string `json:"state"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
}

type SearchIssuesResponse struct {
	Search struct {
		PageInfo struct {
			HasNextPage bool    `json:"hasNextPage"`
			EndCursor   *string `json:"endCursor"`
		} `json:"pageInfo"`
		Edges []struct {
			Node IssueNode `json:"node"`
		} `json:"edges"`
	} `json:"search"`
}

type UserRepoResponse struct {
	FullName    string `json:"full_name"`
	Permissions struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
	} `json:"permissions"`
}
