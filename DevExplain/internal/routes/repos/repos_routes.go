package repos

import (
	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/go-chi/chi/v5"
	"github.com/yanik-recke/devexplain/internal/service"
)

func RepoRoutes(prefix string, router chi.Router, client chromago.Client, s *service.RepoService) {
	handler := NewRepoHandler(s)

	router.Route(prefix, func(router chi.Router) {
		router.Post("/repo", handler.FetchRepo)
	})
}