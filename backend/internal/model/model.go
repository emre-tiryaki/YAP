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

type ResourceKind string

const (
	ResourceFolder ResourceKind = "folder"
	ResourceFile   ResourceKind = "file"
)

// Resource is a folder or file in a workspace's resource tree.
type Resource struct {
	ID          string
	WorkspaceID string
	// ParentID is empty for an item at the top of the tree.
	ParentID  string
	Kind      ResourceKind
	Name      string
	MimeType  string
	Size      int64
	IsText    bool
	Revision  int64
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
	// TrashedAt is set while the item, or a folder it sits in, is in the trash.
	TrashedAt *time.Time
	// TrashedRootID is the item that was moved to the trash, taking this one
	// with it; equal to ID for the items listed in the trash.
	TrashedRootID string
}

type ResourceAction string

const (
	ResourceCreated  ResourceAction = "created"
	ResourceUploaded ResourceAction = "uploaded"
	ResourceEdited   ResourceAction = "edited"
	ResourceRenamed  ResourceAction = "renamed"
	ResourceMoved    ResourceAction = "moved"
	ResourceCopied   ResourceAction = "copied"
	ResourceTrashed  ResourceAction = "trashed"
	ResourceRestored ResourceAction = "restored"
	ResourcePurged   ResourceAction = "purged"
	ResourceLinked   ResourceAction = "linked"
	ResourceUnlinked ResourceAction = "unlinked"
)

// ResourceEvent is one entry of the resource activity log.
type ResourceEvent struct {
	ID           string
	WorkspaceID  string
	ResourceID   string
	ResourceKind ResourceKind
	ActorID      string
	ActorName    string
	Action       ResourceAction
	TaskID       string
	// Details is a JSON object; its keys depend on the action.
	Details   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LinkedTask is a task a resource is linked to, with where the task lives.
type LinkedTask struct {
	TaskID      string
	Title       string
	ProjectID   string
	ProjectName string
	PlanID      string
	CompletedAt *time.Time
}

// StagedFile is uploaded content written to a temporary file, waiting to be
// placed in the resource tree.
type StagedFile struct {
	TempPath string
	Size     int64
	// IsText reports valid UTF-8 without NUL bytes.
	IsText bool
	// Head holds the first bytes, for content type sniffing.
	Head []byte
}

// ResourceEventScope narrows an activity log listing; the zero value lists
// the whole workspace.
type ResourceEventScope struct {
	// ResourceID lists the events of the resource and everything ever placed
	// under it.
	ResourceID string
	// ProjectID lists the events of resources linked to the project's tasks
	// (and their contents) plus link changes on those tasks.
	ProjectID string
}
