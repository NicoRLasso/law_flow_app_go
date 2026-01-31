package jobs

import (
	"errors"
	"law_flow_app_go/models"
	"law_flow_app_go/services/judicial"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockProvider is a mock of judicial.Provider
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) GetProcessIDByRadicado(radicado string) (*judicial.GenericProcessSummary, error) {
	args := m.Called(radicado)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*judicial.GenericProcessSummary), args.Error(1)
}

func (m *MockProvider) GetProcessDetail(processID string) (map[string]interface{}, error) {
	args := m.Called(processID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockProvider) GetProcessActions(processID string) ([]judicial.GenericAction, error) {
	args := m.Called(processID)
	return args.Get(0).([]judicial.GenericAction), args.Error(1)
}

func setupJudicialJobTestDB(dsn string) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	db.AutoMigrate(&models.Firm{}, &models.Case{}, &models.CaseDomain{}, &models.CaseBranch{}, &models.CaseSubtype{}, &models.User{}, &models.JudicialProcess{}, &models.JudicialProcessAction{}, &models.ChoiceOption{}, &models.Notification{}, &models.Country{})
	return db
}

func strToPtr(s string) *string {
	return &s
}

func TestProcessCase(t *testing.T) {
	// Use unique DSN for isolation
	testDSN := "file:process_case_" + uuid.New().String() + "?mode=memory&cache=shared"
	db := setupJudicialJobTestDB(testDSN)

	firmID := uuid.New().String()
	firm := models.Firm{
		ID:        firmID,
		Name:      "Test Firm",
		CountryID: uuid.New().String(),
		Country:   &models.Country{Name: "CO_PROCESS"},
	}
	db.Create(&firm)

	caseRecord := models.Case{
		ID:           uuid.New().String(),
		FirmID:       firm.ID,
		Firm:         firm,
		CaseNumber:   "CASE-001",
		FilingNumber: strToPtr("12345"),
		Status:       models.CaseStatusOpen,
	}
	db.Create(&caseRecord)

	mockProv := new(MockProvider)
	judicial.RegisterProvider("CO_PROCESS", mockProv)
	defer judicial.RegisterProvider("CO_PROCESS", nil)

	t.Run("Initial sync - Create record and actions", func(t *testing.T) {
		summary := &judicial.GenericProcessSummary{
			ProcessID:  "PROC-1",
			Radicado:   "12345",
			Department: "Bogota",
			Office:     "Juzgado 1",
		}
		detail := map[string]interface{}{"process_type": "Civil"}
		actions := []judicial.GenericAction{
			{
				ExternalID: "ACT-1",
				Type:       "Auto",
				Annotation: "Initial Action",
				ActionDate: time.Now().Add(-24 * time.Hour),
			},
		}

		mockProv.On("GetProcessIDByRadicado", "12345").Return(summary, nil).Once()
		mockProv.On("GetProcessDetail", "PROC-1").Return(detail, nil).Once()
		mockProv.On("GetProcessActions", "PROC-1").Return(actions, nil).Once()

		err := processCase(db, caseRecord)
		assert.NoError(t, err)

		// Verify JudicialProcess was created
		var jp models.JudicialProcess
		err = db.Where("case_id = ?", caseRecord.ID).First(&jp).Error
		assert.NoError(t, err)
		assert.Equal(t, "PROC-1", jp.ProcessID)

		// Verify Actions were created
		var dbActions []models.JudicialProcessAction
		db.Where("judicial_process_id = ?", jp.ID).Find(&dbActions)
		assert.Len(t, dbActions, 1)
	})

	t.Run("Subsequent sync - Only new actions", func(t *testing.T) {
		var jp models.JudicialProcess
		db.Where("case_id = ?", caseRecord.ID).First(&jp)

		newActions := []judicial.GenericAction{
			{
				ExternalID: "ACT-2", // New
				Type:       "Sentencia",
				ActionDate: time.Now(),
			},
			{
				ExternalID: "ACT-1", // Old
				Type:       "Auto",
				ActionDate: time.Now().Add(-24 * time.Hour),
			},
		}

		mockProv.On("GetProcessActions", "PROC-1").Return(newActions, nil).Once()

		err := processCase(db, caseRecord)
		assert.NoError(t, err)

		var dbActions []models.JudicialProcessAction
		db.Where("judicial_process_id = ?", jp.ID).Find(&dbActions)
		assert.Len(t, dbActions, 2)
	})
}

func TestUpdateAllJudicialProcesses(t *testing.T) {
	testDSN := "file:all_judicial_" + uuid.New().String() + "?mode=memory&cache=shared"
	db := setupJudicialJobTestDB(testDSN)

	firmID := uuid.New().String()
	firm := models.Firm{
		ID:        firmID,
		Name:      "Test Firm",
		CountryID: uuid.New().String(),
		Country:   &models.Country{Name: "CO_ALL"},
	}
	db.Create(&firm)

	db.Create(&models.Case{
		ID:           uuid.New().String(),
		FirmID:       firm.ID,
		CaseNumber:   "C1",
		Status:       models.CaseStatusOpen,
		FilingNumber: strToPtr("R1"),
	})

	db.Create(&models.Case{
		ID:           uuid.New().String(),
		FirmID:       firm.ID,
		CaseNumber:   "C2",
		Status:       models.CaseStatusClosed,
		FilingNumber: strToPtr("R2"),
	})

	mockProv := new(MockProvider)
	judicial.RegisterProvider("CO_ALL", mockProv)
	defer judicial.RegisterProvider("CO_ALL", nil)

	mockProv.On("GetProcessIDByRadicado", "R1").Return(nil, errors.New("not found")).Once()

	UpdateAllJudicialProcesses(db)
	mockProv.AssertExpectations(t)
}
