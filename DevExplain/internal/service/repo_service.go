package service

import (
	"context"
	"fmt"
	"log"
	"strconv"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	ollama "github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
	"github.com/google/go-github/v72/github"
)


type RepoService struct {
	dbClient chromago.Client
	githubClient *github.Client
	ollamaBaseUrl string
	embedModel string
}


func NewRepoService(dbClient chromago.Client, ollamaBaseUrl string, embedModel string, token string) *RepoService {
	client := github.NewClient(nil).WithAuthToken(token)
	return &RepoService{
		dbClient,
		client,
		ollamaBaseUrl,
		embedModel,
	}
}

func (r *RepoService) IndexRepo(ctx context.Context, url string) error {
	// TODO parse url

	// Check if repository exists
	repo, res, err := r.githubClient.Repositories.Get(ctx, "yanik-recke", url)
	if err != nil {
		log.Println(res.StatusCode)
		return fmt.Errorf("erorr while trying to get repo: %w", err)
	}

	commits, res, err := r.githubClient.Repositories.ListCommits(ctx, "yanik-recke", url, nil)

	if err != nil {
		log.Println(res.StatusCode)
		return fmt.Errorf("error while trying to find commits: %w", err)
	}
	
	ef, err := ollama.NewOllamaEmbeddingFunction(ollama.WithBaseURL(r.ollamaBaseUrl), ollama.WithModel(embeddings.EmbeddingModel(r.embedModel)))

	if  err != nil {
		return fmt.Errorf("error while trying to create embedding function: %w", err)
	}

	collection, err := r.dbClient.GetOrCreateCollection(
			ctx, 
			strconv.FormatInt(repo.GetID(), 10), 
			chromago.WithEmbeddingFunctionCreate(ef),
		)

	if  err != nil {
		return fmt.Errorf("error while trying to get collection: %w", err)
	}

	for i := range commits {
		commit, res, err := r.githubClient.Repositories.GetCommit(ctx, "yanik-recke", url, *commits[i].SHA, nil)
		
		if err != nil {
			log.Printf("error retrieving commit with sha: %s - github responded with code: %d", *commits[i].SHA, res.StatusCode)
			continue
		}

		log.Println("Creating metadata")
		log.Println(*commit.Commit.GetAuthor().Name)
		// Create metadata
		metadata := chromago.NewDocumentMetadata(chromago.NewStringAttribute("author", *commit.Commit.GetAuthor().Name), chromago.NewStringAttribute("commit_message", commit.Commit.GetMessage()))

		var diffs string

		log.Println("Created metadata")
		for j := range commit.Files {
			diffs += "Filename: " + *commit.Files[j].Filename + "\nContent:"  + *commit.Files[j].Patch + "\n"
		}

		err = collection.Add(ctx, 
			chromago.WithTexts(diffs), 
			chromago.WithIDGenerator(chromago.NewUUIDGenerator()),
			chromago.WithMetadatas(metadata))

		if err != nil {
			log.Println(err)
			log.Printf("error adding commit with sha: %s", *commits[i].SHA)
		}
	}

	return nil
}