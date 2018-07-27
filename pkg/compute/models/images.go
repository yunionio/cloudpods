package models

import "time"

const (
	IMAGE_STATUS_ACTIVE  = "active"
	IMAGE_STATUS_QUEUED  = "queued"
	IMAGE_STATUS_KILLED  = "killed"
	IMAGE_STATUS_DELETED = "deleted"
)

type SImage struct {
	Checksum        string
	ContainerFormat string
	CreatedAt       time.Time
	Deleted         bool
	DiskFormat      string
	Id              string
	IsPublic        bool
	MinDisk         int
	MinRam          int
	Name            string
	Owner           string
	Properties      map[string]string
	Protected       bool
	Size            int
	Status          string
	UpdatedAt       time.Time
}
