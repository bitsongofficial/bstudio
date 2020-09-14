package models

import (
	"time"
)

type Upload struct {
	OriginalCid string    `bson:"original_cid,omitempty" json:"original_cid" validate:"required"`
	Uid         string    `bson:"uid,omitempty" json:"uid" validate:"required"`
	Filename    string    `bson:"filename,omitempty" json:"filename" validate:"required"`
	Status      string    `bson:"status,omitempty" json:"status" validate:"required"`
	Percentage  uint      `bson:"percentage,omitempty" json:"percentage" validate:"required"`
	HlsCid      string    `bson:"hls_cid,omitempty" json:"hls_cid,omitempty"`
	CreatedAt   time.Time `bson:"created_at,omitempty"`
	UpdatedAt   time.Time `bson:"updated_at,omitempty"`
}
