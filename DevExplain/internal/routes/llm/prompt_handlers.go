package prompts

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/yanik-recke/devexplain/internal/service"
)


type PromptHandler struct {
	llmService *service.LlmService
}

type PromptRequest struct {
	Prompt string `json:"prompt"`
	RepoId string `json:"repoid"`
}

func NewPromptHandler(llmService *service.LlmService) *PromptHandler {
	return &PromptHandler{
		// Creates its own LLM Service
		llmService,
	}
}

func (s *PromptHandler) GetResponseToPrompt(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var p PromptRequest
    err := decoder.Decode(&p)

	if err != nil || p.Prompt == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	response, err := s.llmService.DoSemanticSearch(context.TODO(), p.Prompt, p.RepoId)

	if err != nil {
		log.Println("Error during the semantic search: %w", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(response))
}