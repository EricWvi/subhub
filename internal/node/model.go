package node

import "time"

type Node struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateNodeInput struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

type UpdateNodeInput struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}
