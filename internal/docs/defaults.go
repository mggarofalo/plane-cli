package docs

import "strings"

// DefaultBaseURL is the default docs URL used when no custom URL is configured.
const DefaultBaseURL = "https://developers.plane.so"

// RebaseTopics returns a deep copy of DefaultTopics with all entry URLs
// rewritten from DefaultBaseURL to the given base URL. If baseURL equals
// DefaultBaseURL (or is empty), it returns DefaultTopics directly.
func RebaseTopics(baseURL string) []Topic {
	if baseURL == "" || baseURL == DefaultBaseURL {
		return DefaultTopics
	}
	// Normalize trailing slash to prevent double-slash URLs (e.g.
	// "https://custom.example.com/" → "https://custom.example.com").
	// This mirrors the normalization in FetchLLMSTxt.
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == DefaultBaseURL {
		return DefaultTopics
	}
	out := make([]Topic, len(DefaultTopics))
	for i, t := range DefaultTopics {
		entries := make([]Entry, len(t.Entries))
		for j, e := range t.Entries {
			entries[j] = Entry{
				Title:       e.Title,
				URL:         strings.Replace(e.URL, DefaultBaseURL, baseURL, 1),
				Description: e.Description,
			}
		}
		out[i] = Topic{
			Name:    t.Name,
			Entries: entries,
		}
	}
	return out
}

// DefaultTopics contains the hardcoded doc entries used as a fallback when
// the remote llms.txt cannot be fetched and no cache exists.
var DefaultTopics = []Topic{
	{
		Name: "introduction",
		Entries: []Entry{
			{Title: "API Introduction", URL: DefaultBaseURL + "/api-reference/introduction"},
		},
	},
	{
		Name: "project",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/project/overview"},
			{Title: "Create Project", URL: DefaultBaseURL + "/api-reference/project/add-project"},
			{Title: "List Projects", URL: DefaultBaseURL + "/api-reference/project/list-projects"},
			{Title: "Get Project", URL: DefaultBaseURL + "/api-reference/project/get-project-detail"},
			{Title: "Update Project", URL: DefaultBaseURL + "/api-reference/project/update-project-detail"},
			{Title: "Delete Project", URL: DefaultBaseURL + "/api-reference/project/delete-project"},
		},
	},
	{
		Name: "state",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/state/overview"},
			{Title: "Create State", URL: DefaultBaseURL + "/api-reference/state/add-state"},
			{Title: "List States", URL: DefaultBaseURL + "/api-reference/state/list-states"},
			{Title: "Get State", URL: DefaultBaseURL + "/api-reference/state/get-state-detail"},
			{Title: "Update State", URL: DefaultBaseURL + "/api-reference/state/update-state-detail"},
			{Title: "Delete State", URL: DefaultBaseURL + "/api-reference/state/delete-state"},
		},
	},
	{
		Name: "label",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/label/overview"},
			{Title: "Create Label", URL: DefaultBaseURL + "/api-reference/label/add-label"},
			{Title: "List Labels", URL: DefaultBaseURL + "/api-reference/label/list-labels"},
			{Title: "Get Label", URL: DefaultBaseURL + "/api-reference/label/get-label-detail"},
			{Title: "Update Label", URL: DefaultBaseURL + "/api-reference/label/update-label-detail"},
			{Title: "Delete Label", URL: DefaultBaseURL + "/api-reference/label/delete-label"},
		},
	},
	{
		Name: "issue",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/issue/overview"},
			{Title: "Create Work Item", URL: DefaultBaseURL + "/api-reference/issue/add-issue"},
			{Title: "List Work Items", URL: DefaultBaseURL + "/api-reference/issue/list-issues"},
			{Title: "Get Work Item", URL: DefaultBaseURL + "/api-reference/issue/get-issue-detail"},
			{Title: "Get by Sequence ID", URL: DefaultBaseURL + "/api-reference/issue/get-issue-sequence-id"},
			{Title: "Search Work Items", URL: DefaultBaseURL + "/api-reference/issue/search-issues"},
			{Title: "Update Work Item", URL: DefaultBaseURL + "/api-reference/issue/update-issue-detail"},
			{Title: "Delete Work Item", URL: DefaultBaseURL + "/api-reference/issue/delete-issue"},
		},
	},
	{
		Name: "link",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/link/overview"},
			{Title: "Add Link", URL: DefaultBaseURL + "/api-reference/link/add-link"},
			{Title: "List Links", URL: DefaultBaseURL + "/api-reference/link/list-links"},
			{Title: "Get Link", URL: DefaultBaseURL + "/api-reference/link/get-link-detail"},
			{Title: "Update Link", URL: DefaultBaseURL + "/api-reference/link/update-link-detail"},
			{Title: "Delete Link", URL: DefaultBaseURL + "/api-reference/link/delete-link"},
		},
	},
	{
		Name: "activity",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/issue-activity/overview"},
			{Title: "List Activities", URL: DefaultBaseURL + "/api-reference/issue-activity/list-issue-activities"},
			{Title: "Get Activity", URL: DefaultBaseURL + "/api-reference/issue-activity/get-issue-activity-detail"},
		},
	},
	{
		Name: "comment",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/issue-comment/overview"},
			{Title: "Add Comment", URL: DefaultBaseURL + "/api-reference/issue-comment/add-issue-comment"},
			{Title: "List Comments", URL: DefaultBaseURL + "/api-reference/issue-comment/list-issue-comments"},
			{Title: "Get Comment", URL: DefaultBaseURL + "/api-reference/issue-comment/get-issue-comment-detail"},
			{Title: "Update Comment", URL: DefaultBaseURL + "/api-reference/issue-comment/update-issue-comment-detail"},
			{Title: "Delete Comment", URL: DefaultBaseURL + "/api-reference/issue-comment/delete-issue-comment"},
		},
	},
	{
		Name: "attachment",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/issue-attachments/overview"},
			{Title: "Get Attachments", URL: DefaultBaseURL + "/api-reference/issue-attachments/get-attachments"},
			{Title: "Get Attachment Detail", URL: DefaultBaseURL + "/api-reference/issue-attachments/get-attachment-detail"},
			{Title: "Get Upload Credentials", URL: DefaultBaseURL + "/api-reference/issue-attachments/get-upload-credentials"},
			{Title: "Upload File", URL: DefaultBaseURL + "/api-reference/issue-attachments/upload-file"},
			{Title: "Complete Upload", URL: DefaultBaseURL + "/api-reference/issue-attachments/complete-upload"},
			{Title: "Update Attachment", URL: DefaultBaseURL + "/api-reference/issue-attachments/update-attachment"},
			{Title: "Delete Attachment", URL: DefaultBaseURL + "/api-reference/issue-attachments/delete-attachment"},
		},
	},
	{
		Name: "cycle",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/cycle/overview"},
			{Title: "Create Cycle", URL: DefaultBaseURL + "/api-reference/cycle/add-cycle"},
			{Title: "Add Work Items", URL: DefaultBaseURL + "/api-reference/cycle/add-cycle-work-items"},
			{Title: "Transfer Work Items", URL: DefaultBaseURL + "/api-reference/cycle/transfer-cycle-work-items"},
			{Title: "Archive Cycle", URL: DefaultBaseURL + "/api-reference/cycle/archive-cycle"},
			{Title: "List Cycles", URL: DefaultBaseURL + "/api-reference/cycle/list-cycles"},
			{Title: "Get Cycle", URL: DefaultBaseURL + "/api-reference/cycle/get-cycle-detail"},
			{Title: "List Cycle Work Items", URL: DefaultBaseURL + "/api-reference/cycle/list-cycle-work-items"},
			{Title: "List Archived Cycles", URL: DefaultBaseURL + "/api-reference/cycle/list-archived-cycles"},
			{Title: "Update Cycle", URL: DefaultBaseURL + "/api-reference/cycle/update-cycle-detail"},
			{Title: "Unarchive Cycle", URL: DefaultBaseURL + "/api-reference/cycle/unarchive-cycle"},
			{Title: "Remove Work Item", URL: DefaultBaseURL + "/api-reference/cycle/remove-cycle-work-item"},
			{Title: "Delete Cycle", URL: DefaultBaseURL + "/api-reference/cycle/delete-cycle"},
		},
	},
	{
		Name: "module",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/module/overview"},
			{Title: "Create Module", URL: DefaultBaseURL + "/api-reference/module/add-module"},
			{Title: "Add Work Items", URL: DefaultBaseURL + "/api-reference/module/add-module-work-items"},
			{Title: "Archive Module", URL: DefaultBaseURL + "/api-reference/module/archive-module"},
			{Title: "List Modules", URL: DefaultBaseURL + "/api-reference/module/list-modules"},
			{Title: "Get Module", URL: DefaultBaseURL + "/api-reference/module/get-module-detail"},
			{Title: "List Module Work Items", URL: DefaultBaseURL + "/api-reference/module/list-module-work-items"},
			{Title: "List Archived Modules", URL: DefaultBaseURL + "/api-reference/module/list-archived-modules"},
			{Title: "Update Module", URL: DefaultBaseURL + "/api-reference/module/update-module-detail"},
			{Title: "Unarchive Module", URL: DefaultBaseURL + "/api-reference/module/unarchive-module"},
			{Title: "Remove Work Item", URL: DefaultBaseURL + "/api-reference/module/remove-module-work-item"},
			{Title: "Delete Module", URL: DefaultBaseURL + "/api-reference/module/delete-module"},
		},
	},
	{
		Name: "page",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/page/overview"},
			{Title: "Add Workspace Page", URL: DefaultBaseURL + "/api-reference/page/add-workspace-page"},
			{Title: "Add Project Page", URL: DefaultBaseURL + "/api-reference/page/add-project-page"},
			{Title: "Get Workspace Page", URL: DefaultBaseURL + "/api-reference/page/get-workspace-page"},
			{Title: "Get Project Page", URL: DefaultBaseURL + "/api-reference/page/get-project-page"},
		},
	},
	{
		Name: "intake",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/intake-issue/overview"},
			{Title: "Add Intake Issue", URL: DefaultBaseURL + "/api-reference/intake-issue/add-intake-issue"},
			{Title: "List Intake Issues", URL: DefaultBaseURL + "/api-reference/intake-issue/list-intake-issues"},
			{Title: "Get Intake Issue", URL: DefaultBaseURL + "/api-reference/intake-issue/get-intake-issue-detail"},
			{Title: "Update Intake Issue", URL: DefaultBaseURL + "/api-reference/intake-issue/update-intake-issue-detail"},
			{Title: "Delete Intake Issue", URL: DefaultBaseURL + "/api-reference/intake-issue/delete-intake-issue"},
		},
	},
	{
		Name: "worklog",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/worklogs/overview"},
			{Title: "Create Worklog", URL: DefaultBaseURL + "/api-reference/worklogs/create-worklog"},
			{Title: "Get Worklogs for Issue", URL: DefaultBaseURL + "/api-reference/worklogs/get-worklogs-for-issue"},
			{Title: "Get Total Time", URL: DefaultBaseURL + "/api-reference/worklogs/get-total-time"},
			{Title: "Update Worklog", URL: DefaultBaseURL + "/api-reference/worklogs/update-worklog"},
			{Title: "Delete Worklog", URL: DefaultBaseURL + "/api-reference/worklogs/delete-worklog"},
		},
	},
	{
		Name: "epic",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/epics/overview"},
			{Title: "List Epics", URL: DefaultBaseURL + "/api-reference/epics/list-epics"},
			{Title: "Get Epic", URL: DefaultBaseURL + "/api-reference/epics/get-epic-detail"},
		},
	},
	{
		Name: "initiative",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/initiative/overview"},
			{Title: "Create Initiative", URL: DefaultBaseURL + "/api-reference/initiative/add-initiative"},
			{Title: "List Initiatives", URL: DefaultBaseURL + "/api-reference/initiative/list-initiatives"},
			{Title: "Get Initiative", URL: DefaultBaseURL + "/api-reference/initiative/get-initiative-detail"},
			{Title: "Update Initiative", URL: DefaultBaseURL + "/api-reference/initiative/update-initiative-detail"},
			{Title: "Delete Initiative", URL: DefaultBaseURL + "/api-reference/initiative/delete-initiative"},
		},
	},
	{
		Name: "customer",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/customer/overview"},
			{Title: "Add Customer", URL: DefaultBaseURL + "/api-reference/customer/add-customer"},
			{Title: "Link Work Items", URL: DefaultBaseURL + "/api-reference/customer/link-work-items-to-customer"},
			{Title: "List Customers", URL: DefaultBaseURL + "/api-reference/customer/list-customers"},
			{Title: "Get Customer", URL: DefaultBaseURL + "/api-reference/customer/get-customer-detail"},
			{Title: "List Customer Work Items", URL: DefaultBaseURL + "/api-reference/customer/list-customer-work-items"},
			{Title: "Update Customer", URL: DefaultBaseURL + "/api-reference/customer/update-customer-detail"},
			{Title: "Unlink Work Item", URL: DefaultBaseURL + "/api-reference/customer/unlink-work-item-from-customer"},
			{Title: "Delete Customer", URL: DefaultBaseURL + "/api-reference/customer/delete-customer"},
		},
	},
	{
		Name: "teamspace",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/teamspace/overview"},
			{Title: "Create Teamspace", URL: DefaultBaseURL + "/api-reference/teamspace/add-teamspace"},
			{Title: "List Teamspaces", URL: DefaultBaseURL + "/api-reference/teamspace/list-teamspaces"},
			{Title: "Get Teamspace", URL: DefaultBaseURL + "/api-reference/teamspace/get-teamspace-detail"},
			{Title: "Update Teamspace", URL: DefaultBaseURL + "/api-reference/teamspace/update-teamspace-detail"},
			{Title: "Delete Teamspace", URL: DefaultBaseURL + "/api-reference/teamspace/delete-teamspace"},
		},
	},
	{
		Name: "sticky",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/sticky/overview"},
			{Title: "Add Sticky", URL: DefaultBaseURL + "/api-reference/sticky/add-sticky"},
			{Title: "List Stickies", URL: DefaultBaseURL + "/api-reference/sticky/list-stickies"},
			{Title: "Get Sticky", URL: DefaultBaseURL + "/api-reference/sticky/get-sticky-detail"},
			{Title: "Update Sticky", URL: DefaultBaseURL + "/api-reference/sticky/update-sticky-detail"},
			{Title: "Delete Sticky", URL: DefaultBaseURL + "/api-reference/sticky/delete-sticky"},
		},
	},
	{
		Name: "member",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/members/overview"},
			{Title: "Get Workspace Members", URL: DefaultBaseURL + "/api-reference/members/get-workspace-members"},
			{Title: "Get Project Members", URL: DefaultBaseURL + "/api-reference/members/get-project-members"},
		},
	},
	{
		Name: "user",
		Entries: []Entry{
			{Title: "Overview", URL: DefaultBaseURL + "/api-reference/user/overview"},
			{Title: "Get Current User", URL: DefaultBaseURL + "/api-reference/user/get-current-user"},
		},
	},
}
