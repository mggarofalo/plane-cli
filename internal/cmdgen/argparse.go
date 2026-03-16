package cmdgen

import (
	"fmt"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

// ParsedArgs holds the result of manually parsing --flag=value and --flag value pairs.
type ParsedArgs struct {
	Values map[string]string
	Slices map[string][]string
}

// Get returns the value for a flag, or empty string.
func (p *ParsedArgs) Get(name string) string {
	return p.Values[name]
}

// GetSlice returns the slice value for a flag.
func (p *ParsedArgs) GetSlice(name string) []string {
	return p.Slices[name]
}

// Has returns true if a flag was provided.
func (p *ParsedArgs) Has(name string) bool {
	_, ok := p.Values[name]
	if ok {
		return true
	}
	_, ok = p.Slices[name]
	return ok
}

// ParseRawArgs manually parses --flag=value and --flag value pairs from raw args.
// This is used for Mode B (lazy) commands where DisableFlagParsing is true.
func ParseRawArgs(args []string, params []docs.ParamSpec) (*ParsedArgs, error) {
	result := &ParsedArgs{
		Values: make(map[string]string),
		Slices: make(map[string][]string),
	}

	// Build a lookup of known flag names → param spec
	flagMap := make(map[string]*docs.ParamSpec)
	for i := range params {
		flagName := ParamToFlagName(params[i].Name)
		flagMap[flagName] = &params[i]
		// Register markdown alias for _html params (e.g., --description → description_html spec)
		if IsHTMLParam(params[i].Name) {
			mdFlag := MarkdownFlagName(params[i].Name)
			flagMap[mdFlag] = &params[i]
		}
	}

	// Also recognize global flags
	globalFlags := map[string]bool{
		"workspace": true, "w": true,
		"project": true, "p": true,
		"output": true, "o": true,
		"api-url": true, "api-key": true,
		"verbose": true, "per-page": true,
		"cursor": true, "all": true,
		"dry-run": true, "n": true,
		"strict": true,
		"field": true, "fields": true,
		"id-only": true,
		"help": true, "h": true,
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		if !strings.HasPrefix(arg, "-") {
			i++
			continue
		}

		// Strip leading dashes
		trimmed := strings.TrimLeft(arg, "-")

		// Handle --flag=value
		var name, value string
		if eqIdx := strings.Index(trimmed, "="); eqIdx >= 0 {
			name = trimmed[:eqIdx]
			value = trimmed[eqIdx+1:]
		} else {
			name = trimmed
			// Look ahead for value
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				i++
			} else {
				// Boolean flag
				value = "true"
			}
		}

		// Store global flags too
		if globalFlags[name] {
			result.Values[name] = value
			i++
			continue
		}

		// Check if this is a known parameter
		if spec, ok := flagMap[name]; ok {
			if spec.Type == "string[]" {
				// Split comma-separated values
				parts := strings.Split(value, ",")
				result.Slices[name] = append(result.Slices[name], parts...)
			} else {
				result.Values[name] = value
			}
		} else {
			// Unknown flag — store it anyway, might be useful
			result.Values[name] = value
		}

		i++
	}

	// Validate required params
	for _, p := range params {
		if !p.Required {
			continue
		}
		flagName := ParamToFlagName(p.Name)
		if !result.Has(flagName) {
			return result, fmt.Errorf("required flag --%s not provided", flagName)
		}
	}

	return result, nil
}

// IsHelpRequested checks if --help or -h is in the raw args.
func IsHelpRequested(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}
