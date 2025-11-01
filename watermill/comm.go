package watermill

type MessageHandler func(messageId, messageData string) error
