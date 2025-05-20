package models

import "time"

type EventModel struct {
	ID          int
	Date        time.Time
	Description string
	Image       []byte
}
