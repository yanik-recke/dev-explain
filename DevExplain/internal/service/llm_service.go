package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

const LLMContext = 
`You are a technical expert that has to explain codebase changes 
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
	Filename	string
	Diff		string
	Message 	string
}

type LlmService struct {
	client chromago.Client
	ollamaBaseUrl string
	genUrl string
	embedModel string
	convModel string
}

func NewLlmService(client chromago.Client, ollamaBaseUrl string, genUrl string, embedModel string, convModel string) *LlmService {
	return &LlmService{
		client,
		ollamaBaseUrl,
		genUrl,
		embedModel,
		convModel,
	}
}

// Vectorizes input and does semantic search
// in vector store
func (l *LlmService) DoSemanticSearch(ctx context.Context, prompt string, repoName string) (string, error) {
	log.Println("in 'doSemanticSearch'")

	collection, err := l.client.GetCollection(ctx, repoName,
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
		chromago.WithNResults(1),
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
	enhancedPrompt := LLMContext +  "Using this information: " + data + "\nPlease reply to this prompt:\n" + prompt
	log.Println(enhancedPrompt)
	reqBody, err := json.Marshal(GeneratedRequest{
		Model: l.convModel,
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