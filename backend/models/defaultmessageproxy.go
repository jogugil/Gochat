package models

import (
	"backend/entities"
	"backend/utils"
	"fmt"
)

type DefaultMessageProxy struct {
	transformer entities.MessageTransformer
}

func (p *DefaultMessageProxy) TransformToExternal(msg entities.Message) ([]byte, error) {
	return p.transformer.ToExternal(msg)
}

func (p *DefaultMessageProxy) TransformFromExternal(rawMsg []byte) (entities.Message, error) {
	return p.transformer.FromExternal(rawMsg)
}

func NewMessageProxy(config utils.Config) (entities.MessageProxy, error) {
	switch config.MessagingType {
	case "kafka":
		return &DefaultMessageProxy{transformer: NewKafkaTransformer()}, nil
	case "nats":
		return &DefaultMessageProxy{transformer: NewNatsTransformer()}, nil
	default:
		return nil, fmt.Errorf("unsupported messaging type: %s", config.MessagingType)
	}
}
