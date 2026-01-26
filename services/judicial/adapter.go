package judicial

import (
	"fmt"
	"net/http"
	"time"
)

// Provider defines the interface for all country-specific judicial implementations
type Provider interface {
	// GetProcessIDByRadicado searches and returns a unified summary
	GetProcessIDByRadicado(radicado string) (*GenericProcessSummary, error)

	// GetProcessDetail fetches details and returns them as a simplified map
	// The map should contain keys like: "office", "judge", "process_type", "subjects", "department"
	GetProcessDetail(processID string) (map[string]interface{}, error)

	// GetProcessActions fetches actions and returns unified GenericActions
	GetProcessActions(processID string) ([]GenericAction, error)
}

// GenericProcessSummary normalizes the initial search return
type GenericProcessSummary struct {
	ProcessID  string
	Radicado   string
	IsPrivate  bool
	Department string // Common enough to keep top level, or put in metadata
	Office     string // Common enough
	Subject    string // Common enough
}

// GenericAction normalizes events across different systems
type GenericAction struct {
	ExternalID       string
	Type             string    // The action name/type
	Annotation       string    // Description
	ActionDate       time.Time // When it happened
	RegistrationDate time.Time // When it was recorded in system
	InitialDate      *time.Time
	FinalDate        *time.Time
	HasDocuments     bool
	Metadata         map[string]interface{} // Any extra country-specific fields
}

// BaseService provides common functionality like HTTP client
type BaseService struct {
	client *http.Client
}

// NewBaseService creates a configured base service
func NewBaseService() BaseService {
	return BaseService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetProvider returns the correct implementation based on country code
func GetProvider(countryCode string) (Provider, error) {
	switch countryCode {
	case "CO", "Colombia", "colombia":
		return NewColombiaService(), nil
	default:
		return nil, fmt.Errorf("judicial provider not implemented for country: %s", countryCode)
	}
}
