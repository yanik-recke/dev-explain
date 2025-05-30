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
	"github.com/yanik-recke/devexplain/internal/parser"
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


// Tries to retrieve the Github repository of 
// the passed url and embed all the commits.
// Instead of embedding the whole commit diff, each file
// will be embedded separately so that a query only returns relevant
// files
func (r *RepoService) IndexRepo(ctx context.Context, url string) error {

	author, repoName, err := parser.ParseGitHubURL(url)
	if err != nil {
		return fmt.Errorf("error while trying to parse url: %w", err)
	}

	// Check if repository exists
	repo, res, err := r.githubClient.Repositories.Get(ctx, author, repoName)
	if err != nil {
		log.Println(res.StatusCode)
		return fmt.Errorf("erorr while trying to get repo: %w", err)
	}

	// Get commits
	commits, res, err := r.githubClient.Repositories.ListCommits(ctx, author, repoName, nil)

	if err != nil {
		log.Println(res.StatusCode)
		return fmt.Errorf("error while trying to find commits: %w", err)
	}
	
	// Create embedding function for collection
	ef, err := ollama.NewOllamaEmbeddingFunction(ollama.WithBaseURL(r.ollamaBaseUrl), ollama.WithModel(embeddings.EmbeddingModel(r.embedModel)))

	if  err != nil {
		return fmt.Errorf("error while trying to create embedding function: %w", err)
	}

	// Get collection or create it, if it does not exist yet
	collection, err := r.dbClient.GetOrCreateCollection(
			ctx, 
			strconv.FormatInt(repo.GetID(), 10), 
			chromago.WithEmbeddingFunctionCreate(ef),
		)

	if  err != nil {
		return fmt.Errorf("error while trying to get collection: %w", err)
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
			chromago.NewStringAttribute("commit", commit.Commit.GetSHA()))

		log.Println("Created metadata")
		for j := range commit.Files {
			err = collection.Add(
				ctx,
				chromago.WithTexts(*commit.Files[j].Patch), 
				chromago.WithIDGenerator(chromago.NewUUIDGenerator()),
				chromago.WithMetadatas(metadata))

			if err != nil {
				log.Println(err)
				log.Printf("error adding commit with sha: %s", *commits[i].SHA)
			}
		}

	}

	return nil
}
