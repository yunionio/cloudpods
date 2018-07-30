package models

import (
	"github.com/jinzhu/gorm"
)

type Resourcer interface {
	DB() *gorm.DB
	TableName() string
	Model() interface{}
	Models() interface{}
}

type Modeler interface {
	UUID() string
}
