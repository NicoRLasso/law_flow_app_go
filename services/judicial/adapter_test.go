package judicial

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAdapterProvider struct {
	mock.Mock
}

func (m *MockAdapterProvider) GetProcessIDByRadicado(radicado string) (*GenericProcessSummary, error) {
	args := m.Called(radicado)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GenericProcessSummary), args.Error(1)
}

func (m *MockAdapterProvider) GetProcessDetail(processID string) (map[string]interface{}, error) {
	args := m.Called(processID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockAdapterProvider) GetProcessActions(processID string) ([]GenericAction, error) {
	args := m.Called(processID)
	return args.Get(0).([]GenericAction), args.Error(1)
}

func TestGetProvider(t *testing.T) {
	t.Run("Default Colombia provider", func(t *testing.T) {
		p, err := GetProvider("CO")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.IsType(t, &ColombiaService{}, p)

		p, err = GetProvider("Colombia")
		assert.NoError(t, err)
		assert.NotNil(t, p)

		p, err = GetProvider("colombia")
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("Unsupported country", func(t *testing.T) {
		p, err := GetProvider("US")
		assert.Error(t, err)
		assert.Nil(t, p)
		assert.Contains(t, err.Error(), "judicial provider not implemented")
	})

	t.Run("Registered mock provider", func(t *testing.T) {
		mockP := new(MockAdapterProvider)
		RegisterProvider("MOCK", mockP)
		defer RegisterProvider("MOCK", nil)

		p, err := GetProvider("MOCK")
		assert.NoError(t, err)
		assert.Equal(t, mockP, p)
	})
}

func TestRegisterProvider(t *testing.T) {
	mockP := new(MockAdapterProvider)
	RegisterProvider("TEST", mockP)
	defer RegisterProvider("TEST", nil)

	p, ok := providers["TEST"]
	assert.True(t, ok)
	assert.Equal(t, mockP, p)
}

func TestNewBaseService(t *testing.T) {
	svc := NewBaseService()
	assert.NotNil(t, svc.client)
	assert.Equal(t, 30*time.Second, svc.client.Timeout)
}
