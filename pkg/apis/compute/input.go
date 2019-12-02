package compute

type RegionalResourceCreateInput struct {
	Cloudregion   string `json:"cloudregion"`
	CloudregionId string `json:"cloudregion_id"`
}

type ManagedResourceCreateInput struct {
	Manager   string `json:"manager"`
	ManagerId string `json:"manager_id"`
}

type DeletePreventableCreateInput struct {
	DisableDelete *bool `json:"disable_delete"`
}
