package i18n

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
)

//go:embed *.json
var fs embed.FS

// translations stores flattened keys: "en" -> "nav.home" -> "Home"
var (
	translations = make(map[string]map[string]string)
	mutex        sync.RWMutex
	defaultLang  = "en"
)

// Load initializes the translations from the embedded JSON files.
// It searches for any .json file in the locales directory (embedded) and loads it.
func Load() error {
	mutex.Lock()
	defer mutex.Unlock()

	entries, err := fs.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read embedded locales: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			lang := strings.TrimSuffix(entry.Name(), ".json")
			content, err := fs.ReadFile(entry.Name())
			if err != nil {
				return fmt.Errorf("failed to read locale file %s: %w", entry.Name(), err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(content, &result); err != nil {
				return fmt.Errorf("failed to unmarshal locale %s: %w", entry.Name(), err)
			}

			flat := make(map[string]string)
			flatten("", result, flat)
			translations[lang] = flat
			log.Printf("Loaded locale: %s (%d keys)", lang, len(flat))
		}
	}

	return nil
}

// flatten recursively flattens a nested map into dot-notation keys.
func flatten(prefix string, nested map[string]interface{}, result map[string]string) {
	for k, v := range nested {
		newKey := k
		if prefix != "" {
			newKey = prefix + "." + k
		}

		switch child := v.(type) {
		case map[string]interface{}:
			flatten(newKey, child, result)
		case string:
			result[newKey] = child
		default:
			// Fallback for types not strictly string (e.g. numbers in json), convert to string
			result[newKey] = fmt.Sprintf("%v", child)
		}
	}
}

// T retrieves a translation for the given key using the language from the context.
// If the key is missing in the target language, it falls back to the default language,
// and then to the key itself.
// Supports simple named variable replacement {name} if args are provided (map[string]interface{}).
func T(ctx context.Context, key string, args ...map[string]interface{}) string {
	lang := GetLocale(ctx)
	return Translate(lang, key, args...)
}

// Translate retrieves a translation for a specific language code.
func Translate(lang, key string, args ...map[string]interface{}) string {
	mutex.RLock()
	defer mutex.RUnlock()

	// 1. Try requested language
	if trans, ok := translations[lang]; ok {
		if val, ok := trans[key]; ok {
			return format(val, args...)
		}
	}

	// 2. Try default language if different
	if lang != defaultLang {
		if trans, ok := translations[defaultLang]; ok {
			if val, ok := trans[key]; ok {
				return format(val, args...)
			}
		}
	}

	// 3. Fallback to key
	return key
}

// format replaces {var} placeholders with values from args if present.
func format(text string, args ...map[string]interface{}) string {
	if len(args) == 0 {
		return text
	}

	// Simple replacement for {key}
	vars := args[0]
	for k, v := range vars {
		placeholder := "{" + k + "}"
		valStr := fmt.Sprintf("%v", v)
		text = strings.ReplaceAll(text, placeholder, valStr)
	}
	return text
}

// Keys for context storage
type contextKey string

const LocaleContextKey contextKey = "locale"

// GetLocale extracts the locale from the context, defaulting to "en".
// It looks for a string value stored under "locale" (set by middleware).
func GetLocale(ctx context.Context) string {
	// Try to get from echo context if available (passed as std context)
	// Since we are using standard context.Context here, we rely on the middleware
	// to put the value into the request context.

	if val := ctx.Value(LocaleContextKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}

	// Fallback to "language" string key which might be set by some middleware
	if val := ctx.Value("language"); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}

	return defaultLang
}
