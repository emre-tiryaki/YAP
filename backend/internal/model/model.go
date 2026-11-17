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

// ExportTable is the live rows of one table in a data export, each row keyed
// by column name.
type ExportTable struct {
	Name string
	Rows []map[string]any
}

// ImportRowState says what an exported id stands for where it is imported.
type ImportRowState string

const (
	// RowAbsent: no row has the id; it can be used as it is.
	RowAbsent ImportRowState = "absent"
	// RowLive: a live row of the workspace has the id; it is already here.
	RowLive ImportRowState = "live"
	// RowDeleted: a deleted row of the workspace has the id; it can be
	// brought back.
	RowDeleted ImportRowState = "deleted"
	// RowElsewhere: a row of another workspace has the id.
	RowElsewhere ImportRowState = "elsewhere"
)

// The Import* types are rows read back from a data export's data.json. The
// JSON names are the column names. Nullable columns are pointers; IDs are the
// ones of the exporting device until the import maps them.

type ImportProject struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	CreatedAt   string  `json:"created_at"`
	ArchivedAt  *string `json:"archived_at"`
}

type ImportPlan struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	TeamID      *string `json:"team_id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	CreatedAt   string  `json:"created_at"`
	ArchivedAt  *string `json:"archived_at"`
}

type ImportStatus struct {
	ID          string  `json:"id"`
	PlanID      string  `json:"plan_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Color       string  `json:"color"`
	Category    string  `json:"category"`
	Position    string  `json:"position"`
}

type ImportTask struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	PlanID      string  `json:"plan_id"`
	StatusID    string  `json:"status_id"`
	ParentID    *string `json:"parent_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Position    string  `json:"position"`
	DueDate     *string `json:"due_date"`
	CreatedAt   string  `json:"created_at"`
	CompletedAt *string `json:"completed_at"`
}

type ImportAssignee struct {
	TaskID     string `json:"task_id"`
	AssignedAt string `json:"assigned_at"`
}

type ImportComment struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ImportResource struct {
	ID        string  `json:"id"`
	ParentID  *string `json:"parent_id"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	MimeType  *string `json:"mime_type"`
	Revision  int64   `json:"revision"`
	CreatedAt string  `json:"created_at"`
	// Filled in by the import from the archive's bytes.
	Size   int64 `json:"-"`
	IsText bool  `json:"-"`
}

type ImportTaskResource struct {
	TaskID     string `json:"task_id"`
	ResourceID string `json:"resource_id"`
	LinkedAt   string `json:"linked_at"`
}

type ImportEvent struct {
	ID         string  `json:"id"`
	ResourceID string  `json:"resource_id"`
	Action     string  `json:"action"`
	TaskID     *string `json:"task_id"`
	Details    string  `json:"details"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type ImportSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RecentComment is a comment with what is needed to point at its task.
type RecentComment struct {
	Comment
	TaskTitle string
	ProjectID string
	PlanID    string
	// ParentID is the top-level task of a subtask's comment; empty otherwise.
	ParentID string
}
