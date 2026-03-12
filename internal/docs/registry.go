package docs

const BaseURL = "https://developers.plane.so"

// Entry represents a single documentation page.
type Entry struct {
	Title string
	Path  string // relative to BaseURL, e.g. "/api-reference/issue/add-issue"
}

// URL returns the full URL for this entry.
func (e Entry) URL() string {
	return BaseURL + e.Path
}

// Topic groups related doc entries under a heading.
type Topic struct {
	Name    string
	Entries []Entry
}

// Registry is the full set of doc topics.
var Registry = []Topic{
	{
		Name: "introduction",
		Entries: []Entry{
			{"API Introduction", "/api-reference/introduction"},
		},
	},
	{
		Name: "project",
		Entries: []Entry{
			{"Overview", "/api-reference/project/overview"},
			{"Create Project", "/api-reference/project/add-project"},
			{"List Projects", "/api-reference/project/list-projects"},
			{"Get Project", "/api-reference/project/get-project-detail"},
			{"Update Project", "/api-reference/project/update-project-detail"},
			{"Delete Project", "/api-reference/project/delete-project"},
		},
	},
	{
		Name: "state",
		Entries: []Entry{
			{"Overview", "/api-reference/state/overview"},
			{"Create State", "/api-reference/state/add-state"},
			{"List States", "/api-reference/state/list-states"},
			{"Get State", "/api-reference/state/get-state-detail"},
			{"Update State", "/api-reference/state/update-state-detail"},
			{"Delete State", "/api-reference/state/delete-state"},
		},
	},
	{
		Name: "label",
		Entries: []Entry{
			{"Overview", "/api-reference/label/overview"},
			{"Create Label", "/api-reference/label/add-label"},
			{"List Labels", "/api-reference/label/list-labels"},
			{"Get Label", "/api-reference/label/get-label-detail"},
			{"Update Label", "/api-reference/label/update-label-detail"},
			{"Delete Label", "/api-reference/label/delete-label"},
		},
	},
	{
		Name: "issue",
		Entries: []Entry{
			{"Overview", "/api-reference/issue/overview"},
			{"Create Work Item", "/api-reference/issue/add-issue"},
			{"List Work Items", "/api-reference/issue/list-issues"},
			{"Get Work Item", "/api-reference/issue/get-issue-detail"},
			{"Get by Sequence ID", "/api-reference/issue/get-issue-sequence-id"},
			{"Search Work Items", "/api-reference/issue/search-issues"},
			{"Update Work Item", "/api-reference/issue/update-issue-detail"},
			{"Delete Work Item", "/api-reference/issue/delete-issue"},
		},
	},
	{
		Name: "link",
		Entries: []Entry{
			{"Overview", "/api-reference/link/overview"},
			{"Add Link", "/api-reference/link/add-link"},
			{"List Links", "/api-reference/link/list-links"},
			{"Get Link", "/api-reference/link/get-link-detail"},
			{"Update Link", "/api-reference/link/update-link-detail"},
			{"Delete Link", "/api-reference/link/delete-link"},
		},
	},
	{
		Name: "activity",
		Entries: []Entry{
			{"Overview", "/api-reference/issue-activity/overview"},
			{"List Activities", "/api-reference/issue-activity/list-issue-activities"},
			{"Get Activity", "/api-reference/issue-activity/get-issue-activity-detail"},
		},
	},
	{
		Name: "comment",
		Entries: []Entry{
			{"Overview", "/api-reference/issue-comment/overview"},
			{"Add Comment", "/api-reference/issue-comment/add-issue-comment"},
			{"List Comments", "/api-reference/issue-comment/list-issue-comments"},
			{"Get Comment", "/api-reference/issue-comment/get-issue-comment-detail"},
			{"Update Comment", "/api-reference/issue-comment/update-issue-comment-detail"},
			{"Delete Comment", "/api-reference/issue-comment/delete-issue-comment"},
		},
	},
	{
		Name: "attachment",
		Entries: []Entry{
			{"Overview", "/api-reference/issue-attachments/overview"},
			{"Get Attachments", "/api-reference/issue-attachments/get-attachments"},
			{"Get Attachment Detail", "/api-reference/issue-attachments/get-attachment-detail"},
			{"Get Upload Credentials", "/api-reference/issue-attachments/get-upload-credentials"},
			{"Upload File", "/api-reference/issue-attachments/upload-file"},
			{"Complete Upload", "/api-reference/issue-attachments/complete-upload"},
			{"Update Attachment", "/api-reference/issue-attachments/update-attachment"},
			{"Delete Attachment", "/api-reference/issue-attachments/delete-attachment"},
		},
	},
	{
		Name: "cycle",
		Entries: []Entry{
			{"Overview", "/api-reference/cycle/overview"},
			{"Create Cycle", "/api-reference/cycle/add-cycle"},
			{"Add Work Items", "/api-reference/cycle/add-cycle-work-items"},
			{"Transfer Work Items", "/api-reference/cycle/transfer-cycle-work-items"},
			{"Archive Cycle", "/api-reference/cycle/archive-cycle"},
			{"List Cycles", "/api-reference/cycle/list-cycles"},
			{"Get Cycle", "/api-reference/cycle/get-cycle-detail"},
			{"List Cycle Work Items", "/api-reference/cycle/list-cycle-work-items"},
			{"List Archived Cycles", "/api-reference/cycle/list-archived-cycles"},
			{"Update Cycle", "/api-reference/cycle/update-cycle-detail"},
			{"Unarchive Cycle", "/api-reference/cycle/unarchive-cycle"},
			{"Remove Work Item", "/api-reference/cycle/remove-cycle-work-item"},
			{"Delete Cycle", "/api-reference/cycle/delete-cycle"},
		},
	},
	{
		Name: "module",
		Entries: []Entry{
			{"Overview", "/api-reference/module/overview"},
			{"Create Module", "/api-reference/module/add-module"},
			{"Add Work Items", "/api-reference/module/add-module-work-items"},
			{"Archive Module", "/api-reference/module/archive-module"},
			{"List Modules", "/api-reference/module/list-modules"},
			{"Get Module", "/api-reference/module/get-module-detail"},
			{"List Module Work Items", "/api-reference/module/list-module-work-items"},
			{"List Archived Modules", "/api-reference/module/list-archived-modules"},
			{"Update Module", "/api-reference/module/update-module-detail"},
			{"Unarchive Module", "/api-reference/module/unarchive-module"},
			{"Remove Work Item", "/api-reference/module/remove-module-work-item"},
			{"Delete Module", "/api-reference/module/delete-module"},
		},
	},
	{
		Name: "page",
		Entries: []Entry{
			{"Overview", "/api-reference/page/overview"},
			{"Add Workspace Page", "/api-reference/page/add-workspace-page"},
			{"Add Project Page", "/api-reference/page/add-project-page"},
			{"Get Workspace Page", "/api-reference/page/get-workspace-page"},
			{"Get Project Page", "/api-reference/page/get-project-page"},
		},
	},
	{
		Name: "intake",
		Entries: []Entry{
			{"Overview", "/api-reference/intake-issue/overview"},
			{"Add Intake Issue", "/api-reference/intake-issue/add-intake-issue"},
			{"List Intake Issues", "/api-reference/intake-issue/list-intake-issues"},
			{"Get Intake Issue", "/api-reference/intake-issue/get-intake-issue-detail"},
			{"Update Intake Issue", "/api-reference/intake-issue/update-intake-issue-detail"},
			{"Delete Intake Issue", "/api-reference/intake-issue/delete-intake-issue"},
		},
	},
	{
		Name: "worklog",
		Entries: []Entry{
			{"Overview", "/api-reference/worklogs/overview"},
			{"Create Worklog", "/api-reference/worklogs/create-worklog"},
			{"Get Worklogs for Issue", "/api-reference/worklogs/get-worklogs-for-issue"},
			{"Get Total Time", "/api-reference/worklogs/get-total-time"},
			{"Update Worklog", "/api-reference/worklogs/update-worklog"},
			{"Delete Worklog", "/api-reference/worklogs/delete-worklog"},
		},
	},
	{
		Name: "epic",
		Entries: []Entry{
			{"Overview", "/api-reference/epics/overview"},
			{"List Epics", "/api-reference/epics/list-epics"},
			{"Get Epic", "/api-reference/epics/get-epic-detail"},
		},
	},
	{
		Name: "initiative",
		Entries: []Entry{
			{"Overview", "/api-reference/initiative/overview"},
			{"Create Initiative", "/api-reference/initiative/add-initiative"},
			{"List Initiatives", "/api-reference/initiative/list-initiatives"},
			{"Get Initiative", "/api-reference/initiative/get-initiative-detail"},
			{"Update Initiative", "/api-reference/initiative/update-initiative-detail"},
			{"Delete Initiative", "/api-reference/initiative/delete-initiative"},
		},
	},
	{
		Name: "customer",
		Entries: []Entry{
			{"Overview", "/api-reference/customer/overview"},
			{"Add Customer", "/api-reference/customer/add-customer"},
			{"Link Work Items", "/api-reference/customer/link-work-items-to-customer"},
			{"List Customers", "/api-reference/customer/list-customers"},
			{"Get Customer", "/api-reference/customer/get-customer-detail"},
			{"List Customer Work Items", "/api-reference/customer/list-customer-work-items"},
			{"Update Customer", "/api-reference/customer/update-customer-detail"},
			{"Unlink Work Item", "/api-reference/customer/unlink-work-item-from-customer"},
			{"Delete Customer", "/api-reference/customer/delete-customer"},
		},
	},
	{
		Name: "teamspace",
		Entries: []Entry{
			{"Overview", "/api-reference/teamspace/overview"},
			{"Create Teamspace", "/api-reference/teamspace/add-teamspace"},
			{"List Teamspaces", "/api-reference/teamspace/list-teamspaces"},
			{"Get Teamspace", "/api-reference/teamspace/get-teamspace-detail"},
			{"Update Teamspace", "/api-reference/teamspace/update-teamspace-detail"},
			{"Delete Teamspace", "/api-reference/teamspace/delete-teamspace"},
		},
	},
	{
		Name: "sticky",
		Entries: []Entry{
			{"Overview", "/api-reference/sticky/overview"},
			{"Add Sticky", "/api-reference/sticky/add-sticky"},
			{"List Stickies", "/api-reference/sticky/list-stickies"},
			{"Get Sticky", "/api-reference/sticky/get-sticky-detail"},
			{"Update Sticky", "/api-reference/sticky/update-sticky-detail"},
			{"Delete Sticky", "/api-reference/sticky/delete-sticky"},
		},
	},
	{
		Name: "member",
		Entries: []Entry{
			{"Overview", "/api-reference/members/overview"},
			{"Get Workspace Members", "/api-reference/members/get-workspace-members"},
			{"Get Project Members", "/api-reference/members/get-project-members"},
		},
	},
	{
		Name: "user",
		Entries: []Entry{
			{"Overview", "/api-reference/user/overview"},
			{"Get Current User", "/api-reference/user/get-current-user"},
		},
	},
}

// Lookup finds a topic by name (case-insensitive).
func Lookup(name string) *Topic {
	for i := range Registry {
		if equalsIgnoreCase(Registry[i].Name, name) {
			return &Registry[i]
		}
	}
	return nil
}

// LookupEntry finds a specific entry within a topic by matching action keywords.
func LookupEntry(topicName, action string) *Entry {
	topic := Lookup(topicName)
	if topic == nil {
		return nil
	}
	for i := range topic.Entries {
		if containsIgnoreCase(topic.Entries[i].Title, action) {
			return &topic.Entries[i]
		}
	}
	return nil
}

func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func containsIgnoreCase(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			ch, cn := haystack[i+j], needle[j]
			if ch >= 'A' && ch <= 'Z' {
				ch += 32
			}
			if cn >= 'A' && cn <= 'Z' {
				cn += 32
			}
			if ch != cn {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
