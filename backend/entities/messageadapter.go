package entities

type MessageTransformer interface {
	ToExternal(msg Message) ([]byte, error)
	FromExternal(rawMsg []byte) (Message, error)
}
