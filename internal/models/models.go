package models

import "time"

// SubObj — к какому сателлиту ссылается main
type SubObj string

const (
	SubObjTools  SubObj = "tools"
	SubObjTables SubObj = "tables"
	SubObjChairs SubObj = "chairs"
)

type ChairType string

const (
	ChairTypeABC ChairType = "abc"
	ChairTypeCDE ChairType = "cde"
)

type Main struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	SubID     int        `json:"subId"`
	SubObj    SubObj     `json:"subObj"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdateAt  time.Time  `json:"updateAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	// поля сателита — заполняются при чтении
	Description1 *string    `json:"description1,omitempty"`
	Description2 *string    `json:"description2,omitempty"`
	Description3 *string    `json:"description3,omitempty"`
	Type         *ChairType `json:"type,omitempty"`
}

type MainInput struct {
	Title        string     `json:"title"`
	SubObj       SubObj     `json:"subObj"`
	Description1 *string    `json:"description1"`
	Description2 *string    `json:"description2"`
	Description3 *string    `json:"description3"`
	Type         *ChairType `json:"type"`
}
