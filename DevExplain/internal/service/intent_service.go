package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type IntentService struct {
	intentUrl string
	healthUrl string
}

type IntentRequest struct {
	Prompt string `json:"prompt"`
}

type IntentResponse struct {
	Intent string `json:"intent"`
}

func NewIntentService(intentUrl, healthUrl string) *IntentService {
	return &IntentService{
		intentUrl,
		healthUrl,
	}
}

func (i *IntentService) intentIsSpecificSHA(prompt string) (bool, error) {
	params := url.Values{}
	params.Add("prompt", prompt)
	resp, err := http.Post(fmt.Sprintf("%s?%s", i.intentUrl, params.Encode()),
		"application/json",
		nil)

	if err != nil {
		return false, fmt.Errorf("error calling intent service: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return false, fmt.Errorf("error reading response body: %w", err)
	}

	var response IntentResponse
	err = json.Unmarshal(body, &response)

	if err != nil {
		return false, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return response.Intent == "SHA_REQUEST", nil
}
