package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	MorphoGraphQLURL = "https://api.morpho.org/graphql"
)

// ── CLIENT ───────────────────────────────────────────────────────────────────

// Query reste inchangé, c'est une implémentation générique solide.
func Query(ctx context.Context, query string, out any) error {
	body, err := json.Marshal(struct {
		Query string `json:"query"`
	}{Query: query})
	if err != nil {
		return fmt.Errorf("marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, MorphoGraphQLURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []GraphQLError  `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return envelope.Errors[0]
	}

	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}

type GraphQLError struct {
	Message string `json:"message"`
}

func (e GraphQLError) Error() string { return "graphql: " + e.Message }
