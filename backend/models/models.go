// Package models defines the shapes of documents stored in MongoDB.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"` // never sent to client
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}

type Link struct {
	ID        primitive.ObjectID `bson:"_id"`
	Code      string             `bson:"code"`
	URL       string             `bson:"url"`
	UserID    primitive.ObjectID `bson:"userId"`
	Clicks    int64              `bson:"clicks"`          // flushed from Redis
	Daily     map[string]int64   `bson:"daily,omitempty"` // "2026-11-13" -> clicks
	ExpiresAt *time.Time         `bson:"expiresAt,omitempty"`
	CreatedAt time.Time          `bson:"createdAt"`
}
