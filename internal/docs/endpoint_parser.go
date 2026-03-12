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
		for _, p := range extractPathParams(spec.PathTemplate) {
			spec.Params = append(spec.Params, p)
		}
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

		// Next line(s) may be the description
		desc := ""
		if i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			// Description line: not empty, not another param, not a header
			if nextLine != "" && !inlineParamRe.MatchString(nextLine) && !strings.HasPrefix(nextLine, "#") {
				desc = nextLine
				i++ // skip description line
			}
		}

		params = append(params, ParamSpec{
			Name:        name,
			Type:        typStr,
			Required:    required,
			Description: desc,
			Location:    location,
		})
	}

	return params
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
		switch {
		case lower == "name" || lower == "parameter" || lower == "field" || lower == "property":
			cm.name = i
		case lower == "type" || lower == "data type":
			cm.typ = i
		case lower == "required" || lower == "mandatory":
			cm.required = i
		case lower == "description" || lower == "details":
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
	}

	return p
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
