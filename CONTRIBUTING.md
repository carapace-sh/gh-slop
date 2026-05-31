# Contributing

This is a clanker generated project.
Install requirements and make prompts.

## Requirements

- [Carapace](https://github.com/carapace-sh/carapace-bin)
- [Crush](https://github.com/charmbracelet/crush)
- [Slopware](https://carapace-sh.github.io/carapace-bin/slopware.html) (MCP, skills)

## Examples
-
  > modify the repo flag to accept multiple comma separated repositories (complete as unique list)\
  > if there is more than one prefix the pr number with owner/repository (e.g. carapace-sh/carapace-bin#123)\
  > also, move the list rendering logic to a separate package in pkg to keep the command concise 
  
  [![asciicast](https://asciinema.org/a/1167988.svg)](https://asciinema.org/a/1167988)

-
  > create a public actions for ListNewContributors called ActionSloppers and expose it as macro\
  > it should take a list of repositories as argument and default to the current one\
  > value should be the username\
  > values should be styled according to the list output (hex colors) and the amount of assumed slop PRs (0=dim,1=yellow,2=red)\
  > description contain the amount of assumed slop prs vs the amount of his open prs and the contributors name\
  > cache this for 15 minutes per repository
  > 
  > slopper ([1/2] full name)\
  > another ([4/8] full name)
  
  [![asciicast](https://asciinema.org/a/1168218.svg)](https://asciinema.org/a/1168218)
