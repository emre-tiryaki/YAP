package service

import "github.com/emre-tiryaki/YAP/backend/internal/repository"

func labelRepo() *repository.LabelRepository { return repository.NewLabelRepository() }

func (e accountEnv) projectsRepo() *repository.ProjectRepository {
	return repository.NewProjectRepository()
}
