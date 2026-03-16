package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(resolutionCmd)
}

// ResolutionHelpText is the full reference for the name resolution system.
// Exported so tests can verify it stays in sync.
const ResolutionHelpText = `Name Resolution
===============

The CLI automatically resolves human-readable names to UUIDs so you
don't need to look up IDs manually.

Supported resource types
------------------------
  Resource    Matched fields              Example
  --------    ---------------             -------
  state       name                        --state "In Progress"
  label       name                        --label-id "Bug"
  cycle       name                        --cycle "Sprint 12"
  module      name                        --module "Backend v2"
  member      display_name                --assignee-id "Jane Doe"
  project     identifier                  -p PLANECLI

Sequence IDs
------------
Flags that accept work-item references (--work-item-id, --parent, --issues)
also accept Plane sequence IDs in IDENTIFIER-N format:

  plane issue get -p PROJ --work-item-id PROJ-42

The CLI resolves PROJ-42 to the full UUID via the work-items API.

How resolution works
--------------------
1. The CLI checks if the value is already a UUID (36-char dash format).
   If so, it is used as-is.
2. For sequence IDs (e.g. PROJ-42), the CLI calls the work-items lookup
   endpoint to resolve to a UUID.
3. For name-resolvable params, the CLI checks a local disk cache
   (keyed by workspace + project). On a cache hit, the UUID is returned
   immediately. On a miss, the CLI fetches the full resource list from
   the API and caches the result.

Cache behavior
--------------
  Soft TTL:  1 hour  — after this, a background refresh is triggered
                       but the cached value is still returned.
  Hard TTL:  7 days  — after this, a synchronous refresh is required
                       before the value can be used.
  Location:  ~/.cache/plane-cli/resources/<profile>/

Use 'plane cache clear' to purge the resolution cache.

Control flags
-------------
  --no-resolve    Skip all name resolution; pass values as literal strings.
                  Useful when you already have UUIDs or want to bypass the
                  cache entirely.

  --strict        Treat name-resolution failures as hard errors (exit code 4).
                  Without this flag, a failed resolution emits a warning to
                  stderr and passes the literal value through to the API,
                  which typically returns a 400/422 error.

Tips for AI agents
------------------
- Prefer human-readable names over UUIDs for readability and
  maintainability. The CLI resolves them automatically.
- Use sequence IDs (PROJ-42) for issue cross-references.
- Combine --strict with name-based flags to fail fast on typos.
- Run 'plane state list -p <PROJECT> -o json' to see available state
  names for a project.
- The --field and --id-only flags work well with resolution: create a
  resource by name and capture its UUID in one step.
`

var resolutionCmd = &cobra.Command{
	Use:   "resolution",
	Short: "Explain the name resolution system",
	Long:  "Show detailed documentation about the CLI's automatic name-to-UUID resolution.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(ResolutionHelpText)
	},
}
