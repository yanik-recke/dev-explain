package service

import (
	"net/http"
	"testing"
)

func TestIntentUrl(t *testing.T) {
	s := IntentService{intentUrl: "http://localhost:3003/classify",
		healthUrl: "http://localhost:3003/health"}
	resp, err := http.Get(s.healthUrl)
	if err != nil {
		t.Errorf("error calling health endpoint: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("health endpoint returned wrong status code: got %v want %v",
			resp.StatusCode, http.StatusOK)
	}
}

func TestIntentIsSpecificSHA(t *testing.T) {
	s := IntentService{intentUrl: "http://localhost:3003/classify",
		healthUrl: "http://localhost:3003/health"}

	// Example prompt
	prompt := "What happened in commit p4q5r6s7?"

	result, err := s.intentIsSpecificSHA(prompt)

	if err != nil {
		t.Errorf("error calling intent service: %v", err)
	}

	if result != true {
		t.Errorf("intent service returned wrong result")
	}
}
