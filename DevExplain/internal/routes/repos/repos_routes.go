package repos

import (
	"net/http"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/go-chi/chi/v5"
	"github.com/yanik-recke/devexplain/internal/service"
)

func RepoRoutes(prefix string, router chi.Router, client chromago.Client, s *service.RepoService) {
	handler := NewRepoHandler(s)

	router.Route(prefix, func(router chi.Router) {
		router.Options("/repo", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		})
		router.Post("/repo", handler.FetchRepo)
	})
}