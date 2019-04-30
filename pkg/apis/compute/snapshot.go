package compute

import "yunion.io/x/onecloud/pkg/apis"

type SSnapshotCreateInput struct {
	apis.Meta

	Name      string `json:"name"`
	ProjectId string `json:"project_id"`

	DiskId        string `json:"disk_id"`
	StorageId     string `json:"storage_id"`
	CreatedBy     string `json:"created_by"`
	Location      string `json:"location"`
	Size          int    `json:"size"`
	DiskType      string `json:"disk_type"`
	CloudregionId string `json:"cloudregion_id"`
}
