package service

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	maxSearchQueryRunes = 200
	searchLimitPerKind  = 8
)

type SearchRepository interface {
	SearchProjects(ctx context.Context, q database.DBTX, workspaceID, needle string, limit int) ([]model.SearchProject, error)
	SearchPlans(ctx context.Context, q database.DBTX, workspaceID, needle string, limit int) ([]model.SearchPlan, error)
	SearchTasks(ctx context.Context, q database.DBTX, workspaceID, needle string, limit int) ([]model.SearchTask, error)
	SearchResources(ctx context.Context, q database.DBTX, workspaceID, needle string, limit int) ([]model.SearchResource, error)
	TaskByKey(ctx context.Context, q database.DBTX, workspaceID, identifier string, number int) (model.SearchTask, bool, error)
}

type SearchService struct {
	db   *sql.DB
	repo SearchRepository
}

func NewSearchService(db *sql.DB, repo SearchRepository) *SearchService {
	return &SearchService{db: db, repo: repo}
}

// Search finds the live projects, plans, tasks and resources of the session's
// workspace whose text contains query, case-insensitively. A blank query
// matches nothing.
func (s *SearchService) Search(ctx context.Context, session model.Session, query string) (model.SearchResults, error) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > maxSearchQueryRunes {
		return model.SearchResults{}, fmt.Errorf("%w: search query is too long", ErrInvalidInput)
	}
	res := model.SearchResults{
		Projects:  []model.SearchProject{},
		Plans:     []model.SearchPlan{},
		Tasks:     []model.SearchTask{},
		Resources: []model.SearchResource{},
	}
	if query == "" {
		return res, nil
	}
	needle := database.Fold(query)

	// One read transaction so the groups agree with each other.
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		if res.Projects, err = s.repo.SearchProjects(ctx, tx, session.WorkspaceID, needle, searchLimitPerKind); err != nil {
			return err
		}
		if res.Plans, err = s.repo.SearchPlans(ctx, tx, session.WorkspaceID, needle, searchLimitPerKind); err != nil {
			return err
		}
		if res.Tasks, err = s.repo.SearchTasks(ctx, tx, session.WorkspaceID, needle, searchLimitPerKind); err != nil {
			return err
		}
		// A task key such as ABC-12 finds that task first.
		if m := keyPattern.FindStringSubmatch(query); m != nil {
			number, convErr := strconv.Atoi(m[2])
			if convErr == nil {
				task, found, err := s.repo.TaskByKey(ctx, tx, session.WorkspaceID, strings.ToUpper(m[1]), number)
				if err != nil {
					return err
				}
				if found {
					res.Tasks = append([]model.SearchTask{task}, slices.DeleteFunc(res.Tasks, func(t model.SearchTask) bool { return t.ID == task.ID })...)
				}
			}
		}
		res.Resources, err = s.repo.SearchResources(ctx, tx, session.WorkspaceID, needle, searchLimitPerKind)
		return err
	})
	if err != nil {
		return model.SearchResults{}, err
	}
	return res, nil
}
