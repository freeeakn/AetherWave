package blockchain

import (
	"fmt"
)

// serializeMessages сериализует сообщения блока в строку
func serializeMessages(messages []Message) string {
	var result string
	for _, msg := range messages {
		result += fmt.Sprintf("%s%s%s%d", msg.Sender, msg.Recipient, msg.Content, msg.Timestamp)
	}
	return result
}
