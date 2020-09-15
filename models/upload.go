package models

import (
	"time"
)

type Upload struct {
	OriginalCid string    `bson:"original_cid,omitempty" json:"original_cid" validate:"required"`
	Uid         string    `bson:"uid,omitempty" json:"uid" validate:"required"`
	Filename    string    `bson:"filename,omitempty" json:"filename" validate:"required"`
	Status      string    `bson:"status,omitempty" json:"status" validate:"required"`
	Size        int64     `bson:"size,omitempty" json:"size" validate:"required"`
	Duration    float64   `bson:"duration,omitempty" json:"duration"`
	Percentage  uint      `bson:"percentage,omitempty" json:"percentage" validate:"required"`
	Hls         string    `bson:"hls,omitempty" json:"hls,omitempty"`
	CreatedAt   time.Time `bson:"created_at,omitempty"`
	UpdatedAt   time.Time `bson:"updated_at,omitempty"`
}

const (
	UPLOAD_STATUS_PENDING string = "pending"
)
