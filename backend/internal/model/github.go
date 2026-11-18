package model

import (
	"errors"
	"time"
)

// Errors reported by the GitHub client. They live here so the service layer
// can recognize them without importing the client.
var (
	// ErrGitHubUnavailable means GitHub could not be reached (no network,
	// timeouts, 5xx answers or rate limiting).
	ErrGitHubUnavailable = errors.New("github unavailable")
	// ErrGitHubUnauthorized means the token was rejected.
	ErrGitHubUnauthorized = errors.New("github unauthorized")
	// ErrGitHubNotFound means the repository or branch does not exist, or
	// the token cannot see it.
	ErrGitHubNotFound = errors.New("github not found")

	// ErrGitHubClientInvalid means GitHub does not know the OAuth client id.
	ErrGitHubClientInvalid = errors.New("github client id not found")
	// ErrDeviceFlowDisabled means the OAuth app does not allow the device
	// flow ("Enable Device Flow" is off in its settings).
	ErrDeviceFlowDisabled = errors.New("device flow disabled")

	// Device flow answers while waiting for the user.
	ErrDeviceAuthPending = errors.New("authorization pending")
	ErrDeviceSlowDown    = errors.New("slow down")
	ErrDeviceExpired     = errors.New("device code expired")
	ErrDeviceDenied      = errors.New("access denied")

	// ErrRepositoryTooLarge means a repository's files exceed the limits.
	ErrRepositoryTooLarge = errors.New("repository too large")
	// ErrGitMissing means the git program is not installed.
	ErrGitMissing = errors.New("git is not installed")

	// ErrNoCredential means no GitHub token is stored for the user.
	ErrNoCredential = errors.New("no credential")
	// ErrSecretStoreUnavailable means the OS keychain cannot be used.
	ErrSecretStoreUnavailable = errors.New("secret store unavailable")
)

// DeviceCode starts a GitHub device flow: the user enters UserCode at
// VerificationURI while the app polls with DeviceCode.
type DeviceCode struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	ExpiresIn       time.Duration
	Interval        time.Duration
}

type GitHubUser struct {
	ID        int64
	Login     string
	Name      string
	AvatarURL string
}

// GitHubCredential is what is kept in the OS keychain for a user. The token
// never reaches the database, the logs or the browser.
type GitHubCredential struct {
	Token       string     `json:"token"`
	Scopes      string     `json:"scopes"`
	User        GitHubUser `json:"user"`
	ConnectedAt time.Time  `json:"connected_at"`
}

type GitHubRepo struct {
	ID            int64
	FullName      string
	Description   string
	HTMLURL       string
	CloneURL      string
	DefaultBranch string
	Private       bool
	PushedAt      *time.Time
}

type CheckoutState string

const (
	// CheckoutPending has no files yet.
	CheckoutPending CheckoutState = "pending"
	// CheckoutFetching is downloading right now.
	CheckoutFetching CheckoutState = "fetching"
	// CheckoutReady has the files of CommitSHA.
	CheckoutReady CheckoutState = "ready"
)

// CheckoutError is why the last fetch or update attempt failed.
type CheckoutError string

const (
	CheckoutOffline      CheckoutError = "offline"
	CheckoutUnauthorized CheckoutError = "unauthorized"
	CheckoutNotFound     CheckoutError = "not_found"
	CheckoutTooLarge     CheckoutError = "too_large"
	CheckoutFailed       CheckoutError = "failed"
)

// GitHubCommit is one commit of a branch's history, newest first.
type GitHubCommit struct {
	SHA     string
	Parents []string
	Message string
	// AuthorName is the name in the commit; AuthorLogin and AvatarURL are
	// empty when GitHub cannot match it to an account.
	AuthorName  string
	AuthorLogin string
	AvatarURL   string
	Date        time.Time
	HTMLURL     string
}

// Repository is a resource folder filled with a GitHub repository, with the
// state of this device's copy.
type Repository struct {
	ResourceID  string
	WorkspaceID string
	GitHubID    int64
	FullName    string
	HTMLURL     string
	Branch      string
	Private     bool
	AddedBy     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// TrashedAt is the folder's trash time; trashed repositories are not
	// updated.
	TrashedAt *time.Time

	State     CheckoutState
	CommitSHA string
	FetchedAt *time.Time
	CheckedAt *time.Time
	Error     CheckoutError
	// AttemptedAt is the time of the last fetch or update attempt.
	AttemptedAt *time.Time
}

// RepositoryClone is a git clone the user made of a repository into a
// folder of their choice. The app never changes it after cloning.
type RepositoryClone struct {
	ID         string
	ResourceID string
	Path       string
	CreatedAt  time.Time
	// Exists says whether the clone is still on the disk (not stored).
	Exists bool
}

// LocalFolder is a folder on this computer, listed so the user can pick
// where to clone into.
type LocalFolder struct {
	Path string
	// Parent is "" at the root of the file system.
	Parent string
	Home   string
	// Folders are the names of the subfolders, hidden ones left out.
	Folders []string
}

// RepoEntry is a file or folder inside a repository's copy.
type RepoEntry struct {
	Name string
	// Path is "/"-separated from the top of the repository.
	Path     string
	Kind     ResourceKind
	Size     int64
	MimeType string
	IsText   bool
	ModTime  time.Time
}
