package models

import (
	"backend/entities"
	"time"
)

type RoomMetrics struct {
	MessagesSent    int64     `json:"messagesSent"`
	UsersPerHour    int64     `json:"usersPerHour"`
	MessagesPerHour int64     `json:"messagesPerHour"`
	LastUpdate      time.Time `json:"lastUpdate"`
}

func (metrics *RoomMetrics) UpdateUserMetrics(user entities.User) {
	// Increment the number of active users
	metrics.UsersPerHour++
}

func (metrics *RoomMetrics) UpdateMessageMetrics(message entities.Message) {
	// Increment the number of messages sent
	metrics.MessagesPerHour++
	metrics.MessagesSent++
}

func (metrics *RoomMetrics) GetSummary() map[string]int64 {
	summary := make(map[string]int64)
	summary["messagesSent"] = metrics.MessagesSent
	summary["usersPerHour"] = metrics.UsersPerHour
	summary["messagesPerHour"] = metrics.MessagesPerHour
	return summary
}
