package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/config"
	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/github"
	"github.com/emre-tiryaki/YAP/backend/internal/handler"
	"github.com/emre-tiryaki/YAP/backend/internal/llm"
	"github.com/emre-tiryaki/YAP/backend/internal/repofetch"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
	"github.com/emre-tiryaki/YAP/backend/internal/secrets"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
	"github.com/emre-tiryaki/YAP/backend/internal/storage"
)

func main() {
	log.Println("Server is starting...")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
	taskSvc := service.NewTaskService(db, projectRepo, taskRepo, repository.NewTaskActivityRepository())
	commentSvc := service.NewCommentService(db, taskRepo, commentRepo)
	labelSvc := service.NewLabelService(db, projectRepo, repository.NewLabelRepository())
	workspaceSvc := service.NewWorkspaceService(db, repository.NewWorkspaceRepository())

	store, err := storage.NewFS(cfg.ResourcesPath)
	if err != nil {
		log.Fatalf("Resource storage error: %v", err)
	}
	resourceRepo := repository.NewResourceRepository()
	resourceSvc := service.NewResourceService(db, resourceRepo, taskRepo, store)
	settingsSvc := service.NewSettingsService(db, settingsRepo)
	aiRepo := repository.NewAIRepository()
	aiKeys := secrets.NewAIKeyring()
	router := llm.NewRouter()
	aiConnSvc := service.NewAIConnectionService(db, aiRepo, aiKeys, router)
	fetcher := repofetch.NewFetcher(repofetch.NewGit(), repofetch.DefaultLimits)
	githubSvc := service.NewGitHubService(github.New(cfg.GitHubClientID), secrets.NewKeyring(), fetcher.GitAvailable())
	repositorySvc := service.NewRepositoryService(db, repository.NewRepoRepository(), resourceSvc, githubSvc,
		settingsRepo, store, fetcher)
	go repositorySvc.Run(ctx, session)

	activityRepo := repository.NewTaskActivityRepository()
	aiPrompt := service.NewAIPrompt(db, service.AIContextSources{
		Tasks: taskRepo, Activity: activityRepo, Comments: commentRepo, Projects: projectRepo,
		Resources: resourceSvc, Repos: repositorySvc, Placement: aiRepo,
	})
	aiChatSvc := service.NewAIChatService(db, aiRepo, aiConnSvc, aiPrompt, settingsRepo, router)
	if err := aiChatSvc.RecoverInterrupted(ctx); err != nil {
		log.Fatalf("AI recovery error: %v", err)
	}
	aiChatSvc.Start(ctx)

	accountSvc := service.NewAccountService(db, repository.NewAccountRepository(), resourceRepo,
		repository.NewWorkspaceRepository(), settingsRepo, store, cfg.DisplayName).WithAI(aiRepo, aiKeys, aiChatSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/hello-world", handler.HelloWorld)
	handler.NewWorkspaceHandler(session, workspaceSvc).Register(mux)
	handler.NewProjectHandler(session, projectSvc).Register(mux)
	handler.NewTaskHandler(session, taskSvc).Register(mux)
	handler.NewCommentHandler(session, commentSvc).Register(mux)
	handler.NewLabelHandler(session, labelSvc).Register(mux)
	handler.NewResourceHandler(session, resourceSvc).Register(mux)
	handler.NewRepositoryHandler(session, repositorySvc).Register(mux)
	handler.NewGitHubHandler(session, githubSvc).Register(mux)
	handler.NewSettingsHandler(session, settingsSvc).Register(mux)
	handler.NewAIHandler(session, aiChatSvc, aiConnSvc).Register(mux)
	handler.NewAccountHandler(session, accountSvc).Register(mux)
	handler.NewSearchHandler(session, service.NewSearchService(db, repository.NewSearchRepository())).Register(mux)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx) // exiting anyway
	}()

	log.Printf("listening on http://localhost%s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	repositorySvc.Wait()
	aiChatSvc.Wait()
}
