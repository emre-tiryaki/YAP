package model

// SearchResults holds the live items of a workspace whose text matches a
// search query, grouped by kind.
type SearchResults struct {
	Projects  []SearchProject
	Plans     []SearchPlan
	Tasks     []SearchTask
	Resources []SearchResource
}

type SearchProject struct {
	ID   string
	Name string
	Icon string
}

type SearchPlan struct {
	ID          string
	ProjectID   string
	ProjectName string
	Name        string
	Icon        string
}

type SearchTask struct {
	ID          string
	ProjectID   string
	ProjectName string
	PlanID      string
	PlanName    string
	ParentID    string
	Title       string
	Done        bool
}

type SearchResource struct {
	ID         string
	ParentID   string
	ParentName string
	Kind       ResourceKind
	Name       string
	MimeType   string
	IsText     bool
}
