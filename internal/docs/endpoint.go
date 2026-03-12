package docs

import "strings"

// ParamLocation indicates where a parameter is sent.
type ParamLocation string

const (
	ParamPath  ParamLocation = "path"
	ParamQuery ParamLocation = "query"
	ParamBody  ParamLocation = "body"
)

// ParamSpec describes a single API endpoint parameter.
type ParamSpec struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"` // "string", "string[]", "number", "boolean"
	Required    bool          `json:"required"`
	Description string        `json:"description,omitempty"`
	Location    ParamLocation `json:"location"`
}

// EndpointSpec describes a parsed API endpoint.
type EndpointSpec struct {
	TopicName    string      `json:"topic_name"`
	EntryTitle   string      `json:"entry_title"`
	SourceURL    string      `json:"source_url"`
	Method       string      `json:"method"`
	PathTemplate string      `json:"path_template"`
	Params       []ParamSpec `json:"params,omitempty"`
	StatusCode   int         `json:"status_code,omitempty"`
}

// RequiresWorkspace returns true if the path contains {workspace_slug}.
func (s *EndpointSpec) RequiresWorkspace() bool {
	return strings.Contains(s.PathTemplate, "{workspace_slug}")
}

// RequiresProject returns true if the path contains {project_id}.
func (s *EndpointSpec) RequiresProject() bool {
	return strings.Contains(s.PathTemplate, "{project_id}")
}
