package docs

import (
	"regexp"
	"strings"
)

var (
	methodPathRe = regexp.MustCompile(`(?i)(GET|POST|PATCH|PUT|DELETE)\s*(/api/v1/[^\s]+)`)
	statusCodeRe = regexp.MustCompile(`(?i)(?:Response|Status)[^0-9]*(\d{3})`)
	// Matches: `param_name`:requiredtype or `param_name`:optionaltype
	inlineParamRe = regexp.MustCompile("(?m)^`(\\w+)`:(required|optional)(\\S+)")
	// Matches a docs enum bullet: "- `blocked_by` - Blocked By". Plane renders
	// a parameter's valid values as a <ul> under its description, which
	// htmlToMarkdown turns into lines of this shape.
	enumBulletRe = regexp.MustCompile("^-\\s+`([A-Za-z0-9_.-]+)`")
)

// ParseEndpointPage extracts an EndpointSpec from a markdown doc page.
func ParseEndpointPage(markdown, topicName string, entry Entry) *EndpointSpec {
	spec := &EndpointSpec{
		TopicName:  topicName,
		EntryTitle: entry.Title,
		SourceURL:  entry.URL,
	}

	// Extract method + path
	if m := methodPathRe.FindStringSubmatch(markdown); len(m) == 3 {
		spec.Method = strings.ToUpper(m[1])
		spec.PathTemplate = cleanPath(m[2])
	} else {
		// Fallback: infer method from entry title
		spec.Method = inferMethodFromTitle(entry.Title)
	}

	// Extract path parameters from template
	if spec.PathTemplate != "" {
		spec.Params = append(spec.Params, extractPathParams(spec.PathTemplate)...)
	}

	// Extract parameters: try inline format first (Plane docs style),
	// fall back to markdown tables
	inlineParams := parseInlineParams(markdown)
	if len(inlineParams) > 0 {
		spec.Params = append(spec.Params, inlineParams...)
	} else {
		tableParams := parseParamTables(markdown)
		spec.Params = append(spec.Params, tableParams...)
	}

	// Extract status code
	if m := statusCodeRe.FindStringSubmatch(markdown); len(m) == 2 {
		var code int
		for _, c := range m[1] {
			code = code*10 + int(c-'0')
		}
		spec.StatusCode = code
	} else {
		spec.StatusCode = inferStatusCode(spec.Method)
	}

	return spec
}

func cleanPath(path string) string {
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimRight(path, "/")
	return path + "/"
}

func inferMethodFromTitle(title string) string {
	lower := strings.ToLower(title)
	switch {
	case strings.HasPrefix(lower, "create") || strings.HasPrefix(lower, "add"):
		return "POST"
	case strings.HasPrefix(lower, "list") || strings.HasPrefix(lower, "get") || strings.HasPrefix(lower, "search"):
		return "GET"
	case strings.HasPrefix(lower, "update"):
		return "PATCH"
	case strings.HasPrefix(lower, "delete") || strings.HasPrefix(lower, "remove"):
		return "DELETE"
	case strings.Contains(lower, "archive") && !strings.Contains(lower, "unarchive") && !strings.Contains(lower, "list"):
		return "POST"
	case strings.Contains(lower, "unarchive"):
		return "DELETE"
	case strings.Contains(lower, "transfer"):
		return "POST"
	default:
		return "GET"
	}
}

func inferStatusCode(method string) int {
	switch method {
	case "POST":
		return 201
	case "DELETE":
		return 204
	default:
		return 200
	}
}

var pathParamRe = regexp.MustCompile(`\{(\w+)\}`)

func extractPathParams(pathTemplate string) []ParamSpec {
	matches := pathParamRe.FindAllStringSubmatch(pathTemplate, -1)
	var params []ParamSpec
	for _, m := range matches {
		name := m[1]
		if name == "workspace_slug" || name == "project_id" {
			continue
		}
		params = append(params, ParamSpec{
			Name:     name,
			Type:     "string",
			Required: true,
			Location: ParamPath,
		})
	}
	return params
}

// parseInlineParams parses the Plane docs inline parameter format:
//
//	`param_name`:requiredstring
//	Description text.
//
// Parameters are grouped under section headers like "### Path Parameters",
// "### Body Parameters", "### Query Parameters".
func parseInlineParams(markdown string) []ParamSpec {
	var params []ParamSpec
	lines := strings.Split(markdown, "\n")
	location := ParamBody // default

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Detect section headers to determine param location
		if strings.HasPrefix(line, "###") || strings.HasPrefix(line, "## ") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "path param") {
				location = ParamPath
			} else if strings.Contains(lower, "query param") {
				location = ParamQuery
			} else if strings.Contains(lower, "body param") {
				location = ParamBody
			} else if strings.Contains(lower, "scope") || strings.Contains(lower, "response") {
				// Stop parsing params once we hit scopes or response sections
				break
			}
			continue
		}

		// Try to match inline param pattern
		m := inlineParamRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		name := m[1]
		required := m[2] == "required"
		typStr := normalizeType(m[3])

		// Skip path params already extracted from the URL template
		if location == ParamPath && (name == "workspace_slug" || name == "project_id") {
			continue
		}
		// Skip path params entirely — they're already extracted from the template
		if location == ParamPath {
			continue
		}

		// A parameter is followed by an optional prose description and then an
		// optional bulleted list of its valid values.
		desc := ""
		next := i + 1
		for next < len(lines) && strings.TrimSpace(lines[next]) == "" {
			next++
		}
		if next < len(lines) {
			nextLine := strings.TrimSpace(lines[next])
			// Description line: not another param, not a header, and not
			// already the start of the value list. Without that last check, a
			// param documented only by bullets takes its first bullet as the
			// description — which is why --priority used to read "- high - High".
			if !inlineParamRe.MatchString(nextLine) &&
				!strings.HasPrefix(nextLine, "#") && !enumBulletRe.MatchString(nextLine) {
				desc = nextLine
				next++
			}
		}

		bulletEnum, consumed := parseEnumBullets(lines, next)
		i = consumed - 1 // the loop's i++ moves past what we read

		p := ParamSpec{
			Name:        name,
			Type:        typStr,
			Required:    required,
			Description: desc,
			Location:    location,
		}
		// A bulleted list is the more reliable signal; fall back to scraping
		// the description for pages that use the table format instead.
		if len(bulletEnum) > 1 {
			p.Enum = bulletEnum
		} else if enumVals := extractEnum(desc); len(enumVals) > 0 {
			p.Enum = enumVals
		}
		params = append(params, p)
	}

	return params
}

// parseEnumBullets reads a run of enum bullets beginning at or after start.
// Blank lines are skipped: htmlToMarkdown surrounds each list item with
// newlines, so the items arrive separated by blank lines. Scanning stops at
// the first non-blank line that is not a bullet — in practice the next
// parameter or section header — so it cannot run into an unrelated list.
//
// It returns the values and the index just past the final bullet, leaving any
// trailing blank lines for the caller.
func parseEnumBullets(lines []string, start int) ([]string, int) {
	var values []string
	end := start
	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		m := enumBulletRe.FindStringSubmatch(line)
		if m == nil {
			break
		}
		values = append(values, m[1])
		end = i + 1
	}
	if len(values) == 0 {
		return nil, start
	}
	return values, end
}

// parseParamTables extracts parameters from markdown tables.
// Looks for tables with columns like: Name | Type | Required | Description
func parseParamTables(markdown string) []ParamSpec {
	var params []ParamSpec
	lines := strings.Split(markdown, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !isTableHeader(line) {
			continue
		}

		headers := splitTableRow(line)
		colMap := mapColumns(headers)
		if colMap.name < 0 {
			continue
		}

		i++
		if i < len(lines) && isSeparatorLine(strings.TrimSpace(lines[i])) {
			i++
		}

		location := inferLocationFromContext(lines, i-3)

		for ; i < len(lines); i++ {
			row := strings.TrimSpace(lines[i])
			if row == "" || !strings.Contains(row, "|") {
				break
			}
			if isSeparatorLine(row) {
				continue
			}

			cells := splitTableRow(row)
			p := extractParamFromRow(cells, colMap, location)
			if p != nil {
				params = append(params, *p)
			}
		}
	}

	return params
}

type columnMap struct {
	name, typ, required, desc int
}

func mapColumns(headers []string) columnMap {
	cm := columnMap{name: -1, typ: -1, required: -1, desc: -1}
	for i, h := range headers {
		lower := strings.ToLower(strings.TrimSpace(h))
		switch lower {
		case "name", "parameter", "field", "property":
			cm.name = i
		case "type", "data type":
			cm.typ = i
		case "required", "mandatory":
			cm.required = i
		case "description", "details":
			cm.desc = i
		}
	}
	return cm
}

func isTableHeader(line string) bool {
	if !strings.Contains(line, "|") {
		return false
	}
	lower := strings.ToLower(line)
	return (strings.Contains(lower, "name") || strings.Contains(lower, "parameter") || strings.Contains(lower, "field")) &&
		(strings.Contains(lower, "type") || strings.Contains(lower, "required") || strings.Contains(lower, "description"))
}

func isSeparatorLine(line string) bool {
	cleaned := strings.ReplaceAll(line, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "|", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	return cleaned == ""
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func inferLocationFromContext(lines []string, approxIdx int) ParamLocation {
	start := approxIdx - 5
	if start < 0 {
		start = 0
	}
	end := approxIdx + 2
	if end > len(lines) {
		end = len(lines)
	}

	for i := start; i < end; i++ {
		lower := strings.ToLower(lines[i])
		if strings.Contains(lower, "query") && strings.Contains(lower, "param") {
			return ParamQuery
		}
		if strings.Contains(lower, "request body") || strings.Contains(lower, "body param") || strings.Contains(lower, "payload") {
			return ParamBody
		}
		if strings.Contains(lower, "path param") {
			return ParamPath
		}
	}
	return ParamBody
}

func extractParamFromRow(cells []string, cm columnMap, defaultLocation ParamLocation) *ParamSpec {
	if cm.name < 0 || cm.name >= len(cells) {
		return nil
	}

	name := strings.TrimSpace(cells[cm.name])
	name = strings.Trim(name, "`*")
	if name == "" {
		return nil
	}

	p := &ParamSpec{
		Name:     name,
		Type:     "string",
		Location: defaultLocation,
	}

	if cm.typ >= 0 && cm.typ < len(cells) {
		p.Type = normalizeType(strings.TrimSpace(cells[cm.typ]))
	}
	if cm.required >= 0 && cm.required < len(cells) {
		req := strings.ToLower(strings.TrimSpace(cells[cm.required]))
		p.Required = req == "yes" || req == "true" || req == "required" || req == "✓" || req == "✅"
	}
	if cm.desc >= 0 && cm.desc < len(cells) {
		p.Description = strings.TrimSpace(cells[cm.desc])
		if enumVals := extractEnum(p.Description); len(enumVals) > 0 {
			p.Enum = enumVals
		}
	}

	return p
}

// enumFromDescRe matches patterns like "value1, value2, value3" at the end of a description
// after a colon or keywords like "one of", "values:", "options:", "enum:".
var enumFromDescRe = regexp.MustCompile(`(?i)(?::|one of|values?|options?|enum)\s*[:=]?\s*` + "`?" + `([\w]+(?:\s*,\s*[\w]+)+)` + "`?")

// extractEnum attempts to extract enum values from a parameter description.
// It looks for patterns like "Priority: urgent, high, medium, low, none"
// or "one of: active, paused, completed".
func extractEnum(desc string) []string {
	m := enumFromDescRe.FindStringSubmatch(desc)
	if m == nil {
		return nil
	}
	raw := m[1]
	parts := strings.Split(raw, ",")
	var values []string
	for _, p := range parts {
		v := strings.TrimSpace(p)
		v = strings.Trim(v, "`\"'")
		if v == "" {
			continue
		}
		// Bail out on numeric values. Every real enum in this API is a set of
		// identifiers; a number means we matched prose describing limits, as
		// in "Number of results per page (default: 20, max: 100)" — which
		// otherwise yields the nonsense enum ["20", "max"].
		if isAllDigits(v) {
			return nil
		}
		values = append(values, v)
	}
	if len(values) < 2 {
		return nil
	}
	return values
}

// isAllDigits reports whether s is non-empty and contains only ASCII digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func normalizeType(t string) string {
	lower := strings.ToLower(strings.TrimSpace(t))
	lower = strings.Trim(lower, "`")
	switch {
	case lower == "string" || lower == "str" || lower == "uuid" || lower == "date" || lower == "datetime":
		return "string"
	case lower == "integer" || lower == "int" || lower == "number" || lower == "float":
		return "number"
	case lower == "boolean" || lower == "bool":
		return "boolean"
	case strings.Contains(lower, "array") || strings.Contains(lower, "[]") || strings.Contains(lower, "list"):
		return "string[]"
	default:
		return "string"
	}
}
