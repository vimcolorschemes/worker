package emoji

import (
	"testing"
)

func TestConvertColonEmojis(t *testing.T) {
	t.Run("should convert colon emojis to real emojis", func(t *testing.T) {
		value := ":thumbsup: this is a test phrase :package:"
		expected := "👍 this is a test phrase 📦"

		converted := ConvertColonEmojis(value)

		if converted != expected {
			t.Errorf("Incorrect result for ConvertColonEmojis; got: %s, expected: %s", converted, expected)
		}
	})

	t.Run("should not edit a value without colon emojis", func(t *testing.T) {
		value := "this is a test phrase"

		converted := ConvertColonEmojis(value)

		if value != converted {
			t.Errorf("Incorrect result for ConvertColonEmojis; got: %s, expected: %s", converted, value)
		}
	})

	t.Run("should not edit a value with real emojis and no colon emojis", func(t *testing.T) {
		value := "👍 this is a test phrase 📦"

		converted := ConvertColonEmojis(value)

		if value != converted {
			t.Errorf("Incorrect result for ConvertColonEmojis; got: %s, expected: %s", converted, value)
		}
	})

	t.Run("should convert colon emojis to real emojis, and not edit real emojis", func(t *testing.T) {
		value := ":thumbsup: :deciduous_tree: this is a 👞 test phrase :package: 🐈"
		expected := "👍 🌳 this is a 👞 test phrase 📦 🐈"

		converted := ConvertColonEmojis(value)

		if converted != expected {
			t.Errorf("Incorrect result for ConvertColonEmojis; got: %s, expected: %s", converted, expected)
		}
	})
}
