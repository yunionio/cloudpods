/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"container/list"
	"io"
)

type Dm_build_931 struct {
	dm_build_932 *list.List
	dm_build_933 *dm_build_985
	dm_build_934 int
}

func Dm_build_935() *Dm_build_931 {
	return &Dm_build_931{
		dm_build_932: list.New(),
		dm_build_934: 0,
	}
}

func (dm_build_937 *Dm_build_931) Dm_build_936() int {
	return dm_build_937.dm_build_934
}

func (dm_build_939 *Dm_build_931) Dm_build_938(dm_build_940 *Dm_build_1009, dm_build_941 int) int {
	var dm_build_942 = 0
	var dm_build_943 = 0
	for dm_build_942 < dm_build_941 && dm_build_939.dm_build_933 != nil {
		dm_build_943 = dm_build_939.dm_build_933.dm_build_993(dm_build_940, dm_build_941-dm_build_942)
		if dm_build_939.dm_build_933.dm_build_988 == 0 {
			dm_build_939.dm_build_975()
		}
		dm_build_942 += dm_build_943
		dm_build_939.dm_build_934 -= dm_build_943
	}
	return dm_build_942
}

func (dm_build_945 *Dm_build_931) Dm_build_944(dm_build_946 []byte, dm_build_947 int, dm_build_948 int) int {
	var dm_build_949 = 0
	var dm_build_950 = 0
	for dm_build_949 < dm_build_948 && dm_build_945.dm_build_933 != nil {
		dm_build_950 = dm_build_945.dm_build_933.dm_build_997(dm_build_946, dm_build_947, dm_build_948-dm_build_949)
		if dm_build_945.dm_build_933.dm_build_988 == 0 {
			dm_build_945.dm_build_975()
		}
		dm_build_949 += dm_build_950
		dm_build_945.dm_build_934 -= dm_build_950
		dm_build_947 += dm_build_950
	}
	return dm_build_949
}

func (dm_build_952 *Dm_build_931) Dm_build_951(dm_build_953 io.Writer, dm_build_954 int) int {
	var dm_build_955 = 0
	var dm_build_956 = 0
	for dm_build_955 < dm_build_954 && dm_build_952.dm_build_933 != nil {
		dm_build_956 = dm_build_952.dm_build_933.dm_build_1002(dm_build_953, dm_build_954-dm_build_955)
		if dm_build_952.dm_build_933.dm_build_988 == 0 {
			dm_build_952.dm_build_975()
		}
		dm_build_955 += dm_build_956
		dm_build_952.dm_build_934 -= dm_build_956
	}
	return dm_build_955
}

func (dm_build_958 *Dm_build_931) Dm_build_957(dm_build_959 []byte, dm_build_960 int, dm_build_961 int) {
	if dm_build_961 == 0 {
		return
	}
	var dm_build_962 = dm_build_989(dm_build_959, dm_build_960, dm_build_961)
	if dm_build_958.dm_build_933 == nil {
		dm_build_958.dm_build_933 = dm_build_962
	} else {
		dm_build_958.dm_build_932.PushBack(dm_build_962)
	}
	dm_build_958.dm_build_934 += dm_build_961
}

func (dm_build_964 *Dm_build_931) dm_build_963(dm_build_965 int) byte {
	var dm_build_966 = dm_build_965
	var dm_build_967 = dm_build_964.dm_build_933
	for dm_build_966 > 0 && dm_build_967 != nil {
		if dm_build_967.dm_build_988 == 0 {
			continue
		}
		if dm_build_966 > dm_build_967.dm_build_988-1 {
			dm_build_966 -= dm_build_967.dm_build_988
			dm_build_967 = dm_build_964.dm_build_932.Front().Value.(*dm_build_985)
		} else {
			break
		}
	}
	return dm_build_967.dm_build_1006(dm_build_966)
}
func (dm_build_969 *Dm_build_931) Dm_build_968(dm_build_970 *Dm_build_931) {
	if dm_build_970.dm_build_934 == 0 {
		return
	}
	var dm_build_971 = dm_build_970.dm_build_933
	for dm_build_971 != nil {
		dm_build_969.dm_build_972(dm_build_971)
		dm_build_970.dm_build_975()
		dm_build_971 = dm_build_970.dm_build_933
	}
	dm_build_970.dm_build_934 = 0
}
func (dm_build_973 *Dm_build_931) dm_build_972(dm_build_974 *dm_build_985) {
	if dm_build_974.dm_build_988 == 0 {
		return
	}
	if dm_build_973.dm_build_933 == nil {
		dm_build_973.dm_build_933 = dm_build_974
	} else {
		dm_build_973.dm_build_932.PushBack(dm_build_974)
	}
	dm_build_973.dm_build_934 += dm_build_974.dm_build_988
}

func (dm_build_976 *Dm_build_931) dm_build_975() {
	var dm_build_977 = dm_build_976.dm_build_932.Front()
	if dm_build_977 == nil {
		dm_build_976.dm_build_933 = nil
	} else {
		dm_build_976.dm_build_933 = dm_build_977.Value.(*dm_build_985)
		dm_build_976.dm_build_932.Remove(dm_build_977)
	}
}

func (dm_build_979 *Dm_build_931) Dm_build_978() []byte {
	var dm_build_980 = make([]byte, dm_build_979.dm_build_934)
	var dm_build_981 = dm_build_979.dm_build_933
	var dm_build_982 = 0
	var dm_build_983 = len(dm_build_980)
	var dm_build_984 = 0
	for dm_build_981 != nil {
		if dm_build_981.dm_build_988 > 0 {
			if dm_build_983 > dm_build_981.dm_build_988 {
				dm_build_984 = dm_build_981.dm_build_988
			} else {
				dm_build_984 = dm_build_983
			}
			copy(dm_build_980[dm_build_982:dm_build_982+dm_build_984], dm_build_981.dm_build_986[dm_build_981.dm_build_987:dm_build_981.dm_build_987+dm_build_984])
			dm_build_982 += dm_build_984
			dm_build_983 -= dm_build_984
		}
		if dm_build_979.dm_build_932.Front() == nil {
			dm_build_981 = nil
		} else {
			dm_build_981 = dm_build_979.dm_build_932.Front().Value.(*dm_build_985)
		}
	}
	return dm_build_980
}

type dm_build_985 struct {
	dm_build_986 []byte
	dm_build_987 int
	dm_build_988 int
}

func dm_build_989(dm_build_990 []byte, dm_build_991 int, dm_build_992 int) *dm_build_985 {
	return &dm_build_985{
		dm_build_990,
		dm_build_991,
		dm_build_992,
	}
}

func (dm_build_994 *dm_build_985) dm_build_993(dm_build_995 *Dm_build_1009, dm_build_996 int) int {
	if dm_build_994.dm_build_988 <= dm_build_996 {
		dm_build_996 = dm_build_994.dm_build_988
	}
	dm_build_995.Dm_build_1092(dm_build_994.dm_build_986[dm_build_994.dm_build_987 : dm_build_994.dm_build_987+dm_build_996])
	dm_build_994.dm_build_987 += dm_build_996
	dm_build_994.dm_build_988 -= dm_build_996
	return dm_build_996
}

func (dm_build_998 *dm_build_985) dm_build_997(dm_build_999 []byte, dm_build_1000 int, dm_build_1001 int) int {
	if dm_build_998.dm_build_988 <= dm_build_1001 {
		dm_build_1001 = dm_build_998.dm_build_988
	}
	copy(dm_build_999[dm_build_1000:dm_build_1000+dm_build_1001], dm_build_998.dm_build_986[dm_build_998.dm_build_987:dm_build_998.dm_build_987+dm_build_1001])
	dm_build_998.dm_build_987 += dm_build_1001
	dm_build_998.dm_build_988 -= dm_build_1001
	return dm_build_1001
}

func (dm_build_1003 *dm_build_985) dm_build_1002(dm_build_1004 io.Writer, dm_build_1005 int) int {
	if dm_build_1003.dm_build_988 <= dm_build_1005 {
		dm_build_1005 = dm_build_1003.dm_build_988
	}
	dm_build_1004.Write(dm_build_1003.dm_build_986[dm_build_1003.dm_build_987 : dm_build_1003.dm_build_987+dm_build_1005])
	dm_build_1003.dm_build_987 += dm_build_1005
	dm_build_1003.dm_build_988 -= dm_build_1005
	return dm_build_1005
}
func (dm_build_1007 *dm_build_985) dm_build_1006(dm_build_1008 int) byte {
	return dm_build_1007.dm_build_986[dm_build_1007.dm_build_987+dm_build_1008]
}
