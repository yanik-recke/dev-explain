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

type RepoHandler struct {
	repoService *service.RepoService
}

func NewRepoHandler(repoService *service.RepoService) *RepoHandler {
	return &RepoHandler{
		repoService,
	}
}

func (s *RepoHandler) FetchRepo(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var f FetchRepoRequest
	err := decoder.Decode(&f)

	if err != nil || f.Url == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err = s.repoService.IndexRepo(context.TODO(), f.Url); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}