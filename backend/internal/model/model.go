package model

import "time"

type StatusCategory string

const (
	StatusToDo       StatusCategory = "todo"
	StatusInProgress StatusCategory = "in_progress"
	StatusDone       StatusCategory = "done"
)

const (
	KindPersonal = "personal"
	RoleOwner    = "owner"
)

type User struct {
	ID          string
	DisplayName string
}

type Session struct {
	UserID        string
	DisplayName   string
	WorkspaceID   string
	WorkspaceName string
	MembershipID  string
}

// Profile is the editable identity shown for a session: the user's display
// name and the workspace name.
type Profile struct {
	WorkspaceID   string
	DisplayName   string
	WorkspaceName string
}

type Project struct {
	ID          string
	WorkspaceID string
	Name        string
	Description string
	Icon        string
	TaskCount   int
}

type Plan struct {
	ID          string
	WorkspaceID string
	ProjectID   string
	TeamID      string
	TeamName    string
	Name        string
	Description string
	Icon        string
	TaskCount   int
	DoneCount   int
}

type Status struct {
	ID          string
	WorkspaceID string
	PlanID      string
	Title       string
	Description string
	Color       string
	Category    StatusCategory
	Position    string
}

type Task struct {
	ID             string
	WorkspaceID    string
	ProjectID      string
	PlanID         string
	ProjectName    string
	StatusID       string
	CreatedBy      string
	Title          string
	Description    string
	Position       string
	StatusCategory StatusCategory
	DueDate        *time.Time
	// CompletedAt is set while the task is finished. It does not depend on
	// the task's status: moving a task between columns leaves it alone.
	CompletedAt *time.Time
	// ParentID is empty for a top-level task.
	ParentID         string
	SubtaskCount     int
	DoneSubtaskCount int
	CommentCount     int
}

type Comment struct {
	ID          string
	WorkspaceID string
	TaskID      string
	AuthorID    string
	AuthorName  string
	Body        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
