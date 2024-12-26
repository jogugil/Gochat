package entities

type MessageProxy interface {
	TransformToExternal(msg Message) ([]byte, error)
	TransformFromExternal(rawMsg []byte) (Message, error)
}
