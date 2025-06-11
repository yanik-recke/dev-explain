package application

import (
	"context"
	"fmt"
	"log"
	"net/http"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
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

func New(ollamaUrl, chatUrl, embedModel, chatModel, token, intentUrl, healthUrl string) *App {

	router := chi.NewRouter()

	router.Use(middleware.Logger)

	client := initDbClient("root", "devexplain", ollamaUrl, embedModel)

	repoService := *service.NewRepoService(client, ollamaUrl, embedModel, token)
	intentService := *service.NewIntentService(intentUrl, healthUrl)
	llmService := *service.NewLlmService(client, ollamaUrl, chatUrl, embedModel, chatModel, token, &intentService)

	prompts.PromptRoutes("/api/ai/", router, client, &llmService)
	repos.RepoRoutes("/api/repos/", router, client, &repoService)

	app := &App{
		router,
		client,
	}

	return app
}

// Start
// Starts the server
// Parameter:
//
//	ctx: TODO
func (a *App) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    ":3001",
		Handler: a.router,
	}

	err := server.ListenAndServe()

	if err != nil {
		return fmt.Errorf("failed upon startup: %w", err)
	}

	return nil
}

func initDbClient(tenantName, dbName, ollamaBaseUrl, embedModel string) chromago.Client {
	client, err := chromago.NewHTTPClient()

	if err != nil {
		log.Fatalf("Error initialzing DB client")
	}

	// Trying to get tenant, if does not exist, try to create one
	tenant, err := client.GetTenant(context.TODO(), chromago.NewTenant(tenantName))

	if err != nil {
		log.Printf("Error getting tenant, trying to create tenant")

		tenant, err = client.CreateTenant(context.TODO(), chromago.NewTenant(tenantName))

		if err != nil {
			log.Fatalf("could not get or create tenant")
		}
	}

	client.UseTenant(context.TODO(), tenant)

	db, err := client.GetDatabase(context.TODO(), chromago.NewDatabase(dbName, tenant))

	if err != nil {
		log.Print("Error getting db, trying to create db")

		db, err = client.CreateDatabase(context.TODO(), chromago.NewDatabase(dbName, tenant))

		if err != nil {
			log.Fatalf("could not get or create database")
		}
	}

	client.UseDatabase(context.TODO(), db)

	// Create embedding function for collection
	ef, err := ollama.NewOllamaEmbeddingFunction(ollama.WithBaseURL(ollamaBaseUrl), ollama.WithModel(embeddings.EmbeddingModel(embedModel)))

	if err != nil {
		log.Fatalf("Error trying to create embedding function for repo collection: %w", err)
	}

	// Get or create collection
	_, err = client.GetOrCreateCollection(context.TODO(), "repos", chromago.WithEmbeddingFunctionCreate(ef))

	if err != nil {
		log.Fatalf("Error trying to get or create collection for repos: %w", err)
	}

	return client
}
