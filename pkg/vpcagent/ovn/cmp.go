package ovn

import (
	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/ovsdb/types"

	"yunion.io/x/onecloud/pkg/vpcagent/ovnutil"
)

// cmp scans the database for irows.  For those present, mark them with ocver.
// If all rows are found, return true to indicate this.  Otherwise return as
// 2nd value the args to destroy these found records
func cmp(db *ovn_nb.OVNNorthbound, ocver string, irows ...types.IRow) (bool, []string) {
	irowsFound := make([]types.IRow, 0, len(irows))
	irowsDiff := make([]types.IRow, 0)

	for _, irow := range irows {
		irowFound := db.FindOneMatchNonZeros(irow)
		if irowFound != nil {
			irowsFound = append(irowsFound, irowFound)
		} else {
			if irowDiff := db.FindOneMatchByAnyIndex(irow); irowDiff != nil {
				irowsDiff = append(irowsDiff, irowDiff)
			}
		}
	}
	// mark them anyway even if not all found, to avoid the destroy
	// call at sweep stage
	for _, irowFound := range irowsFound {
		irowFound.SetExternalId(externalKeyOcVersion, ocver)
	}
	if len(irowsFound) == len(irows) {
		return true, nil
	}
	irowsCleanup := append(irowsFound, irowsDiff...)
	args := ovnutil.OvnNbctlArgsDestroy(irowsCleanup)
	return false, args
}
