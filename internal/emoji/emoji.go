package emoji

import (
	emojiHelper "github.com/enescakir/emoji"
)

// ConvertColonEmojis converts all colon emojis to real emojis in a string
func ConvertColonEmojis(value string) string {
	return emojiHelper.Parse(value)
}
