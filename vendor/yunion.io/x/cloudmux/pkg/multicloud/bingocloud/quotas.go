package bingocloud

type SQuotas struct {
	OwnerId    string
	Resource   string
	ResourceEn string
	ResourceZh string
	HardLimit  int
	InUse      int
}
