package judicial

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var ColombiaBaseURL = "https://consultaprocesos.ramajudicial.gov.co:448/api/v2"

// ColombiaService implements Provider for Colombia
type ColombiaService struct {
	BaseService
}

// NewColombiaService creates a new instance
func NewColombiaService() *ColombiaService {
	return &ColombiaService{
		BaseService: NewBaseService(),
	}
}

// ColombianTime handles dates without timezone
type ColombianTime struct {
	time.Time
}

func (ct *ColombianTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		return nil
	}
	s = s[1 : len(s)-1] // Remove quotes

	// Try format from error: 2006-01-02T15:04:05
	t, err := time.Parse("2006-01-02T15:04:05", s)
	if err == nil {
		ct.Time = t
		return nil
	}

	// Try standard RFC3339 just in case
	t, err = time.Parse(time.RFC3339, s)
	if err == nil {
		ct.Time = t
		return nil
	}

	return err
}

// === Colombia Internal Structs ===

type coSearchResponse struct {
	Procesos []coProcessSummary `json:"procesos"`
}

type coProcessSummary struct {
	IDProceso            int64          `json:"idProceso"`
	IDConexion           int64          `json:"idConexion"`
	EsPrivado            bool           `json:"esPrivado"`
	FechaProceso         *ColombianTime `json:"fechaProceso"`
	FechaUltimaActuacion *ColombianTime `json:"fechaUltimaActuacion"`
	Despacho             string         `json:"despacho"`
	Departamento         string         `json:"departamento"`
	SujetosProcesales    string         `json:"sujetosProcesales"`
}

type coProcessDetail struct {
	TipoProceso string `json:"tipoProceso"`
	Ponente     string `json:"ponente"`
}

type coProcessAction struct {
	IDRegActuacion int64          `json:"idRegActuacion"`
	Actuacion      string         `json:"actuacion"`
	Anotacion      string         `json:"anotacion"`
	FechaActuacion *ColombianTime `json:"fechaActuacion"`
	FechaRegistro  *ColombianTime `json:"fechaRegistro"`
	FechaInicial   *ColombianTime `json:"fechaInicial"`
	FechaFinal     *ColombianTime `json:"fechaFinal"`
	ConDocumentos  bool           `json:"conDocumentos"`
}

// GetProcessIDByRadicado implements Provider
func (s *ColombiaService) GetProcessIDByRadicado(radicado string) (*GenericProcessSummary, error) {
	params := url.Values{}
	params.Add("numero", radicado)
	params.Add("SoloActivos", "true")
	params.Add("pagina", "1")

	reqURL := fmt.Sprintf("%s/Procesos/Consulta/NumeroRadicacion?%s", ColombiaBaseURL, params.Encode())

	resp, err := s.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var searchResp coSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchResp.Procesos) == 0 {
		return nil, nil // No process found
	}

	coProc := searchResp.Procesos[0]
	return &GenericProcessSummary{
		ProcessID:  fmt.Sprintf("%d", coProc.IDProceso),
		Radicado:   radicado,
		IsPrivate:  coProc.EsPrivado,
		Department: coProc.Departamento,
		Office:     coProc.Despacho,
		Subject:    coProc.SujetosProcesales,
	}, nil
}

// GetProcessDetail implements Provider
func (s *ColombiaService) GetProcessDetail(processID string) (map[string]interface{}, error) {
	reqURL := fmt.Sprintf("%s/Proceso/Detalle/%s", ColombiaBaseURL, processID)

	resp, err := s.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var detailResp coProcessDetail
	if err := json.NewDecoder(resp.Body).Decode(&detailResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return map[string]interface{}{
		"process_type": detailResp.TipoProceso,
		"judge":        detailResp.Ponente,
	}, nil
}

// GetProcessActions implements Provider
func (s *ColombiaService) GetProcessActions(processID string) ([]GenericAction, error) {
	// Only fetch page 1 for now to catch latest updates.
	reqURL := fmt.Sprintf("%s/Proceso/Actuaciones/%s?pagina=1", ColombiaBaseURL, processID)

	resp, err := s.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var actionsResp struct {
		Actuaciones []coProcessAction `json:"actuaciones"`
		Pagina      struct {
			TotalPaginas int `json:"totalPaginas"`
		} `json:"paginacion"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&actionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var genericActions []GenericAction
	for _, act := range actionsResp.Actuaciones {
		genAct := GenericAction{
			ExternalID:   fmt.Sprintf("%d", act.IDRegActuacion),
			Type:         act.Actuacion,
			Annotation:   act.Anotacion,
			HasDocuments: act.ConDocumentos,
			Metadata:     make(map[string]interface{}),
		}
		if act.FechaActuacion != nil {
			genAct.ActionDate = act.FechaActuacion.Time
		}
		if act.FechaRegistro != nil {
			genAct.RegistrationDate = act.FechaRegistro.Time
		}
		if act.FechaInicial != nil {
			t := act.FechaInicial.Time
			genAct.InitialDate = &t
		}
		if act.FechaFinal != nil {
			t := act.FechaFinal.Time
			genAct.FinalDate = &t
		}

		genericActions = append(genericActions, genAct)
	}

	return genericActions, nil
}
