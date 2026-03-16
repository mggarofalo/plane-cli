package cmdgen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// stdinReader is the source for stdin data. Overridable in tests.
var stdinReader io.Reader = os.Stdin

// ReadStdinJSON reads JSON from stdin and returns it as a map.
// Returns an error if the input is not valid JSON or is not an object.
func ReadStdinJSON() (map[string]any, error) {
	data, err := io.ReadAll(stdinReader)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("--stdin specified but stdin is empty")
	}

	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("invalid JSON on stdin: %w", err)
	}

	return body, nil
}

// MergeStdinWithFlags merges stdin JSON with flag-collected body params.
// Flag values take precedence over stdin values (explicit flags override).
func MergeStdinWithFlags(stdinBody, flagBody map[string]any) map[string]any {
	if stdinBody == nil && flagBody == nil {
		return nil
	}

	merged := make(map[string]any)

	// Start with stdin values
	for k, v := range stdinBody {
		merged[k] = v
	}

	// Override with flag values (flags take precedence)
	for k, v := range flagBody {
		merged[k] = v
	}

	if len(merged) == 0 {
		return nil
	}

	return merged
}

// isStdin returns true when the --stdin flag is set. Nil-safe.
func isStdin(deps *Deps) bool {
	return deps != nil && deps.FlagStdin != nil && *deps.FlagStdin
}

// ResolveStdinBody applies name-to-UUID resolution on body fields that came
// from stdin. Fields already resolved by flag collection are skipped (the
// caller passes only stdin-originated keys via the stdinKeys set). This
// ensures that human-readable names in stdin JSON (e.g., "state": "In Progress")
// are resolved the same way as flag values.
func ResolveStdinBody(ctx context.Context, body map[string]any, stdinKeys map[string]bool, deps *Deps) error {
	if body == nil || deps == nil {
		return nil
	}
	for key := range stdinKeys {
		val, ok := body[key]
		if !ok {
			continue
		}
		strVal, isStr := val.(string)
		if !isStr || strVal == "" {
			continue
		}
		resolved, err := resolveIfNeeded(ctx, strVal, key, nil, "", deps)
		if err != nil {
			return err
		}
		body[key] = resolved
	}
	return nil
}

// StdinKeys returns the set of keys from a stdin body that are not overridden
// by flag-provided values. These are the keys that need name resolution.
func StdinKeys(stdinBody, flagBody map[string]any) map[string]bool {
	keys := make(map[string]bool)
	for k := range stdinBody {
		if _, overridden := flagBody[k]; !overridden {
			keys[k] = true
		}
	}
	return keys
}
