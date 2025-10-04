package models

import "time"

type ConversationResult struct {
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UUID            string    `db:"uuid" json:"uuid"`
	ReferenceNumber string    `db:"reference_number" json:"reference_number"`
	Subject         string    `db:"subject" json:"subject"`
}

type MessageResult struct {
	CreatedAt                   time.Time `db:"created_at" json:"created_at"`
	TextContent                 string    `db:"text_content" json:"text_content"`
	ConversationCreatedAt       time.Time `db:"conversation_created_at" json:"conversation_created_at"`
	ConversationUUID            string    `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationReferenceNumber string    `db:"conversation_reference_number" json:"conversation_reference_number"`
}

type ContactResult struct {
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Email     string    `db:"email" json:"email"`
}
