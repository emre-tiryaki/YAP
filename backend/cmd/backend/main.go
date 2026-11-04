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

	session, err := service.NewSessionService(db, sessionRepo).Bootstrap(ctx, cfg.DisplayName)
	if err != nil {
		log.Fatalf("Session error: %v", err)
	}

	projectSvc := service.NewProjectService(db, projectRepo)
	taskSvc := service.NewTaskService(db, projectRepo, taskRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/hello-world", handler.HelloWorld)
	handler.NewWorkspaceHandler(session).Register(mux)
	handler.NewProjectHandler(session, projectSvc).Register(mux)
	handler.NewTaskHandler(session, taskSvc).Register(mux)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://localhost%s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
