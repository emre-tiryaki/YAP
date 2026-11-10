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

type Project struct {
	ID          string
	WorkspaceID string
	Name        string
	Description string
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
}
