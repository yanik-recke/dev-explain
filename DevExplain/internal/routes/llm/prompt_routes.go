package prompts

import (
	"net/http"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/go-chi/chi/v5"
	"github.com/yanik-recke/devexplain/internal/service"
)

func PromptRoutes(prefix string, router chi.Router, client chromago.Client, s *service.LlmService) {
	handler := NewPromptHandler(s)

	router.Route(prefix, func(router chi.Router) {
		router.Options("/prompt", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		})
		// POST route to receive prompt
		router.Post("/prompt", handler.GetResponseToPrompt)
	})

}