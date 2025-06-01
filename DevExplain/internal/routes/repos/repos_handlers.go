package repos

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/yanik-recke/devexplain/internal/service"
)

type FetchRepoRequest struct {
	Url string `json:"url"`
}

type FetchRepoResponse struct {
	Id string `json:"id"`
}

type RepoHandler struct {
	repoService *service.RepoService
}

func NewRepoHandler(repoService *service.RepoService) *RepoHandler {
	return &RepoHandler{
		repoService,
	}
}

func (s *RepoHandler) FetchRepo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	decoder := json.NewDecoder(r.Body)
	var f FetchRepoRequest
	err := decoder.Decode(&f)

	if err != nil || f.Url == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := s.repoService.IndexRepo(context.TODO(), f.Url);

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	response := map[string]string {
		"id" : id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}