package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/v72/github"
	"github.com/yanik-recke/devexplain/internal/parser"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	ollama "github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
)

const LLMContext = `You are a technical expert that has to explain codebase changes 
to users that might not be familiar with the technology used in the code base.
Unless instructed otherwise, focus on the effects of the changes and not 
their technical implementations.

As data to base your response on you will receive the author name, the commit message
and the file names with their corresponding differences.

A question will follow that you need to answer. Do not repeat these instructions.
`

type GeneratedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GeneratedResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type CommitData struct {
	Filename string
	Diff     string
	Message  string
}

type LlmService struct {
	client        chromago.Client
	ollamaBaseUrl string
	genUrl        string
	embedModel    string
	convModel     string
	intentService *IntentService
	githubClient  *github.Client
}

func NewLlmService(client chromago.Client, ollamaBaseUrl, genUrl, embedModel, convModel, token string, intentService *IntentService) *LlmService {
	githubClient := github.NewClient(nil).WithAuthToken(token)
	return &LlmService{
		client,
		ollamaBaseUrl,
		genUrl,
		embedModel,
		convModel,
		intentService,
		githubClient,
	}
}

// Vectorizes input and does semantic search
// on vector store
func (l *LlmService) DoSemanticSearch(ctx context.Context, prompt string, repoId string) (string, error) {
	log.Println("in 'doSemanticSearch'")

	isSpecific, err := l.intentService.intentIsSpecificSHA(prompt)

	if err != nil {
		log.Println("error while trying to determine if prompt asks for specific commit SHA: " + err.Error())
		isSpecific = false
	}

	if isSpecific {
		log.Println("user seems to have asked for specific commit")
		data, err := l.getSpecificCommit(ctx, prompt, repoId)

		if err != nil {
			return "", fmt.Errorf("error while trying to get specific commit by SHA: %w", err)
		}

		if data != "" {
			response, err := l.generateResponse(ctx, prompt, data)

			if err != nil {
				return "", fmt.Errorf("failed to generate answer: %w", err)
			}

			return response, nil
		}
	}

	log.Println("user seems to not have asked for a specific commit, or getting the commit failed")
	// If we get here, then either isSpecific is false or an error occurred
	// while trying to get the specific commit by SHA
	collection, err := l.client.GetCollection(ctx, repoId,
		chromago.WithEmbeddingFunctionGet(embeddings.NewConsistentHashEmbeddingFunction()))

	if err != nil {
		return "", fmt.Errorf("failed to get collection: %w", err)
	}

	log.Println("got collection")

	embedded, err := l.embed(ctx, prompt)

	if err != nil {
		return "", fmt.Errorf("failed to embed input string: %w", err)
	}

	queryResults, err := collection.Query(
		ctx,
		chromago.WithQueryEmbeddings(embedded),
		chromago.WithIncludeQuery(chromago.IncludeMetadatas, chromago.IncludeDocuments),
		chromago.WithNResults(5),
	)

	if err != nil {
		return "", fmt.Errorf("failed to query: %w", err)
	}

	log.Println("queried db")

	var data string
	// range queryResults.CountGroups()
	for i := range 1 {
		data += "Relevant commit: " + strconv.Itoa(i) + ":\n"

		diffs := queryResults.GetDocumentsGroups()[i]
		for j := range diffs {
			metadata := queryResults.GetMetadatasGroups()[i]
			message, success := metadata[j].GetString("commit_message")

			if success {
				data += "Commit message: " + message + "\n"
			} else {
				log.Println("error while trying to retrieve commit message from commits")
			}

			author, success := metadata[j].GetString("author")

			if success {
				data += "Author: " + author + "\n"
			} else {
				log.Println("error while trying to retrieve author from commits")
			}

			commit, success := metadata[j].GetString("commit")

			if success {
				data += "Commit: " + commit + "\n"
			} else {
				log.Println("error while trying to retrieve author from commits")
			}

			data += "Diffs:\n"
			data += diffs[j].ContentString() + "\n"

			data += "-------------\n"
		}

	}

	// log.Println(data)
	response, err := l.generateResponse(ctx, prompt, data)

	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	return response, nil
}

func (l *LlmService) getSpecificCommit(ctx context.Context, prompt, id string) (string, error) {

	sha := parser.ParseSHA(prompt)
	if sha == "" {
		return "", fmt.Errorf("failed to parse / find commit SHA")
	}

	log.Printf("got commit %s: ", sha)

	// Convert string to int64 (base 10, 64-bit)
	i64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to convert id to i64: %w", err)
	}

	repo, res, err := l.githubClient.Repositories.GetByID(ctx, i64)

	if err != nil {
		return "", fmt.Errorf("failed to get repo by id: %w, github res code: %d", err, res.StatusCode)
	}

	log.Printf("got repo by id, owner: %s, name: %s, looking for sha: %s", repo.GetOwner().GetLogin(), repo.GetName(), sha)

	commit, res, err := l.githubClient.Repositories.GetCommit(ctx, repo.GetOwner().GetLogin(), repo.GetName(), sha, nil)

	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w with response code: %d", err, res.StatusCode)
	}

	log.Println("got commit")

	response := "Author: " + *commit.GetAuthor().Name + "\n"
	response += "Message: " + commit.GetCommit().GetMessage() + "\n"
	response += "Diffs:\n"

	log.Println("prepared message")

	for i := range commit.Files {
		response += commit.Files[i].GetPatch() + "\n"
	}

	log.Printf("created response: %s", response)

	return response, nil
}

func (l *LlmService) embed(ctx context.Context, prompt string) (embeddings.Embedding, error) {

	ef, err := ollama.NewOllamaEmbeddingFunction(ollama.WithBaseURL(l.ollamaBaseUrl), ollama.WithModel(embeddings.EmbeddingModel(l.embedModel)))
	if err != nil {
		fmt.Printf("Error creating Ollama embedding function: %s \n", err)
	}

	result, err := ef.EmbedQuery(ctx, prompt)

	if err != nil {
		fmt.Printf("Error during embedding: %s \n", err)
	}

	return result, nil
}

func (l *LlmService) generateResponse(ctx context.Context, prompt string, data string) (string, error) {
	enhancedPrompt := LLMContext + "Using this information: " + data + "\nPlease reply to this prompt:\n" + prompt
	log.Println(enhancedPrompt)
	reqBody, err := json.Marshal(GeneratedRequest{
		Model:  l.convModel,
		Prompt: enhancedPrompt,
		Stream: false,
	})

	if err != nil {
		return "", fmt.Errorf("failed to marshal: %w", err)
	}

	log.Println("marshalling succeeded")

	req, err := http.NewRequestWithContext(ctx, "POST", l.genUrl, bytes.NewBuffer(reqBody))

	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	log.Println("created request")

	// Responses may take some time
	client := &http.Client{Timeout: 120 * time.Second}
	res, err := client.Do(req)

	if err != nil {
		return "", fmt.Errorf("failed to request: %w", err)
	}

	log.Println("sent req and received response")

	// Defer close
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("ollama API returned status %d: %s", res.StatusCode, string(bodyBytes))
	}

	var genResponse GeneratedResponse
	if err := json.NewDecoder(res.Body).Decode(&genResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	log.Println("decoded response")

	// Prune <think> tag
	cleanedResponse := regexp.MustCompile(`(?s)<think>.*?</think>\s*`).ReplaceAllString(genResponse.Response, "")
	// Remove leading and trailing white spaces
	cleanedResponse = regexp.MustCompile(`^\s*|\s*$`).ReplaceAllString(cleanedResponse, "")

	return cleanedResponse, nil
}
