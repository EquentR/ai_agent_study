package model

type DataVersion struct {
	ID      int    `json:"id" gorm:"type:integer;not null;primaryKey;comment:ID"`
	Version string `json:"version" gorm:"type:varchar(32);not null;comment:ID"`
}
