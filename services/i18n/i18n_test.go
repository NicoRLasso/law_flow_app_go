package i18n

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlatten(t *testing.T) {
	nested := map[string]interface{}{
		"nav": map[string]interface{}{
			"home": "Home",
			"settings": map[string]interface{}{
				"title": "Settings",
			},
		},
		"count": 123,
	}

	flat := make(map[string]string)
	flatten("", nested, flat)

	assert.Equal(t, "Home", flat["nav.home"])
	assert.Equal(t, "Settings", flat["nav.settings.title"])
	assert.Equal(t, "123", flat["count"])
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		args     map[string]interface{}
		expected string
	}{
		{
			name:     "No placeholders",
			text:     "Hello World",
			args:     nil,
			expected: "Hello World",
		},
		{
			name:     "Single placeholder",
			text:     "Hello {name}",
			args:     map[string]interface{}{"name": "John"},
			expected: "Hello John",
		},
		{
			name:     "Multiple placeholders",
			text:     "{greeting} {name}, you have {count} messages",
			args:     map[string]interface{}{"greeting": "Hi", "name": "Doe", "count": 5},
			expected: "Hi Doe, you have 5 messages",
		},
		{
			name:     "Missing argument",
			text:     "Hello {name}",
			args:     map[string]interface{}{"other": "val"},
			expected: "Hello {name}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.args == nil {
				result = format(tt.text)
			} else {
				result = format(tt.text, tt.args)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLocale(t *testing.T) {
	t.Run("Default locale", func(t *testing.T) {
		ctx := context.Background()
		assert.Equal(t, "en", GetLocale(ctx))
	})

	t.Run("Locale from LocaleContextKey", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), LocaleContextKey, "es")
		assert.Equal(t, "es", GetLocale(ctx))
	})

	t.Run("Locale from language string key", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "language", "fr")
		assert.Equal(t, "fr", GetLocale(ctx))
	})
}

func TestTranslateLogic(t *testing.T) {
	// Setup test translations
	mutex.Lock()
	// Back up existing if any
	oldTrans := translations
	translations = make(map[string]map[string]string)
	translations["en"] = map[string]string{
		"test.hello":   "Hello",
		"test.welcome": "Welcome {name}",
	}
	translations["es"] = map[string]string{
		"test.hello": "Hola",
	}
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		translations = oldTrans
		mutex.Unlock()
	}()

	t.Run("Direct lookup", func(t *testing.T) {
		assert.Equal(t, "Hola", Translate("es", "test.hello"))
		assert.Equal(t, "Hello", Translate("en", "test.hello"))
	})

	t.Run("Fallback to default", func(t *testing.T) {
		// es doesn't have test.welcome, should fallback to en
		assert.Equal(t, "Welcome Juan", Translate("es", "test.welcome", map[string]interface{}{"name": "Juan"}))
	})

	t.Run("Fallback to key", func(t *testing.T) {
		assert.Equal(t, "missing.key", Translate("es", "missing.key"))
	})
}

func TestT(t *testing.T) {
	// Setup test translations
	mutex.Lock()
	oldTrans := translations
	translations = make(map[string]map[string]string)
	translations["es"] = map[string]string{
		"greet": "Hola {name}",
	}
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		translations = oldTrans
		mutex.Unlock()
	}()

	ctx := context.WithValue(context.Background(), LocaleContextKey, "es")
	result := T(ctx, "greet", map[string]interface{}{"name": "Pedro"})
	assert.Equal(t, "Hola Pedro", result)
}

func TestLoadExecution(t *testing.T) {
	// This tests that Load doesn't panic and loads something
	err := Load()
	assert.NoError(t, err)

	mutex.RLock()
	defer mutex.RUnlock()
	assert.NotEmpty(t, translations["en"])
	assert.NotEmpty(t, translations["es"])
}
