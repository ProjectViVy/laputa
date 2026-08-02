package palace

import "time"

type Wing struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func NewWing(name string) *Wing {
	return &Wing{
		Name:      name,
		CreatedAt: time.Now(),
	}
}
