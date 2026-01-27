package judicial

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestColombianTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		expect  time.Time
		wantErr bool
	}{
		{
			name:    "Custom format 1",
			input:   `"2023-10-27T15:04:05"`,
			expect:  time.Date(2023, 10, 27, 15, 4, 5, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "RFC3339 format",
			input:   `"2023-10-27T15:04:05Z"`,
			expect:  time.Date(2023, 10, 27, 15, 4, 5, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Null value",
			input:   `null`,
			expect:  time.Time{},
			wantErr: false,
		},
		{
			name:    "Invalid format",
			input:   `"27-10-2023"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ct ColombianTime
			err := json.Unmarshal([]byte(tt.input), &ct)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.expect.IsZero() {
					assert.True(t, tt.expect.Equal(ct.Time))
				} else {
					assert.True(t, ct.Time.IsZero())
				}
			}
		})
	}
}

func TestGetProcessIDByRadicado(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/Procesos/Consulta/NumeroRadicacion")
		assert.Equal(t, "12345", r.URL.Query().Get("numero"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"procesos": [
				{
					"idProceso": 101,
					"esPrivado": false,
					"despacho": "Juzgado 1 Civil",
					"departamento": "Bogotá",
					"sujetosProcesales": "John Doe vs Jane Doe"
				}
			]
		}`)
	}))
	defer server.Close()

	// Override base URL for test
	oldURL := ColombiaBaseURL
	ColombiaBaseURL = server.URL
	defer func() { ColombiaBaseURL = oldURL }()

	service := NewColombiaService()
	summary, err := service.GetProcessIDByRadicado("12345")

	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, "101", summary.ProcessID)
	assert.Equal(t, "Juzgado 1 Civil", summary.Office)
	assert.Equal(t, "Bogotá", summary.Department)
}

func TestGetProcessDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/Proceso/Detalle/101")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tipoProceso": "Civil", "ponente": "Judge Dredd"}`)
	}))
	defer server.Close()

	oldURL := ColombiaBaseURL
	ColombiaBaseURL = server.URL
	defer func() { ColombiaBaseURL = oldURL }()

	service := NewColombiaService()
	detail, err := service.GetProcessDetail("101")

	assert.NoError(t, err)
	assert.Equal(t, "Civil", detail["process_type"])
	assert.Equal(t, "Judge Dredd", detail["judge"])
}

func TestGetProcessActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/Proceso/Actuaciones/101")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"actuaciones": [
				{
					"idRegActuacion": 201,
					"actuacion": "Auto Abre Pruebas",
					"anotacion": "Evidence requested",
					"fechaActuacion": "2023-10-27T08:00:00",
					"conDocumentos": true
				}
			],
			"paginacion": {"totalPaginas": 1}
		}`)
	}))
	defer server.Close()

	oldURL := ColombiaBaseURL
	ColombiaBaseURL = server.URL
	defer func() { ColombiaBaseURL = oldURL }()

	service := NewColombiaService()
	actions, err := service.GetProcessActions("101")

	assert.NoError(t, err)
	assert.Len(t, actions, 1)
	assert.Equal(t, "201", actions[0].ExternalID)
	assert.Equal(t, "Auto Abre Pruebas", actions[0].Type)
	assert.True(t, actions[0].HasDocuments)
	assert.False(t, actions[0].ActionDate.IsZero())
}
