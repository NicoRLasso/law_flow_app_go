package services

import (
	"context"
	"fmt"
	"html"
	"log"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// SearchResult represents a search result
type SearchResult struct {
	// Common fields
	Type        string  `json:"type"` // "case" or "service"
	ClientName  string  `json:"client_name"`
	Snippet     string  `json:"snippet"`
	MatchSource string  `json:"match_source"`
	Rank        float64 `json:"rank"`

	// Case-specific fields
	CaseID       string `json:"case_id,omitempty"`
	CaseNumber   string `json:"case_number,omitempty"`
	FilingNumber string `json:"filing_number,omitempty"`
	CaseTitle    string `json:"case_title,omitempty"`
	Status       string `json:"status,omitempty"`

	// Service-specific fields
	ServiceID     string `json:"service_id,omitempty"`
	ServiceNumber string `json:"service_number,omitempty"`
	ServiceTitle  string `json:"service_title,omitempty"`
	ServiceStatus string `json:"service_status,omitempty"`
}

// SearchService handles FTS5 searches
type SearchService struct {
	db *gorm.DB
}

// NewSearchService creates a new search service instance
func NewSearchService(db *gorm.DB) *SearchService {
	return &SearchService{db: db}
}

// Search performs an FTS5 search with firm filter (searches both cases and services)
func (s *SearchService) Search(ctx context.Context, firmID, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	searchQuery := sanitizeFTSQuery(query)
	if searchQuery == "" {
		return []SearchResult{}, nil
	}

	var results []SearchResult

	// Search cases
	caseResults, err := s.searchCases(ctx, firmID, searchQuery, limit)
	if err != nil {
		return nil, err
	}
	log.Printf("[SEARCH DEBUG] Cases found: %d", len(caseResults))
	results = append(results, caseResults...)

	// Search services
	serviceResults, err := s.searchServices(ctx, firmID, searchQuery, limit)
	if err != nil {
		log.Printf("[SEARCH DEBUG] Service search error: %v", err)
		return nil, err
	}
	log.Printf("[SEARCH DEBUG] Services found: %d", len(serviceResults))
	results = append(results, serviceResults...)

	// Sort by rank
	// Simple sort - in production you might want more sophisticated ranking
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// searchCases performs FTS5 search on cases
func (s *SearchService) searchCases(ctx context.Context, firmID, searchQuery string, limit int) ([]SearchResult, error) {
	var results []SearchResult

	sql := `
		SELECT
			m.case_id,
			c.case_number,
			COALESCE(c.filing_number, '') as filing_number,
			COALESCE(c.title, '') as case_title,
			COALESCE(u.name, '') as client_name,
			c.status,
			snippet(cases_fts, -1, '<mark>', '</mark>', '...', 32) as snippet,
			bm25(cases_fts) as rank
		FROM cases_fts
		INNER JOIN cases_fts_mapping m ON cases_fts.rowid = m.rowid
		INNER JOIN cases c ON m.case_id = c.id
		LEFT JOIN users u ON c.client_id = u.id
		WHERE cases_fts MATCH ?
		  AND m.firm_id = ?
		  AND c.deleted_at IS NULL
		ORDER BY rank
		LIMIT ?
	`

	err := s.db.WithContext(ctx).Raw(sql, searchQuery, firmID, limit).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Process results
	for i := range results {
		results[i].Type = "case"
		results[i].MatchSource = determineMatchSource(results[i].Snippet)
		results[i].Snippet = processSnippet(results[i].Snippet)
	}

	return results, nil
}

// searchServices performs FTS5 search on services
func (s *SearchService) searchServices(ctx context.Context, firmID, searchQuery string, limit int) ([]SearchResult, error) {
	var rawResults []struct {
		ServiceID     string
		ServiceNumber string
		ServiceTitle  string
		ClientName    string
		ServiceStatus string
		Snippet       string
		Rank          float64
	}

	sql := `
		SELECT
			m.service_id,
			s.service_number,
			s.title as service_title,
			COALESCE(u.name, '') as client_name,
			s.status as service_status,
			snippet(services_fts, -1, '<mark>', '</mark>', '...', 32) as snippet,
			bm25(services_fts) as rank
		FROM services_fts
		INNER JOIN services_fts_mapping m ON services_fts.rowid = m.rowid
		INNER JOIN legal_services s ON m.service_id = s.id
		LEFT JOIN users u ON s.client_id = u.id
		WHERE services_fts MATCH ?
		  AND m.firm_id = ?
		  AND s.deleted_at IS NULL
		ORDER BY rank
		LIMIT ?
	`

	err := s.db.WithContext(ctx).Raw(sql, searchQuery, firmID, limit).Scan(&rawResults).Error
	if err != nil {
		log.Printf("[SEARCH DEBUG] Service search SQL error: %v", err)
		return nil, fmt.Errorf("service search failed: %w", err)
	}
	log.Printf("[SEARCH DEBUG] Service raw results count: %d", len(rawResults))

	// Convert to SearchResult with Type field
	results := make([]SearchResult, len(rawResults))
	for i, r := range rawResults {
		results[i] = SearchResult{
			Type:          "service",
			ServiceID:     r.ServiceID,
			ServiceNumber: r.ServiceNumber,
			ServiceTitle:  r.ServiceTitle,
			ClientName:    r.ClientName,
			ServiceStatus: r.ServiceStatus,
			Snippet:       processSnippet(r.Snippet),
			MatchSource:   determineMatchSource(r.Snippet),
			Rank:          r.Rank,
		}
	}

	return results, nil
}

// SearchWithRoleFilter performs an FTS5 search with firm and role filters (searches both cases and services)
func (s *SearchService) SearchWithRoleFilter(ctx context.Context, firmID, userID, role, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	searchQuery := sanitizeFTSQuery(query)
	if searchQuery == "" {
		return []SearchResult{}, nil
	}

	var results []SearchResult

	// Search cases with role filter
	caseResults, err := s.searchCasesWithRoleFilter(ctx, firmID, userID, role, searchQuery, limit)
	if err != nil {
		log.Printf("[SEARCH DEBUG] Case search with role filter error: %v", err)
		return nil, err
	}
	log.Printf("[SEARCH DEBUG] Cases found with role filter: %d", len(caseResults))
	results = append(results, caseResults...)

	// Search services with role filter
	serviceResults, err := s.searchServicesWithRoleFilter(ctx, firmID, userID, role, searchQuery, limit)
	if err != nil {
		log.Printf("[SEARCH DEBUG] Service search with role filter error: %v", err)
		return nil, err
	}
	log.Printf("[SEARCH DEBUG] Services found with role filter: %d", len(serviceResults))
	results = append(results, serviceResults...)

	// Sort by rank and limit
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// searchCasesWithRoleFilter performs FTS5 search on cases with role-based access control
func (s *SearchService) searchCasesWithRoleFilter(ctx context.Context, firmID, userID, role, searchQuery string, limit int) ([]SearchResult, error) {
	var results []SearchResult
	var sql string
	var args []interface{}

	baseSelect := `
		SELECT
			m.case_id,
			c.case_number,
			COALESCE(c.filing_number, '') as filing_number,
			COALESCE(c.title, '') as case_title,
			COALESCE(u.name, '') as client_name,
			c.status,
			snippet(cases_fts, -1, '<mark>', '</mark>', '...', 32) as snippet,
			bm25(cases_fts) as rank
		FROM cases_fts
		INNER JOIN cases_fts_mapping m ON cases_fts.rowid = m.rowid
		INNER JOIN cases c ON m.case_id = c.id
		LEFT JOIN users u ON c.client_id = u.id
		WHERE cases_fts MATCH ?
		  AND m.firm_id = ?
		  AND c.deleted_at IS NULL
	`

	switch role {
	case "client":
		// Clients only see their own cases
		sql = baseSelect + ` AND c.client_id = ? ORDER BY rank LIMIT ?`
		args = []interface{}{searchQuery, firmID, userID, limit}
	case "lawyer":
		// Lawyers see assigned cases or cases where they are collaborators
		sql = baseSelect + `
			AND (c.assigned_to_id = ?
				 OR EXISTS (SELECT 1 FROM case_collaborators cc WHERE cc.case_id = c.id AND cc.user_id = ?))
			ORDER BY rank LIMIT ?`
		args = []interface{}{searchQuery, firmID, userID, userID, limit}
	default:
		// Admin and staff see all cases in the firm
		sql = baseSelect + ` ORDER BY rank LIMIT ?`
		args = []interface{}{searchQuery, firmID, limit}
	}

	err := s.db.WithContext(ctx).Raw(sql, args...).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("case search failed: %w", err)
	}

	// Process results
	for i := range results {
		results[i].Type = "case"
		results[i].MatchSource = determineMatchSource(results[i].Snippet)
		results[i].Snippet = processSnippet(results[i].Snippet)
	}

	return results, nil
}

// searchServicesWithRoleFilter performs FTS5 search on services with role-based access control
func (s *SearchService) searchServicesWithRoleFilter(ctx context.Context, firmID, userID, role, searchQuery string, limit int) ([]SearchResult, error) {
	var rawResults []struct {
		ServiceID     string
		ServiceNumber string
		ServiceTitle  string
		ClientName    string
		ServiceStatus string
		Snippet       string
		Rank          float64
	}

	var sql string
	var args []interface{}

	baseSelect := `
		SELECT
			m.service_id,
			s.service_number,
			s.title as service_title,
			COALESCE(u.name, '') as client_name,
			s.status as service_status,
			snippet(services_fts, -1, '<mark>', '</mark>', '...', 32) as snippet,
			bm25(services_fts) as rank
		FROM services_fts
		INNER JOIN services_fts_mapping m ON services_fts.rowid = m.rowid
		INNER JOIN legal_services s ON m.service_id = s.id
		LEFT JOIN users u ON s.client_id = u.id
		WHERE services_fts MATCH ?
		  AND m.firm_id = ?
		  AND s.deleted_at IS NULL
	`

	switch role {
	case "client":
		// Clients only see their own services
		sql = baseSelect + ` AND s.client_id = ? ORDER BY rank LIMIT ?`
		args = []interface{}{searchQuery, firmID, userID, limit}
	case "lawyer":
		// Lawyers see assigned services
		sql = baseSelect + ` AND s.assigned_to_id = ? ORDER BY rank LIMIT ?`
		args = []interface{}{searchQuery, firmID, userID, limit}
	default:
		// Admin and staff see all services in the firm
		sql = baseSelect + ` ORDER BY rank LIMIT ?`
		args = []interface{}{searchQuery, firmID, limit}
	}

	err := s.db.WithContext(ctx).Raw(sql, args...).Scan(&rawResults).Error
	if err != nil {
		log.Printf("[SEARCH DEBUG] Service search SQL error: %v", err)
		return nil, fmt.Errorf("service search failed: %w", err)
	}
	log.Printf("[SEARCH DEBUG] Service raw results count: %d", len(rawResults))

	// Convert to SearchResult with Type field
	results := make([]SearchResult, len(rawResults))
	for i, r := range rawResults {
		results[i] = SearchResult{
			Type:          "service",
			ServiceID:     r.ServiceID,
			ServiceNumber: r.ServiceNumber,
			ServiceTitle:  r.ServiceTitle,
			ClientName:    r.ClientName,
			ServiceStatus: r.ServiceStatus,
			Snippet:       processSnippet(r.Snippet),
			MatchSource:   determineMatchSource(r.Snippet),
			Rank:          r.Rank,
		}
	}

	return results, nil
}

// sanitizeFTSQuery prepares the query for FTS5
func sanitizeFTSQuery(query string) string {
	// Remove FTS5 special characters
	specialChars := regexp.MustCompile(`[*"():\-+^]`)
	cleaned := specialChars.ReplaceAllString(query, " ")

	// Split into words and add prefix for partial matching
	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return ""
	}

	var parts []string
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) >= 2 { // Minimum 2 characters
			parts = append(parts, word+"*")
		}
	}

	if len(parts) == 0 {
		return ""
	}

	// Use OR for multiple words to match any
	return strings.Join(parts, " OR ")
}

// determineMatchSource determines where the match was found
func determineMatchSource(snippet string) string {
	snippet = strings.ToLower(snippet)

	// Check for common patterns
	if strings.Contains(snippet, ".pdf") || strings.Contains(snippet, ".doc") || strings.Contains(snippet, ".xls") {
		return "document"
	}

	return "case"
}

// processSnippet escapes HTML but preserves mark tags
func processSnippet(snippet string) string {
	// First escape everything
	escaped := html.EscapeString(snippet)

	// Then restore mark tags
	escaped = strings.ReplaceAll(escaped, "&lt;mark&gt;", "<mark>")
	escaped = strings.ReplaceAll(escaped, "&lt;/mark&gt;", "</mark>")

	return escaped
}
