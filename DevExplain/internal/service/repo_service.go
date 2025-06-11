package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	ollama "github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
	"github.com/google/go-github/v72/github"
	"github.com/yanik-recke/devexplain/internal/parser"
)

type Commit struct {
	Sha     string `json:"sha"`
	Message string `json:"message"`
}

type SavedRepo struct {
	Id      string   `json:"id"`
	Name    string   `json:"name"`
	Value   string   `json:"value"`
	Commits []Commit `json:"commits"`
}

type RepoService struct {
	dbClient      chromago.Client
	githubClient  *github.Client
	ollamaBaseUrl string
	embedModel    string
}

func NewRepoService(dbClient chromago.Client, ollamaBaseUrl, embedModel, token string) *RepoService {
	client := github.NewClient(nil).WithAuthToken(token)
	return &RepoService{
		dbClient,
		client,
		ollamaBaseUrl,
		embedModel,
	}
}

// Tries to retrieve the Github repository of
// the passed url and embed all the commits.
// Instead of embedding the whole commit diff, each file
// will be embedded separately so that a query only returns relevant
// files
func (r *RepoService) IndexRepo(ctx context.Context, url string) (string, error) {

	author, repoName, err := parser.ParseGitHubURL(url)
	if err != nil {
		return "", fmt.Errorf("error while trying to parse url: %w", err)
	}

	// Check if repository exists
	repo, res, err := r.githubClient.Repositories.Get(ctx, author, repoName)
	if err != nil {
		log.Println(res.StatusCode)
		return "", fmt.Errorf("erorr while trying to get repo: %w", err)
	}

	// Get commits
	commits, res, err := r.githubClient.Repositories.ListCommits(ctx, author, repoName, nil)

	if err != nil {
		log.Println(res.StatusCode)
		return "", fmt.Errorf("error while trying to find commits: %w", err)
	}

	// Create embedding function for collection
	ef, err := ollama.NewOllamaEmbeddingFunction(ollama.WithBaseURL(r.ollamaBaseUrl), ollama.WithModel(embeddings.EmbeddingModel(r.embedModel)))

	if err != nil {
		return "", fmt.Errorf("error while trying to create embedding function: %w", err)
	}

	if r.isRepoSaved(ctx, strconv.FormatInt(repo.GetID(), 10), ef) {
		// TODO only fetch newest commits, for now just return
		// Need to get most recent commit
		// Fetch all commits from NOW to that commit from github
		// Vectorize only these commits
		return strconv.FormatInt(repo.GetID(), 10), nil
	}

	err = r.saveRepo(ctx, repo, commits, ef)

	if err != nil {
		return "", fmt.Errorf("erorr while trying to save repo: %w", err)
	}

	// Get collection or create it, if it does not exist yet
	collection, err := r.dbClient.GetOrCreateCollection(
		ctx,
		strconv.FormatInt(repo.GetID(), 10),
		chromago.WithEmbeddingFunctionCreate(ef),
	)

	if err != nil {
		return "", fmt.Errorf("error while trying to get collection: %w", err)
	}

	// Create embeddings (implicitly) and store them in vector store
	for i := range commits {
		commit, res, err := r.githubClient.Repositories.GetCommit(ctx, author, repoName, *commits[i].SHA, nil)

		if err != nil {
			log.Printf("error retrieving commit with sha: %s - github responded with code: %d", *commits[i].SHA, res.StatusCode)
			continue
		}

		log.Println("Creating metadata")
		log.Println(*commit.Commit.GetAuthor().Name)

		// Create metadata
		// Metadata includes:
		// author 			- Name of commit author
		// commit_message 	- Message of commit
		// commit 			- SHA of commit
		metadata := chromago.NewDocumentMetadata(
			chromago.NewStringAttribute("author", *commit.Commit.GetAuthor().Name),
			chromago.NewStringAttribute("commit_message", commit.Commit.GetMessage()),
			chromago.NewStringAttribute("commit", commit.GetSHA()))

		log.Println("Created metadata")

		for j := range commit.Files {
			err = collection.Add(
				ctx,
				chromago.WithTexts("Commit "+commit.GetSHA()[:10]+"\n"+commit.Files[j].GetPatch()),
				chromago.WithIDGenerator(chromago.NewUUIDGenerator()),
				chromago.WithMetadatas(metadata))

			if err != nil {
				log.Println(err)
				log.Printf("error adding commit with sha: %s", *commits[i].SHA)
			}
		}

	}

	return strconv.FormatInt(repo.GetID(), 10), nil
}

// Tries to gets saved repositories from database
func (r *RepoService) GetSavedRepos(ctx context.Context) ([]SavedRepo, error) {

	// Create embedding function for collection
	ef, err := ollama.NewOllamaEmbeddingFunction(ollama.WithBaseURL(r.ollamaBaseUrl), ollama.WithModel(embeddings.EmbeddingModel(r.embedModel)))

	if err != nil {
		return nil, fmt.Errorf("error while trying to create embedding function: %w", err)
	}

	collection, err := r.dbClient.GetCollection(ctx, "repos", chromago.WithCollectionNameGet("repos"), chromago.WithEmbeddingFunctionGet(ef))

	if err != nil {
		return nil, fmt.Errorf("error while trying to get collection: %w", err)
	}

	result, err := collection.Query(ctx, chromago.WithQueryTexts("*"))

	if err != nil {
		return nil, fmt.Errorf("error during query of collection: %w", err)
	}

	var repos []SavedRepo
	for i := range result.GetMetadatasGroups()[0] {
		id, err := result.GetMetadatasGroups()[0][i].GetString("id")

		if !err {
			return nil, fmt.Errorf("error while getting id from metadata")
		}

		name, err := result.GetMetadatasGroups()[0][i].GetString("name")
		if !err {
			return nil, fmt.Errorf("error while getting name from metadata")
		}

		value, err := result.GetMetadatasGroups()[0][i].GetString("value")
		if !err {
			return nil, fmt.Errorf("error while getting value from metadata")
		}

		commits, err := result.GetMetadatasGroups()[0][i].GetString("commits")
		if !err {
			return nil, fmt.Errorf("error while getting commits from metadata")
		}

		var repo SavedRepo
		repo.Id = id
		repo.Name = name
		repo.Value = value
		var commitsUnmarshalled []Commit
		// with hopes and prayers
		error := json.Unmarshal([]byte(commits), &commitsUnmarshalled)

		if error != nil {
			return nil, fmt.Errorf("error trying to unmarshal commits: %w", error)
		}

		repo.Commits = commitsUnmarshalled

		repos = append(repos, repo)
	}

	if repos == nil {
		return []SavedRepo{}, nil
	}

	return repos, nil
}

func (r *RepoService) saveRepo(ctx context.Context, repo *github.Repository, commits []*github.RepositoryCommit, ef *ollama.OllamaEmbeddingFunction) error {
	collection, err := r.dbClient.GetCollection(ctx, "repos", chromago.WithEmbeddingFunctionGet(ef))

	if err != nil {
		return fmt.Errorf("error while trying to get collection: %w", err)
	}

	var savedCommits []Commit
	for i := range commits {
		savedCommits = append(savedCommits, Commit{commits[i].GetSHA(), commits[i].GetCommit().GetMessage()})
	}

	jsonString, err := json.Marshal(savedCommits)

	if err != nil {
		return fmt.Errorf("error trying to marshal the commits: %w", err)
	}

	commitsMeta := chromago.NewEmptyMetadata()
	commitsMeta.SetString("commits", string(jsonString))
	commitsMeta.SetString("id", strconv.FormatInt(repo.GetID(), 10))
	commitsMeta.SetString("name", repo.GetName())
	commitsMeta.SetString("value", repo.GetName())

	err = collection.Add(ctx, chromago.WithTexts("placeholder"), chromago.WithIDGenerator(chromago.NewUUIDGenerator()), chromago.WithMetadatas(commitsMeta))

	if err != nil {
		log.Println("Error while trying to add to collection")
		log.Println(err)
	}

	return err
}

func (r *RepoService) isRepoSaved(ctx context.Context, id string, ef *ollama.OllamaEmbeddingFunction) bool {
	_, err := r.dbClient.GetCollection(ctx, id, chromago.WithEmbeddingFunctionGet(ef))

	if err != nil {
		log.Println("potential erro: %w", err.Error())
	}

	return err == nil
}
