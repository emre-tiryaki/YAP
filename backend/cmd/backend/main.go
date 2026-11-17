package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/config"
	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/handler"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
	"github.com/emre-tiryaki/YAP/backend/internal/storage"
)

func main() {
	log.Println("Server is starting...")
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := database.New(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer db.Close()

	log.Println("Database has started!")

	sessionRepo := repository.NewSessionRepository()
	projectRepo := repository.NewProjectRepository()
	taskRepo := repository.NewTaskRepository()
	commentRepo := repository.NewCommentRepository()

	session, err := service.NewSessionService(db, sessionRepo).Bootstrap(ctx, cfg.DisplayName)
	if err != nil {
		log.Fatalf("Session error: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository()
	projectSvc := service.NewProjectService(db, projectRepo, settingsRepo)
	taskSvc := service.NewTaskService(db, projectRepo, taskRepo)
	commentSvc := service.NewCommentService(db, taskRepo, commentRepo)
	workspaceSvc := service.NewWorkspaceService(db, repository.NewWorkspaceRepository())

	store, err := storage.NewFS(cfg.ResourcesPath)
	if err != nil {
		log.Fatalf("Resource storage error: %v", err)
	}
	resourceRepo := repository.NewResourceRepository()
	resourceSvc := service.NewResourceService(db, resourceRepo, taskRepo, store)
	settingsSvc := service.NewSettingsService(db, settingsRepo)
	accountSvc := service.NewAccountService(db, repository.NewAccountRepository(), resourceRepo,
		repository.NewWorkspaceRepository(), settingsRepo, store, cfg.DisplayName)

	mux := http.NewServeMux()
	mux.HandleFunc("/hello-world", handler.HelloWorld)
	handler.NewWorkspaceHandler(session, workspaceSvc).Register(mux)
	handler.NewProjectHandler(session, projectSvc).Register(mux)
	handler.NewTaskHandler(session, taskSvc).Register(mux)
	handler.NewCommentHandler(session, commentSvc).Register(mux)
	handler.NewResourceHandler(session, resourceSvc).Register(mux)
	handler.NewSettingsHandler(session, settingsSvc).Register(mux)
	handler.NewAccountHandler(session, accountSvc).Register(mux)
	handler.NewSearchHandler(session, service.NewSearchService(db, repository.NewSearchRepository())).Register(mux)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://localhost%s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
