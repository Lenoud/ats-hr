package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/example/ats-platform/internal/search/model"
)

// ErrNotFound is returned when a document is not found
var ErrNotFound = errors.New("document not found")

// SearchFilter defines search filter options
type SearchFilter struct {
	Query         string   `json:"query"`
	Skills        []string `json:"skills"`
	Status        string   `json:"status"`
	Source        string   `json:"source"`
	MinExperience int      `json:"min_experience"`
	MaxExperience int      `json:"max_experience"`
	Page          int      `json:"page"`
	PageSize      int      `json:"page_size"`
}

// SearchResult represents search results
type SearchResult struct {
	Documents []model.ResumeDocument `json:"documents"`
	Total     int64                  `json:"total"`
}

// ESRepository defines the interface for Elasticsearch operations
type ESRepository interface {
	Index(ctx context.Context, doc *model.ResumeDocument) error
	GetByID(ctx context.Context, id string) (*model.ResumeDocument, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, filter SearchFilter) (*SearchResult, error)
	UpdateStatus(ctx context.Context, id string, status string) error
}

// esRepositoryImpl implements ESRepository with real Elasticsearch client
type esRepositoryImpl struct {
	client *elasticsearch.Client
	index  string
}

// NewESRepository creates a new Elasticsearch repository
func NewESRepository(client *elasticsearch.Client, indexName string) ESRepository {
	if indexName == "" {
		indexName = "resumes"
	}
	return &esRepositoryImpl{
		client: client,
		index:  indexName,
	}
}

// Index indexes a document to Elasticsearch
func (r *esRepositoryImpl) Index(ctx context.Context, doc *model.ResumeDocument) error {
	// Ensure index exists
	if err := r.EnsureIndex(ctx); err != nil {
		return fmt.Errorf("failed to ensure index: %w", err)
	}

	// Marshal document to JSON
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	// Create index request
	req := esapi.IndexRequest{
		Index:      r.index,
		DocumentID: doc.DocumentID(),
		Body:       bytes.NewReader(docJSON),
		Refresh:    "true",
	}

	// Execute request
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	return nil
}

// GetByID retrieves a document by ID
func (r *esRepositoryImpl) GetByID(ctx context.Context, id string) (*model.ResumeDocument, error) {
	// Create get request
	req := esapi.GetRequest{
		Index:      r.index,
		DocumentID: id,
	}

	// Execute request
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	// Parse response
	var response map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract source
	source, ok := response["_source"]
	if !ok {
		return nil, fmt.Errorf("no _source in response")
	}

	// Unmarshal document
	var doc model.ResumeDocument
	if err := json.Unmarshal(source, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &doc, nil
}

// Delete deletes a document from the index
func (r *esRepositoryImpl) Delete(ctx context.Context, id string) error {
	// Create delete request
	req := esapi.DeleteRequest{
		Index:      r.index,
		DocumentID: id,
		Refresh:    "true",
	}

	// Execute request
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return ErrNotFound
		}
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	return nil
}

// Search searches documents based on filter
func (r *esRepositoryImpl) Search(ctx context.Context, filter SearchFilter) (*SearchResult, error) {
	// Build query
	query := r.buildSearchQuery(filter)

	// Marshal query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Calculate from offset
	from := 0
	if filter.Page > 0 {
		from = (filter.Page - 1) * filter.PageSize
	}

	// Create search request
	req := esapi.SearchRequest{
		Index: []string{r.index},
		Body:  bytes.NewReader(queryJSON),
		From:  &from,
		Size:  &filter.PageSize,
	}

	// Execute request
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	// Parse response
	var response struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source model.ResumeDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Extract documents
	docs := make([]model.ResumeDocument, len(response.Hits.Hits))
	for i, hit := range response.Hits.Hits {
		docs[i] = hit.Source
	}

	return &SearchResult{
		Documents: docs,
		Total:     response.Hits.Total.Value,
	}, nil
}

// UpdateStatus updates a document's status
func (r *esRepositoryImpl) UpdateStatus(ctx context.Context, id string, status string) error {
	// Build update script
	update := map[string]interface{}{
		"doc": map[string]interface{}{
			"status":     status,
			"updated_at": "now",
		},
	}

	updateJSON, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	// Create update request
	req := esapi.UpdateRequest{
		Index:      r.index,
		DocumentID: id,
		Body:       bytes.NewReader(updateJSON),
		Refresh:    "true",
	}

	// Execute request
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return ErrNotFound
		}
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	return nil
}

// EnsureIndex creates the index with mappings if it doesn't exist
func (r *esRepositoryImpl) EnsureIndex(ctx context.Context) error {
	// Check if index exists
	req := esapi.IndicesExistsRequest{
		Index: []string{r.index},
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	// If index exists, return
	if res.StatusCode == 200 {
		return nil
	}
	if res.StatusCode != 404 {
		return fmt.Errorf("unexpected response checking index existence: %s", res.String())
	}

	// Create index with mappings
	mappings := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"resume_id": map[string]interface{}{
					"type": "keyword",
				},
				"name": map[string]interface{}{
					"type":     "text",
					"analyzer": "standard",
				},
				"email": map[string]interface{}{
					"type":     "text",
					"analyzer": "standard",
				},
				"skills": map[string]interface{}{
					"type":     "text",
					"analyzer": "standard",
				},
				"experience_years": map[string]interface{}{
					"type": "integer",
				},
				"education": map[string]interface{}{
					"type":     "text",
					"analyzer": "standard",
				},
				"work_history": map[string]interface{}{
					"type":     "text",
					"analyzer": "standard",
				},
				"status": map[string]interface{}{
					"type": "keyword",
				},
				"source": map[string]interface{}{
					"type": "keyword",
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
				"updated_at": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}

	mappingsJSON, err := json.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("failed to marshal mappings: %w", err)
	}

	// Create index
	createReq := esapi.IndicesCreateRequest{
		Index: r.index,
		Body:  bytes.NewReader(mappingsJSON),
	}

	createRes, err := createReq.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", createRes.String())
	}

	return nil
}

// buildSearchQuery builds an Elasticsearch query from the search filter
func (r *esRepositoryImpl) buildSearchQuery(filter SearchFilter) map[string]interface{} {
	boolQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"must":     []interface{}{},
			"filter":   []interface{}{},
			"should":   []interface{}{},
			"must_not": []interface{}{},
		},
	}

	boolQueryMap := boolQuery["bool"].(map[string]interface{})
	mustQueries := boolQueryMap["must"].([]interface{})
	filterQueries := boolQueryMap["filter"].([]interface{})

	// Add full-text search query if provided
	if filter.Query != "" {
		multiMatchQuery := map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": filter.Query,
				"fields": []interface{}{
					"name^2",
					"skills^3",
					"work_history",
					"education",
				},
				"type":      "best_fields",
				"fuzziness": "AUTO",
			},
		}
		mustQueries = append(mustQueries, multiMatchQuery)
	}

	// Add skills filter if provided
	if len(filter.Skills) > 0 {
		skillsQuery := map[string]interface{}{
			"terms": map[string]interface{}{
				"skills": filter.Skills,
			},
		}
		filterQueries = append(filterQueries, skillsQuery)
	}

	// Add status filter if provided
	if filter.Status != "" {
		statusQuery := map[string]interface{}{
			"term": map[string]interface{}{
				"status": filter.Status,
			},
		}
		filterQueries = append(filterQueries, statusQuery)
	}

	// Add source filter if provided
	if filter.Source != "" {
		sourceQuery := map[string]interface{}{
			"term": map[string]interface{}{
				"source": filter.Source,
			},
		}
		filterQueries = append(filterQueries, sourceQuery)
	}

	// Add experience range filter
	if filter.MinExperience > 0 || filter.MaxExperience > 0 {
		rangeQuery := map[string]interface{}{
			"range": map[string]interface{}{
				"experience_years": map[string]interface{}{},
			},
		}
		rangeMap := rangeQuery["range"].(map[string]interface{})["experience_years"].(map[string]interface{})

		if filter.MinExperience > 0 {
			rangeMap["gte"] = filter.MinExperience
		}
		if filter.MaxExperience > 0 {
			rangeMap["lte"] = filter.MaxExperience
		}

		filterQueries = append(filterQueries, rangeQuery)
	}

	// Update the bool query with the modified slices
	boolQueryMap["must"] = mustQueries
	boolQueryMap["filter"] = filterQueries

	// If no filters, use match_all
	if len(mustQueries) == 0 && len(filterQueries) == 0 {
		return map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
	}

	return map[string]interface{}{
		"query": boolQuery,
	}
}
