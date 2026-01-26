package services

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// SearchResult represents a search result
type SearchResult struct {
	CaseID      string `json:"case_id"`
	CaseNumber  string `json:"case_number"`
	CaseTitle   string `json:"case_title"`
	ClientName  string `json:"client_name"`
	Status      string `json:"status"`
	Snippet     string `json:"snippet"`
	MatchSource string `json:"match_source"`
	Rank        float64
}

// SearchService handles FTS5 searches
type SearchService struct {
	db *gorm.DB
}

// NewSearchService creates a new search service instance
func NewSearchService(db *gorm.DB) *SearchService {
	return &SearchService{db: db}
}

// Search performs an FTS5 search with firm filter
func (s *SearchService) Search(ctx context.Context, firmID, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	searchQuery := sanitizeFTSQuery(query)
	if searchQuery == "" {
		return []SearchResult{}, nil
	}

	var results []SearchResult

	sql := `
		SELECT
			m.case_id,
			c.case_number,
			COALESCE(c.title, '') as case_title,
			COALESCE(u.name, '') as client_name,
			c.status,
			snippet(cases_fts, 5, '<mark>', '</mark>', '...', 32) as snippet,
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
		results[i].MatchSource = determineMatchSource(results[i].Snippet)
		results[i].Snippet = processSnippet(results[i].Snippet)
	}

	return results, nil
}

// SearchWithRoleFilter performs an FTS5 search with firm and role filters
func (s *SearchService) SearchWithRoleFilter(ctx context.Context, firmID, userID, role, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	searchQuery := sanitizeFTSQuery(query)
	if searchQuery == "" {
		return []SearchResult{}, nil
	}

	var results []SearchResult
	var sql string
	var args []interface{}

	baseSelect := `
		SELECT
			m.case_id,
			c.case_number,
			COALESCE(c.title, '') as case_title,
			COALESCE(u.name, '') as client_name,
			c.status,
			snippet(cases_fts, 5, '<mark>', '</mark>', '...', 32) as snippet,
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
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Process results
	for i := range results {
		results[i].MatchSource = determineMatchSource(results[i].Snippet)
		results[i].Snippet = processSnippet(results[i].Snippet)
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
