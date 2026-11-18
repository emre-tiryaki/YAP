package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	// MaxRepoArchiveBytes bounds a downloaded source archive.
	MaxRepoArchiveBytes = 1 << 30

	// repoCheckInterval is how often an up-to-date copy is checked against
	// GitHub while automatic updates are on.
	repoCheckInterval = 15 * time.Minute
	// repoRetryInterval is how soon a copy that could not be fetched (or
	// checked because GitHub was unreachable) is tried again.
	repoRetryInterval = time.Minute
	// repoTickInterval is how often the background loop looks for due work.
	repoTickInterval = 30 * time.Second
	// repoSyncTimeout bounds one fetch or update.
	repoSyncTimeout = 30 * time.Minute
	// repoTextSniffBytes are read to decide whether a repository file is
	// text.
	repoTextSniffBytes = 8 << 10
)

type RepoRepository interface {
	Insert(ctx context.Context, q database.DBTX, r model.Repository, now string) error
	Get(ctx context.Context, q database.DBTX, workspaceID, resourceID string) (model.Repository, error)
	List(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Repository, error)
	UpdateInfo(ctx context.Context, q database.DBTX, workspaceID, resourceID, fullName, htmlURL string, private bool, now string) error
	SaveCheckout(ctx context.Context, q database.DBTX, r model.Repository, now string) error
	InsertClone(ctx context.Context, q database.DBTX, c model.RepositoryClone, now string) error
	ListClones(ctx context.Context, q database.DBTX, resourceID string) ([]model.RepositoryClone, error)
	ForgetClone(ctx context.Context, q database.DBTX, resourceID, id, now string) (bool, error)
}

// RepoStore is the part of the file store that holds repository copies.
type RepoStore interface {
	Path(path []string) (string, error)
	TempDir() (string, error)
	TempFile() (*os.File, error)
	InstallDir(src string, path []string) error
	Remove(path []string) error
	DirNames(path []string) ([]string, error)
}

// RepoFetcher unpacks the copies the app shows and clones repositories into
// folders the user picks.
type RepoFetcher interface {
	GitAvailable() bool
	Unzip(ctx context.Context, src io.ReaderAt, size int64, dst string) error
	Clone(ctx context.Context, cloneURL, token, dst string) error
}

type RepositoryService struct {
	db        *sql.DB
	repos     RepoRepository
	resources *ResourceService
	github    *GitHubService
	settings  SettingsRepository
	store     RepoStore
	fetcher   RepoFetcher
	now       func() time.Time

	mu sync.Mutex
	// busy holds the repositories being fetched right now.
	busy map[string]bool
	// background holds the base context of fetches started by requests, so
	// they outlive the request; Run replaces it with its own.
	background context.Context
	wg         sync.WaitGroup
}

func NewRepositoryService(db *sql.DB, repos RepoRepository, resources *ResourceService, github *GitHubService,
	settings SettingsRepository, store RepoStore, fetcher RepoFetcher) *RepositoryService {
	return &RepositoryService{
		db: db, repos: repos, resources: resources, github: github, settings: settings, store: store,
		fetcher: fetcher, now: time.Now, busy: map[string]bool{}, background: context.Background(),
	}
}

func (s *RepositoryService) nowString() string {
	return s.now().UTC().Format(time.RFC3339)
}

func (s *RepositoryService) copyPath(workspaceID, resourceID string) []string {
	return []string{workspaceID, repoDirName, resourceID}
}

type AddRepositoryInput struct {
	// ParentID is the folder the repository goes into; "" is the top.
	ParentID string
	// FullName is "owner/name".
	FullName string
	// Name is the folder's name; empty uses the repository's name.
	Name       string
	OnConflict OnConflict
}

// Add puts a repository into the resource tree as a read-only folder and
// starts fetching it (as the ZIP of the newest commit). The connected
// account is needed to find the repository.
func (s *RepositoryService) Add(ctx context.Context, session model.Session, in AddRepositoryInput) (model.Resource, error) {
	if in.OnConflict != ConflictFail && in.OnConflict != ConflictKeepBoth {
		return model.Resource{}, fmt.Errorf("%w: on_conflict", ErrInvalidInput)
	}
	fullName := strings.TrimSpace(in.FullName)
	if fullName == "" {
		return model.Resource{}, fmt.Errorf("%w: repository is required", ErrInvalidInput)
	}
	token, err := s.github.token(session)
	if err != nil {
		return model.Resource{}, err
	}
	info, err := s.github.api.Repo(ctx, token, fullName)
	if err != nil {
		return model.Resource{}, s.github.apiError(session, token, err)
	}
	if !validBranch(info.DefaultBranch) {
		return model.Resource{}, fmt.Errorf("%w: repository has no usable default branch", ErrConflict)
	}
	name := in.Name
	if strings.TrimSpace(name) == "" {
		_, name, _ = strings.Cut(info.FullName, "/")
	}

	var created model.Resource
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		r, err := s.resources.createFolder(ctx, tx, session, in.ParentID, name, in.OnConflict,
			map[string]any{"repository": info.FullName})
		if err != nil {
			return err
		}
		repo := model.Repository{
			ResourceID: r.ID, WorkspaceID: session.WorkspaceID, GitHubID: info.ID, FullName: info.FullName,
			HTMLURL: info.HTMLURL, Branch: info.DefaultBranch, Private: info.Private,
			AddedBy: session.MembershipID,
		}
		if err := s.repos.Insert(ctx, tx, repo, s.nowString()); err != nil {
			return err
		}
		created, err = s.resources.repo.Get(ctx, tx, session.WorkspaceID, r.ID)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	s.start(session, created.ID)
	return created, nil
}

// validBranch rejects names git would read as options or that are not
// plain ref names.
func validBranch(b string) bool {
	if b == "" || len(b) > 255 || strings.HasPrefix(b, "-") || strings.HasPrefix(b, "/") ||
		strings.HasSuffix(b, "/") || strings.HasSuffix(b, ".lock") || strings.Contains(b, "..") ||
		strings.Contains(b, "//") || strings.Contains(b, "@{") {
		return false
	}
	return strings.IndexFunc(b, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r) || strings.ContainsRune(`~^:?*[\`, r)
	}) < 0
}

// Get returns a repository folder with the state of this device's copy.
func (s *RepositoryService) Get(ctx context.Context, session model.Session, resourceID string) (model.Repository, error) {
	r, err := s.repos.Get(ctx, s.db, session.WorkspaceID, resourceID)
	if err != nil {
		return model.Repository{}, notFound(err, "repository")
	}
	s.mu.Lock()
	if s.busy[resourceID] {
		r.State = model.CheckoutFetching
	}
	s.mu.Unlock()
	return r, nil
}

// splitRepoPath checks a "/"-separated path inside a repository. The empty
// path is the top. A .git folder is never shown (a copy made before clones
// moved out of the app may still have one).
func splitRepoPath(p string) (string, error) {
	p = strings.Trim(p, "/")
	if p == "" {
		return ".", nil
	}
	if !utf8.ValidString(p) || strings.ContainsAny(p, "\\\x00") {
		return "", fmt.Errorf("%w: path", ErrInvalidInput)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", fmt.Errorf("%w: path", ErrInvalidInput)
		}
	}
	if p == ".git" || strings.HasPrefix(p, ".git/") {
		return "", fmt.Errorf("%w: path", ErrNotFound)
	}
	return p, nil
}

// openCopy opens this device's copy of a repository as a root nothing can
// escape, symbolic links included.
func (s *RepositoryService) openCopy(ctx context.Context, session model.Session, resourceID string) (model.Repository, *os.Root, error) {
	repo, err := s.Get(ctx, session, resourceID)
	if err != nil {
		return model.Repository{}, nil, err
	}
	dir, err := s.store.Path(s.copyPath(session.WorkspaceID, resourceID))
	if err != nil {
		return model.Repository{}, nil, err
	}
	root, err := os.OpenRoot(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return model.Repository{}, nil, fmt.Errorf("%w: repository not downloaded yet", ErrFileMissing)
	}
	if err != nil {
		return model.Repository{}, nil, fmt.Errorf("open repository copy: %w", err)
	}
	return repo, root, nil
}

// sniffText reads the start of a file to tell text from binary and pick a
// content type.
func sniffText(f io.Reader, name string, size int64) (string, bool, error) {
	head := make([]byte, repoTextSniffBytes)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", false, err
	}
	head = head[:n]
	isText := size <= MaxTextBytes && !slices.Contains(head, 0)
	if isText {
		// A rune may be cut at the end of the sniffed bytes.
		valid := head
		for i := 0; i < utf8.UTFMax && len(valid) > 0 && !utf8.Valid(valid); i++ {
			valid = valid[:len(valid)-1]
		}
		isText = utf8.Valid(valid) && (len(head)-len(valid) < utf8.UTFMax)
	}
	return detectMime(name, "", head), isText, nil
}

// Entries lists a folder inside a repository's copy: folders first, then
// files, by name. Links and special files are left out.
func (s *RepositoryService) Entries(ctx context.Context, session model.Session, resourceID, dirPath string) ([]model.RepoEntry, error) {
	_, root, err := s.openCopy(ctx, session, resourceID)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	rel, err := splitRepoPath(dirPath)
	if err != nil {
		return nil, err
	}
	dir, err := root.Open(rel)
	if err != nil {
		return nil, fmt.Errorf("%w: folder", ErrNotFound)
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat folder: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: not a folder", ErrInvalidInput)
	}
	items, err := dir.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("read folder: %w", err)
	}
	out := make([]model.RepoEntry, 0, len(items))
	for _, item := range items {
		entryPath := item.Name()
		if rel != "." {
			entryPath = rel + "/" + item.Name()
		}
		if entryPath == ".git" {
			continue
		}
		fi, err := item.Info()
		if err != nil {
			continue // removed while listing
		}
		e := model.RepoEntry{Name: item.Name(), Path: entryPath, ModTime: fi.ModTime().UTC()}
		switch {
		case fi.IsDir():
			e.Kind = model.ResourceFolder
		case fi.Mode().IsRegular():
			e.Kind = model.ResourceFile
			e.Size = fi.Size()
			f, err := root.Open(entryPath)
			if err != nil {
				continue
			}
			e.MimeType, e.IsText, err = sniffText(f, item.Name(), e.Size)
			f.Close()
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", entryPath, err)
			}
		default:
			continue
		}
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b model.RepoEntry) int {
		if a.Kind != b.Kind {
			if a.Kind == model.ResourceFolder {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

// OpenFile returns a file of a repository's copy; the caller closes it.
func (s *RepositoryService) OpenFile(ctx context.Context, session model.Session, resourceID, filePath string) (model.RepoEntry, *os.File, error) {
	_, root, err := s.openCopy(ctx, session, resourceID)
	if err != nil {
		return model.RepoEntry{}, nil, err
	}
	defer root.Close()
	rel, err := splitRepoPath(filePath)
	if err != nil {
		return model.RepoEntry{}, nil, err
	}
	if rel == "." {
		return model.RepoEntry{}, nil, fmt.Errorf("%w: not a file", ErrInvalidInput)
	}
	f, err := root.Open(rel)
	if err != nil {
		return model.RepoEntry{}, nil, fmt.Errorf("%w: file", ErrNotFound)
	}
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		f.Close()
		if err != nil {
			return model.RepoEntry{}, nil, fmt.Errorf("stat file: %w", err)
		}
		return model.RepoEntry{}, nil, fmt.Errorf("%w: not a file", ErrInvalidInput)
	}
	e := model.RepoEntry{Name: path.Base(rel), Path: rel, Kind: model.ResourceFile, Size: fi.Size(), ModTime: fi.ModTime().UTC()}
	e.MimeType, e.IsText, err = sniffText(f, e.Name, e.Size)
	if err == nil {
		_, err = f.Seek(0, io.SeekStart)
	}
	if err != nil {
		f.Close()
		return model.RepoEntry{}, nil, fmt.Errorf("read file: %w", err)
	}
	return e, f, nil
}

// Sync starts fetching the newest commit of a repository now, whatever the
// automatic update setting says. It returns at once; the state shows
// "fetching" until it is done.
func (s *RepositoryService) Sync(ctx context.Context, session model.Session, resourceID string) (model.Repository, error) {
	repo, err := s.Get(ctx, session, resourceID)
	if err != nil {
		return model.Repository{}, err
	}
	if repo.TrashedAt != nil {
		return model.Repository{}, fmt.Errorf("%w: repository is in the trash", ErrNotFound)
	}
	s.start(session, resourceID)
	repo.State = model.CheckoutFetching
	return repo, nil
}

// start fetches a repository in the background unless that is already
// happening.
func (s *RepositoryService) start(session model.Session, resourceID string) {
	s.mu.Lock()
	base := s.background
	s.mu.Unlock()
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.syncOne(base, session, resourceID)
	}()
}

// Wait blocks until the fetches started so far are done (for tests and
// shutdown).
func (s *RepositoryService) Wait() {
	s.wg.Wait()
}

// syncOne fetches one repository if no other fetch of it is running.
func (s *RepositoryService) syncOne(base context.Context, session model.Session, resourceID string) {
	s.mu.Lock()
	if s.busy[resourceID] {
		s.mu.Unlock()
		return
	}
	s.busy[resourceID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.busy, resourceID)
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(base, repoSyncTimeout)
	defer cancel()
	if err := s.sync(ctx, session, resourceID); err != nil && ctx.Err() == nil {
		slog.Error("repository sync failed", "resource_id", resourceID, "error", err)
	}
}

// sync brings this device's copy to the newest commit of the branch. The
// outcome is written to the checkout row; the returned error is only for
// failures the user cannot act on (they are logged).
func (s *RepositoryService) sync(ctx context.Context, session model.Session, resourceID string) error {
	repo, err := s.repos.Get(ctx, s.db, session.WorkspaceID, resourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // purged meanwhile
		}
		return err
	}
	if repo.TrashedAt != nil {
		return nil
	}
	attempted := s.now().UTC()
	repo.AttemptedAt = &attempted

	token, err := s.github.token(session)
	switch {
	case errors.Is(err, ErrGitHubNotConnected):
		token = "" // public repositories work without an account
	case err != nil:
		return s.finish(ctx, repo, model.CheckoutFailed, err)
	}

	info, err := s.github.api.RepoByID(ctx, token, repo.GitHubID)
	if err != nil {
		return s.fail(ctx, session, repo, token, err)
	}
	if info.FullName != repo.FullName || info.HTMLURL != repo.HTMLURL || info.Private != repo.Private {
		if err := s.repos.UpdateInfo(ctx, s.db, session.WorkspaceID, resourceID, info.FullName, info.HTMLURL, info.Private, s.nowString()); err != nil {
			return err
		}
		repo.FullName, repo.HTMLURL, repo.Private = info.FullName, info.HTMLURL, info.Private
	}
	sha, err := s.github.api.BranchHead(ctx, token, repo.FullName, repo.Branch)
	if err != nil {
		return s.fail(ctx, session, repo, token, err)
	}
	checked := s.now().UTC()
	repo.CheckedAt = &checked

	dst := s.copyPath(session.WorkspaceID, resourceID)
	dir, err := s.store.Path(dst)
	if err != nil {
		return err
	}
	_, statErr := os.Stat(dir)
	// A copy still holding .git was a clone kept by an older version; it is
	// replaced by the ZIP.
	if _, gitErr := os.Stat(dir + "/.git"); gitErr == nil {
		statErr = fs.ErrNotExist
	}
	if repo.State == model.CheckoutReady && repo.CommitSHA == sha && statErr == nil {
		return s.finish(ctx, repo, "", nil)
	}

	// The copy is being replaced: say so while it happens.
	fetching := repo
	if fetching.CommitSHA != "" {
		fetching.State = model.CheckoutFetching
	}
	if err := s.repos.SaveCheckout(ctx, s.db, fetching, s.nowString()); err != nil {
		return err
	}

	if err := s.fetchZip(ctx, repo, token, sha, dst); err != nil {
		return s.fail(ctx, session, repo, token, err)
	}
	fetched := s.now().UTC()
	repo.State, repo.CommitSHA, repo.FetchedAt = model.CheckoutReady, sha, &fetched
	return s.finish(ctx, repo, "", nil)
}

func (s *RepositoryService) fetchZip(ctx context.Context, repo model.Repository, token, sha string, dst []string) error {
	body, err := s.github.api.Zipball(ctx, token, repo.FullName, sha)
	if err != nil {
		return err
	}
	defer body.Close()
	tmp, err := s.store.TempFile()
	if err != nil {
		return err
	}
	defer func() {
		tmp.Close()
		_ = os.Remove(tmp.Name()) // the archive is only needed while unpacking
	}()
	n, err := io.Copy(tmp, io.LimitReader(body, MaxRepoArchiveBytes+1))
	if err != nil {
		return fmt.Errorf("%w: download archive: %v", ErrGitHubUnavailable, err)
	}
	if n > MaxRepoArchiveBytes {
		return fmt.Errorf("%w: archive", model.ErrRepositoryTooLarge)
	}
	dir, err := s.store.TempDir()
	if err != nil {
		return err
	}
	if err := s.fetcher.Unzip(ctx, tmp, n, dir); err != nil {
		_ = os.RemoveAll(dir) // unpacking failed; nothing to keep
		return err
	}
	if err := s.store.InstallDir(dir, dst); err != nil {
		_ = os.RemoveAll(dir) // the old copy stays in place
		return err
	}
	return nil
}

// fail records why an attempt failed; the copy on disk, if any, stays.
func (s *RepositoryService) fail(ctx context.Context, session model.Session, repo model.Repository, token string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	code := model.CheckoutFailed
	switch {
	case errors.Is(err, model.ErrGitHubUnavailable):
		code = model.CheckoutOffline
	case errors.Is(err, model.ErrGitHubUnauthorized):
		code = model.CheckoutUnauthorized
		if token != "" {
			s.github.forget(session, token)
		}
	case errors.Is(err, model.ErrGitHubNotFound):
		// Without an account a private repository looks missing.
		code = model.CheckoutNotFound
		if token == "" {
			code = model.CheckoutUnauthorized
		}
	case errors.Is(err, model.ErrRepositoryTooLarge):
		code = model.CheckoutTooLarge
	}
	return s.finish(ctx, repo, code, err)
}

func (s *RepositoryService) finish(ctx context.Context, repo model.Repository, code model.CheckoutError, cause error) error {
	repo.Error = code
	if repo.CommitSHA != "" {
		repo.State = model.CheckoutReady
	} else {
		repo.State = model.CheckoutPending
	}
	if err := s.repos.SaveCheckout(ctx, s.db, repo, s.nowString()); err != nil {
		return err
	}
	if code == model.CheckoutFailed {
		return cause
	}
	return nil
}

// maxCommitPage bounds how far back the history can be paged (5000 commits).
const maxCommitPage = 100

// Commits returns a page (from 1) of the history of the repository's branch,
// newest first, straight from GitHub: it is not stored, so it needs GitHub
// to be reachable.
func (s *RepositoryService) Commits(ctx context.Context, session model.Session, resourceID string, page int) ([]model.GitHubCommit, error) {
	if page < 1 || page > maxCommitPage {
		return nil, fmt.Errorf("%w: page", ErrInvalidInput)
	}
	repo, err := s.Get(ctx, session, resourceID)
	if err != nil {
		return nil, err
	}
	token, err := s.github.token(session)
	switch {
	case errors.Is(err, ErrGitHubNotConnected):
		token = "" // public history needs no account
	case err != nil:
		return nil, err
	}
	commits, err := s.github.api.Commits(ctx, token, repo.FullName, repo.Branch, page)
	if err != nil {
		return nil, s.cloneError(session, token, err)
	}
	return commits, nil
}

type CloneInput struct {
	// ParentPath is the absolute path of an existing folder on this
	// computer.
	ParentPath string
	// Name is the new folder the clone goes into.
	Name string
}

// Clone makes a full git clone of a repository into a new folder the user
// picked. The clone belongs to the user from then on: the app records where
// it is and never changes it.
func (s *RepositoryService) Clone(ctx context.Context, session model.Session, resourceID string, in CloneInput) (model.RepositoryClone, error) {
	if !s.fetcher.GitAvailable() {
		return model.RepositoryClone{}, ErrGitMissing
	}
	parent := in.ParentPath
	if parent == "" || !filepath.IsAbs(parent) {
		return model.RepositoryClone{}, fmt.Errorf("%w: folder must be an absolute path", ErrInvalidInput)
	}
	parent = filepath.Clean(parent)
	name, err := normalizeResourceName(in.Name)
	if err != nil {
		return model.RepositoryClone{}, err
	}
	fi, err := os.Stat(parent)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return model.RepositoryClone{}, fmt.Errorf("%w: folder", ErrNotFound)
	case errors.Is(err, fs.ErrPermission):
		return model.RepositoryClone{}, fmt.Errorf("%w: folder", ErrForbidden)
	case err != nil:
		return model.RepositoryClone{}, fmt.Errorf("stat folder: %w", err)
	case !fi.IsDir():
		return model.RepositoryClone{}, fmt.Errorf("%w: not a folder", ErrInvalidInput)
	}
	target := filepath.Join(parent, name)
	if _, err := os.Lstat(target); err == nil {
		return model.RepositoryClone{}, fmt.Errorf("%w: %s exists", ErrConflict, name)
	}

	repo, err := s.Get(ctx, session, resourceID)
	if err != nil {
		return model.RepositoryClone{}, err
	}
	if repo.TrashedAt != nil {
		return model.RepositoryClone{}, fmt.Errorf("%w: repository is in the trash", ErrNotFound)
	}
	token, err := s.github.token(session)
	switch {
	case errors.Is(err, ErrGitHubNotConnected):
		token = "" // public repositories clone without an account
	case err != nil:
		return model.RepositoryClone{}, err
	}
	info, err := s.github.api.RepoByID(ctx, token, repo.GitHubID)
	if err != nil {
		return model.RepositoryClone{}, s.cloneError(session, token, err)
	}

	cloneCtx, cancel := context.WithTimeout(ctx, repoSyncTimeout)
	defer cancel()
	if err := s.fetcher.Clone(cloneCtx, info.CloneURL, token, target); err != nil {
		// The folder did not exist before; whatever git left there goes.
		_ = os.RemoveAll(target)
		return model.RepositoryClone{}, s.cloneError(session, token, err)
	}
	c := model.RepositoryClone{ID: model.NewID(), ResourceID: resourceID, Path: target, CreatedAt: s.now().UTC().Truncate(time.Second)}
	if err := s.repos.InsertClone(ctx, s.db, c, s.nowString()); err != nil {
		return model.RepositoryClone{}, err
	}
	c.Exists = true
	return c, nil
}

func (s *RepositoryService) cloneError(session model.Session, token string, err error) error {
	if token == "" && (errors.Is(err, model.ErrGitHubNotFound) || errors.Is(err, model.ErrGitHubUnauthorized)) {
		// Without an account a private repository looks missing.
		return fmt.Errorf("%w: sign in to clone this repository", ErrGitHubNotConnected)
	}
	return s.github.apiError(session, token, err)
}

// Clones lists the clones of a repository made on this computer, and
// whether each is still there.
func (s *RepositoryService) Clones(ctx context.Context, session model.Session, resourceID string) ([]model.RepositoryClone, error) {
	if _, err := s.Get(ctx, session, resourceID); err != nil {
		return nil, err
	}
	clones, err := s.repos.ListClones(ctx, s.db, resourceID)
	if err != nil {
		return nil, err
	}
	for i := range clones {
		_, err := os.Stat(filepath.Join(clones[i].Path, ".git"))
		clones[i].Exists = err == nil
	}
	return clones, nil
}

// ForgetClone takes a clone off the list. Its files are left alone.
func (s *RepositoryService) ForgetClone(ctx context.Context, session model.Session, resourceID, cloneID string) error {
	if _, err := s.Get(ctx, session, resourceID); err != nil {
		return err
	}
	found, err := s.repos.ForgetClone(ctx, s.db, resourceID, cloneID, s.nowString())
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: clone", ErrNotFound)
	}
	return nil
}

// LocalFolders lists the subfolders of a folder on this computer, for
// picking where to clone. An empty path is the home folder.
func (s *RepositoryService) LocalFolders(_ context.Context, path string) (model.LocalFolder, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = string(filepath.Separator)
	}
	if path == "" {
		path = home
	}
	if !filepath.IsAbs(path) {
		return model.LocalFolder{}, fmt.Errorf("%w: path must be absolute", ErrInvalidInput)
	}
	path = filepath.Clean(path)
	entries, err := os.ReadDir(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return model.LocalFolder{}, fmt.Errorf("%w: folder", ErrNotFound)
	case errors.Is(err, fs.ErrPermission):
		return model.LocalFolder{}, fmt.Errorf("%w: folder", ErrForbidden)
	case err != nil:
		return model.LocalFolder{}, fmt.Errorf("%w: not a folder", ErrInvalidInput)
	}
	out := model.LocalFolder{Path: path, Home: home, Folders: []string{}}
	if parent := filepath.Dir(path); parent != path {
		out.Parent = parent
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		isDir := e.IsDir()
		if e.Type()&fs.ModeSymlink != 0 {
			fi, err := os.Stat(filepath.Join(path, e.Name()))
			isDir = err == nil && fi.IsDir()
		}
		if isDir {
			out.Folders = append(out.Folders, e.Name())
		}
	}
	slices.SortFunc(out.Folders, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return out, nil
}

// due reports whether the background loop should fetch or check a
// repository now.
func (s *RepositoryService) due(r model.Repository, autoUpdate bool, now time.Time) bool {
	if r.TrashedAt != nil {
		return false
	}
	if r.AttemptedAt == nil {
		return true
	}
	since := now.Sub(*r.AttemptedAt)
	if r.State == model.CheckoutPending {
		return since >= repoRetryInterval
	}
	if !autoUpdate {
		return false
	}
	if r.Error == model.CheckoutOffline {
		return since >= repoRetryInterval
	}
	return since >= repoCheckInterval
}

// Tick does the background work that is due: the first fetch of new
// repositories, and when automatic updates are on, checking the others for
// new commits. Copies of repositories whose folders were purged are removed.
func (s *RepositoryService) Tick(ctx context.Context, session model.Session) error {
	settings, err := loadUserSettings(ctx, s.db, s.settings, session.UserID)
	if err != nil {
		return err
	}
	repos, err := s.repos.List(ctx, s.db, session.WorkspaceID)
	if err != nil {
		return err
	}
	if err := s.collectGarbage(session, repos); err != nil {
		return err
	}
	now := s.now()
	for _, r := range repos {
		if err := ctx.Err(); err != nil {
			return err
		}
		if s.due(r, settings.RepoAutoUpdate, now) {
			s.syncOne(ctx, session, r.ResourceID)
		}
	}
	return nil
}

// collectGarbage removes copies that no repository folder owns any more.
func (s *RepositoryService) collectGarbage(session model.Session, repos []model.Repository) error {
	names, err := s.store.DirNames([]string{session.WorkspaceID, repoDirName})
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(repos))
	for _, r := range repos {
		known[r.ResourceID] = true
	}
	for _, name := range names {
		s.mu.Lock()
		busy := s.busy[name]
		s.mu.Unlock()
		if known[name] || busy {
			continue
		}
		if err := s.store.Remove(s.copyPath(session.WorkspaceID, name)); err != nil {
			return err
		}
	}
	return nil
}

// recover clears "fetching" states left by a fetch that was cut off.
func (s *RepositoryService) recover(ctx context.Context, session model.Session) error {
	repos, err := s.repos.List(ctx, s.db, session.WorkspaceID)
	if err != nil {
		return err
	}
	for _, r := range repos {
		if r.State != model.CheckoutFetching {
			continue
		}
		r.State = model.CheckoutReady
		if r.CommitSHA == "" {
			r.State = model.CheckoutPending
		}
		if err := s.repos.SaveCheckout(ctx, s.db, r, s.nowString()); err != nil {
			return err
		}
	}
	return nil
}

// Run does the background work until ctx ends. Fetches started by requests
// are tied to ctx from now on.
func (s *RepositoryService) Run(ctx context.Context, session model.Session) {
	s.mu.Lock()
	s.background = ctx
	s.mu.Unlock()
	if err := s.recover(ctx, session); err != nil {
		slog.Error("repository recover failed", "error", err)
	}
	ticker := time.NewTicker(repoTickInterval)
	defer ticker.Stop()
	for {
		if err := s.Tick(ctx, session); err != nil && ctx.Err() == nil {
			slog.Error("repository tick failed", "error", err)
		}
		select {
		case <-ctx.Done():
			s.Wait()
			return
		case <-ticker.C:
		}
	}
}
