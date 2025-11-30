package block

import (
	"yunion.io/x/cloudmux/pkg/multicloud/azure/vhdcore/bat"
	"yunion.io/x/cloudmux/pkg/multicloud/azure/vhdcore/footer"
	"yunion.io/x/cloudmux/pkg/multicloud/azure/vhdcore/header"
	"yunion.io/x/cloudmux/pkg/multicloud/azure/vhdcore/reader"
)

// FactoryParams represents type of the parameter for different disk block
// factories.
type FactoryParams struct {
	VhdFooter            *footer.Footer
	VhdHeader            *header.Header
	BlockAllocationTable *bat.BlockAllocationTable
	VhdReader            *reader.VhdReader
	ParentBlockFactory   Factory
}
