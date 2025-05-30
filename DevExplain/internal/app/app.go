package application

import (
	"context"
	"fmt"
	"log"
	"net/http"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	prompts "github.com/yanik-recke/devexplain/internal/routes/llm"
	"github.com/yanik-recke/devexplain/internal/routes/repos"
	"github.com/yanik-recke/devexplain/internal/service"
)

type App struct {
	router http.Handler
	client chromago.Client
}


func New(ollamaUrl, chatUrl, embedModel, chatModel, token string) *App {

	router := chi.NewRouter()

	router.Use(middleware.Logger)

	client, err := chromago.NewHTTPClient()

	if err != nil {
		log.Fatalf("Error initialzing DB client")
	}	

	repoService := *service.NewRepoService(client, ollamaUrl, embedModel, token)
	llmService := *service.NewLlmService(client, ollamaUrl, chatUrl , embedModel, chatModel)

	prompts.PromptRoutes("/api/ai/", router, client, &llmService)
	repos.RepoRoutes("/api/repos/", router, client, &repoService)

	app := &App {
		router,
		client,
	}

	return app
}

// Starts the server
// Parameter:
//		ctx: TODO
func (a *App) Start(ctx context.Context) error {
	server := &http.Server { 
		Addr: ":3001",
		Handler: a.router,
	}

	err := server.ListenAndServe()

	if err != nil {
		return fmt.Errorf("failed upon startup: %w", err)
	}

	return nil
}