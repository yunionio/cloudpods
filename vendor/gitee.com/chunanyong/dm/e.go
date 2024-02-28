/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"bytes"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"math"
)

type dm_build_649 struct{}

var Dm_build_650 = &dm_build_649{}

func (Dm_build_652 *dm_build_649) Dm_build_651(dm_build_653 []byte, dm_build_654 int, dm_build_655 byte) int {
	dm_build_653[dm_build_654] = dm_build_655
	return 1
}

func (Dm_build_657 *dm_build_649) Dm_build_656(dm_build_658 []byte, dm_build_659 int, dm_build_660 int8) int {
	dm_build_658[dm_build_659] = byte(dm_build_660)
	return 1
}

func (Dm_build_662 *dm_build_649) Dm_build_661(dm_build_663 []byte, dm_build_664 int, dm_build_665 int16) int {
	dm_build_663[dm_build_664] = byte(dm_build_665)
	dm_build_664++
	dm_build_663[dm_build_664] = byte(dm_build_665 >> 8)
	return 2
}

func (Dm_build_667 *dm_build_649) Dm_build_666(dm_build_668 []byte, dm_build_669 int, dm_build_670 int32) int {
	dm_build_668[dm_build_669] = byte(dm_build_670)
	dm_build_669++
	dm_build_668[dm_build_669] = byte(dm_build_670 >> 8)
	dm_build_669++
	dm_build_668[dm_build_669] = byte(dm_build_670 >> 16)
	dm_build_669++
	dm_build_668[dm_build_669] = byte(dm_build_670 >> 24)
	dm_build_669++
	return 4
}

func (Dm_build_672 *dm_build_649) Dm_build_671(dm_build_673 []byte, dm_build_674 int, dm_build_675 int64) int {
	dm_build_673[dm_build_674] = byte(dm_build_675)
	dm_build_674++
	dm_build_673[dm_build_674] = byte(dm_build_675 >> 8)
	dm_build_674++
	dm_build_673[dm_build_674] = byte(dm_build_675 >> 16)
	dm_build_674++
	dm_build_673[dm_build_674] = byte(dm_build_675 >> 24)
	dm_build_674++
	dm_build_673[dm_build_674] = byte(dm_build_675 >> 32)
	dm_build_674++
	dm_build_673[dm_build_674] = byte(dm_build_675 >> 40)
	dm_build_674++
	dm_build_673[dm_build_674] = byte(dm_build_675 >> 48)
	dm_build_674++
	dm_build_673[dm_build_674] = byte(dm_build_675 >> 56)
	return 8
}

func (Dm_build_677 *dm_build_649) Dm_build_676(dm_build_678 []byte, dm_build_679 int, dm_build_680 float32) int {
	return Dm_build_677.Dm_build_696(dm_build_678, dm_build_679, math.Float32bits(dm_build_680))
}

func (Dm_build_682 *dm_build_649) Dm_build_681(dm_build_683 []byte, dm_build_684 int, dm_build_685 float64) int {
	return Dm_build_682.Dm_build_701(dm_build_683, dm_build_684, math.Float64bits(dm_build_685))
}

func (Dm_build_687 *dm_build_649) Dm_build_686(dm_build_688 []byte, dm_build_689 int, dm_build_690 uint8) int {
	dm_build_688[dm_build_689] = byte(dm_build_690)
	return 1
}

func (Dm_build_692 *dm_build_649) Dm_build_691(dm_build_693 []byte, dm_build_694 int, dm_build_695 uint16) int {
	dm_build_693[dm_build_694] = byte(dm_build_695)
	dm_build_694++
	dm_build_693[dm_build_694] = byte(dm_build_695 >> 8)
	return 2
}

func (Dm_build_697 *dm_build_649) Dm_build_696(dm_build_698 []byte, dm_build_699 int, dm_build_700 uint32) int {
	dm_build_698[dm_build_699] = byte(dm_build_700)
	dm_build_699++
	dm_build_698[dm_build_699] = byte(dm_build_700 >> 8)
	dm_build_699++
	dm_build_698[dm_build_699] = byte(dm_build_700 >> 16)
	dm_build_699++
	dm_build_698[dm_build_699] = byte(dm_build_700 >> 24)
	return 3
}

func (Dm_build_702 *dm_build_649) Dm_build_701(dm_build_703 []byte, dm_build_704 int, dm_build_705 uint64) int {
	dm_build_703[dm_build_704] = byte(dm_build_705)
	dm_build_704++
	dm_build_703[dm_build_704] = byte(dm_build_705 >> 8)
	dm_build_704++
	dm_build_703[dm_build_704] = byte(dm_build_705 >> 16)
	dm_build_704++
	dm_build_703[dm_build_704] = byte(dm_build_705 >> 24)
	dm_build_704++
	dm_build_703[dm_build_704] = byte(dm_build_705 >> 32)
	dm_build_704++
	dm_build_703[dm_build_704] = byte(dm_build_705 >> 40)
	dm_build_704++
	dm_build_703[dm_build_704] = byte(dm_build_705 >> 48)
	dm_build_704++
	dm_build_703[dm_build_704] = byte(dm_build_705 >> 56)
	return 3
}

func (Dm_build_707 *dm_build_649) Dm_build_706(dm_build_708 []byte, dm_build_709 int, dm_build_710 []byte, dm_build_711 int, dm_build_712 int) int {
	copy(dm_build_708[dm_build_709:dm_build_709+dm_build_712], dm_build_710[dm_build_711:dm_build_711+dm_build_712])
	return dm_build_712
}

func (Dm_build_714 *dm_build_649) Dm_build_713(dm_build_715 []byte, dm_build_716 int, dm_build_717 []byte, dm_build_718 int, dm_build_719 int) int {
	dm_build_716 += Dm_build_714.Dm_build_696(dm_build_715, dm_build_716, uint32(dm_build_719))
	return 4 + Dm_build_714.Dm_build_706(dm_build_715, dm_build_716, dm_build_717, dm_build_718, dm_build_719)
}

func (Dm_build_721 *dm_build_649) Dm_build_720(dm_build_722 []byte, dm_build_723 int, dm_build_724 []byte, dm_build_725 int, dm_build_726 int) int {
	dm_build_723 += Dm_build_721.Dm_build_691(dm_build_722, dm_build_723, uint16(dm_build_726))
	return 2 + Dm_build_721.Dm_build_706(dm_build_722, dm_build_723, dm_build_724, dm_build_725, dm_build_726)
}

func (Dm_build_728 *dm_build_649) Dm_build_727(dm_build_729 []byte, dm_build_730 int, dm_build_731 string, dm_build_732 string, dm_build_733 *DmConnection) int {
	dm_build_734 := Dm_build_728.Dm_build_866(dm_build_731, dm_build_732, dm_build_733)
	dm_build_730 += Dm_build_728.Dm_build_696(dm_build_729, dm_build_730, uint32(len(dm_build_734)))
	return 4 + Dm_build_728.Dm_build_706(dm_build_729, dm_build_730, dm_build_734, 0, len(dm_build_734))
}

func (Dm_build_736 *dm_build_649) Dm_build_735(dm_build_737 []byte, dm_build_738 int, dm_build_739 string, dm_build_740 string, dm_build_741 *DmConnection) int {
	dm_build_742 := Dm_build_736.Dm_build_866(dm_build_739, dm_build_740, dm_build_741)

	dm_build_738 += Dm_build_736.Dm_build_691(dm_build_737, dm_build_738, uint16(len(dm_build_742)))
	return 2 + Dm_build_736.Dm_build_706(dm_build_737, dm_build_738, dm_build_742, 0, len(dm_build_742))
}

func (Dm_build_744 *dm_build_649) Dm_build_743(dm_build_745 []byte, dm_build_746 int) byte {
	return dm_build_745[dm_build_746]
}

func (Dm_build_748 *dm_build_649) Dm_build_747(dm_build_749 []byte, dm_build_750 int) int16 {
	var dm_build_751 int16
	dm_build_751 = int16(dm_build_749[dm_build_750] & 0xff)
	dm_build_750++
	dm_build_751 |= int16(dm_build_749[dm_build_750]&0xff) << 8
	return dm_build_751
}

func (Dm_build_753 *dm_build_649) Dm_build_752(dm_build_754 []byte, dm_build_755 int) int32 {
	var dm_build_756 int32
	dm_build_756 = int32(dm_build_754[dm_build_755] & 0xff)
	dm_build_755++
	dm_build_756 |= int32(dm_build_754[dm_build_755]&0xff) << 8
	dm_build_755++
	dm_build_756 |= int32(dm_build_754[dm_build_755]&0xff) << 16
	dm_build_755++
	dm_build_756 |= int32(dm_build_754[dm_build_755]&0xff) << 24
	return dm_build_756
}

func (Dm_build_758 *dm_build_649) Dm_build_757(dm_build_759 []byte, dm_build_760 int) int64 {
	var dm_build_761 int64
	dm_build_761 = int64(dm_build_759[dm_build_760] & 0xff)
	dm_build_760++
	dm_build_761 |= int64(dm_build_759[dm_build_760]&0xff) << 8
	dm_build_760++
	dm_build_761 |= int64(dm_build_759[dm_build_760]&0xff) << 16
	dm_build_760++
	dm_build_761 |= int64(dm_build_759[dm_build_760]&0xff) << 24
	dm_build_760++
	dm_build_761 |= int64(dm_build_759[dm_build_760]&0xff) << 32
	dm_build_760++
	dm_build_761 |= int64(dm_build_759[dm_build_760]&0xff) << 40
	dm_build_760++
	dm_build_761 |= int64(dm_build_759[dm_build_760]&0xff) << 48
	dm_build_760++
	dm_build_761 |= int64(dm_build_759[dm_build_760]&0xff) << 56
	return dm_build_761
}

func (Dm_build_763 *dm_build_649) Dm_build_762(dm_build_764 []byte, dm_build_765 int) float32 {
	return math.Float32frombits(Dm_build_763.Dm_build_779(dm_build_764, dm_build_765))
}

func (Dm_build_767 *dm_build_649) Dm_build_766(dm_build_768 []byte, dm_build_769 int) float64 {
	return math.Float64frombits(Dm_build_767.Dm_build_784(dm_build_768, dm_build_769))
}

func (Dm_build_771 *dm_build_649) Dm_build_770(dm_build_772 []byte, dm_build_773 int) uint8 {
	return uint8(dm_build_772[dm_build_773] & 0xff)
}

func (Dm_build_775 *dm_build_649) Dm_build_774(dm_build_776 []byte, dm_build_777 int) uint16 {
	var dm_build_778 uint16
	dm_build_778 = uint16(dm_build_776[dm_build_777] & 0xff)
	dm_build_777++
	dm_build_778 |= uint16(dm_build_776[dm_build_777]&0xff) << 8
	return dm_build_778
}

func (Dm_build_780 *dm_build_649) Dm_build_779(dm_build_781 []byte, dm_build_782 int) uint32 {
	var dm_build_783 uint32
	dm_build_783 = uint32(dm_build_781[dm_build_782] & 0xff)
	dm_build_782++
	dm_build_783 |= uint32(dm_build_781[dm_build_782]&0xff) << 8
	dm_build_782++
	dm_build_783 |= uint32(dm_build_781[dm_build_782]&0xff) << 16
	dm_build_782++
	dm_build_783 |= uint32(dm_build_781[dm_build_782]&0xff) << 24
	return dm_build_783
}

func (Dm_build_785 *dm_build_649) Dm_build_784(dm_build_786 []byte, dm_build_787 int) uint64 {
	var dm_build_788 uint64
	dm_build_788 = uint64(dm_build_786[dm_build_787] & 0xff)
	dm_build_787++
	dm_build_788 |= uint64(dm_build_786[dm_build_787]&0xff) << 8
	dm_build_787++
	dm_build_788 |= uint64(dm_build_786[dm_build_787]&0xff) << 16
	dm_build_787++
	dm_build_788 |= uint64(dm_build_786[dm_build_787]&0xff) << 24
	dm_build_787++
	dm_build_788 |= uint64(dm_build_786[dm_build_787]&0xff) << 32
	dm_build_787++
	dm_build_788 |= uint64(dm_build_786[dm_build_787]&0xff) << 40
	dm_build_787++
	dm_build_788 |= uint64(dm_build_786[dm_build_787]&0xff) << 48
	dm_build_787++
	dm_build_788 |= uint64(dm_build_786[dm_build_787]&0xff) << 56
	return dm_build_788
}

func (Dm_build_790 *dm_build_649) Dm_build_789(dm_build_791 []byte, dm_build_792 int) []byte {
	dm_build_793 := Dm_build_790.Dm_build_779(dm_build_791, dm_build_792)

	dm_build_794 := make([]byte, dm_build_793)
	copy(dm_build_794[:int(dm_build_793)], dm_build_791[dm_build_792+4:dm_build_792+4+int(dm_build_793)])
	return dm_build_794
}

func (Dm_build_796 *dm_build_649) Dm_build_795(dm_build_797 []byte, dm_build_798 int) []byte {
	dm_build_799 := Dm_build_796.Dm_build_774(dm_build_797, dm_build_798)

	dm_build_800 := make([]byte, dm_build_799)
	copy(dm_build_800[:int(dm_build_799)], dm_build_797[dm_build_798+2:dm_build_798+2+int(dm_build_799)])
	return dm_build_800
}

func (Dm_build_802 *dm_build_649) Dm_build_801(dm_build_803 []byte, dm_build_804 int, dm_build_805 int) []byte {

	dm_build_806 := make([]byte, dm_build_805)
	copy(dm_build_806[:dm_build_805], dm_build_803[dm_build_804:dm_build_804+dm_build_805])
	return dm_build_806
}

func (Dm_build_808 *dm_build_649) Dm_build_807(dm_build_809 []byte, dm_build_810 int, dm_build_811 int, dm_build_812 string, dm_build_813 *DmConnection) string {
	return Dm_build_808.Dm_build_902(dm_build_809[dm_build_810:dm_build_810+dm_build_811], dm_build_812, dm_build_813)
}

func (Dm_build_815 *dm_build_649) Dm_build_814(dm_build_816 []byte, dm_build_817 int, dm_build_818 string, dm_build_819 *DmConnection) string {
	dm_build_820 := Dm_build_815.Dm_build_779(dm_build_816, dm_build_817)
	dm_build_817 += 4
	return Dm_build_815.Dm_build_807(dm_build_816, dm_build_817, int(dm_build_820), dm_build_818, dm_build_819)
}

func (Dm_build_822 *dm_build_649) Dm_build_821(dm_build_823 []byte, dm_build_824 int, dm_build_825 string, dm_build_826 *DmConnection) string {
	dm_build_827 := Dm_build_822.Dm_build_774(dm_build_823, dm_build_824)
	dm_build_824 += 2
	return Dm_build_822.Dm_build_807(dm_build_823, dm_build_824, int(dm_build_827), dm_build_825, dm_build_826)
}

func (Dm_build_829 *dm_build_649) Dm_build_828(dm_build_830 byte) []byte {
	return []byte{dm_build_830}
}

func (Dm_build_832 *dm_build_649) Dm_build_831(dm_build_833 int8) []byte {
	return []byte{byte(dm_build_833)}
}

func (Dm_build_835 *dm_build_649) Dm_build_834(dm_build_836 int16) []byte {
	return []byte{byte(dm_build_836), byte(dm_build_836 >> 8)}
}

func (Dm_build_838 *dm_build_649) Dm_build_837(dm_build_839 int32) []byte {
	return []byte{byte(dm_build_839), byte(dm_build_839 >> 8), byte(dm_build_839 >> 16), byte(dm_build_839 >> 24)}
}

func (Dm_build_841 *dm_build_649) Dm_build_840(dm_build_842 int64) []byte {
	return []byte{byte(dm_build_842), byte(dm_build_842 >> 8), byte(dm_build_842 >> 16), byte(dm_build_842 >> 24), byte(dm_build_842 >> 32),
		byte(dm_build_842 >> 40), byte(dm_build_842 >> 48), byte(dm_build_842 >> 56)}
}

func (Dm_build_844 *dm_build_649) Dm_build_843(dm_build_845 float32) []byte {
	return Dm_build_844.Dm_build_855(math.Float32bits(dm_build_845))
}

func (Dm_build_847 *dm_build_649) Dm_build_846(dm_build_848 float64) []byte {
	return Dm_build_847.Dm_build_858(math.Float64bits(dm_build_848))
}

func (Dm_build_850 *dm_build_649) Dm_build_849(dm_build_851 uint8) []byte {
	return []byte{byte(dm_build_851)}
}

func (Dm_build_853 *dm_build_649) Dm_build_852(dm_build_854 uint16) []byte {
	return []byte{byte(dm_build_854), byte(dm_build_854 >> 8)}
}

func (Dm_build_856 *dm_build_649) Dm_build_855(dm_build_857 uint32) []byte {
	return []byte{byte(dm_build_857), byte(dm_build_857 >> 8), byte(dm_build_857 >> 16), byte(dm_build_857 >> 24)}
}

func (Dm_build_859 *dm_build_649) Dm_build_858(dm_build_860 uint64) []byte {
	return []byte{byte(dm_build_860), byte(dm_build_860 >> 8), byte(dm_build_860 >> 16), byte(dm_build_860 >> 24), byte(dm_build_860 >> 32), byte(dm_build_860 >> 40), byte(dm_build_860 >> 48), byte(dm_build_860 >> 56)}
}

func (Dm_build_862 *dm_build_649) Dm_build_861(dm_build_863 []byte, dm_build_864 string, dm_build_865 *DmConnection) []byte {
	if dm_build_864 == "UTF-8" {
		return dm_build_863
	}

	if dm_build_865 == nil {
		if e := dm_build_907(dm_build_864); e != nil {
			tmp, err := ioutil.ReadAll(
				transform.NewReader(bytes.NewReader(dm_build_863), e.NewEncoder()),
			)
			if err != nil {
				panic("UTF8 To Charset error!")
			}

			return tmp
		}

		panic("Unsupported Charset!")
	}

	if dm_build_865.encodeBuffer == nil {
		dm_build_865.encodeBuffer = bytes.NewBuffer(nil)
		dm_build_865.encode = dm_build_907(dm_build_865.getServerEncoding())
		dm_build_865.transformReaderDst = make([]byte, 4096)
		dm_build_865.transformReaderSrc = make([]byte, 4096)
	}

	if e := dm_build_865.encode; e != nil {

		dm_build_865.encodeBuffer.Reset()

		n, err := dm_build_865.encodeBuffer.ReadFrom(
			Dm_build_921(bytes.NewReader(dm_build_863), e.NewEncoder(), dm_build_865.transformReaderDst, dm_build_865.transformReaderSrc),
		)
		if err != nil {
			panic("UTF8 To Charset error!")
		}
		var tmp = make([]byte, n)
		if _, err = dm_build_865.encodeBuffer.Read(tmp); err != nil {
			panic("UTF8 To Charset error!")
		}
		return tmp
	}

	panic("Unsupported Charset!")
}

func (Dm_build_867 *dm_build_649) Dm_build_866(dm_build_868 string, dm_build_869 string, dm_build_870 *DmConnection) []byte {
	return Dm_build_867.Dm_build_861([]byte(dm_build_868), dm_build_869, dm_build_870)
}

func (Dm_build_872 *dm_build_649) Dm_build_871(dm_build_873 []byte) byte {
	return Dm_build_872.Dm_build_743(dm_build_873, 0)
}

func (Dm_build_875 *dm_build_649) Dm_build_874(dm_build_876 []byte) int16 {
	return Dm_build_875.Dm_build_747(dm_build_876, 0)
}

func (Dm_build_878 *dm_build_649) Dm_build_877(dm_build_879 []byte) int32 {
	return Dm_build_878.Dm_build_752(dm_build_879, 0)
}

func (Dm_build_881 *dm_build_649) Dm_build_880(dm_build_882 []byte) int64 {
	return Dm_build_881.Dm_build_757(dm_build_882, 0)
}

func (Dm_build_884 *dm_build_649) Dm_build_883(dm_build_885 []byte) float32 {
	return Dm_build_884.Dm_build_762(dm_build_885, 0)
}

func (Dm_build_887 *dm_build_649) Dm_build_886(dm_build_888 []byte) float64 {
	return Dm_build_887.Dm_build_766(dm_build_888, 0)
}

func (Dm_build_890 *dm_build_649) Dm_build_889(dm_build_891 []byte) uint8 {
	return Dm_build_890.Dm_build_770(dm_build_891, 0)
}

func (Dm_build_893 *dm_build_649) Dm_build_892(dm_build_894 []byte) uint16 {
	return Dm_build_893.Dm_build_774(dm_build_894, 0)
}

func (Dm_build_896 *dm_build_649) Dm_build_895(dm_build_897 []byte) uint32 {
	return Dm_build_896.Dm_build_779(dm_build_897, 0)
}

func (Dm_build_899 *dm_build_649) Dm_build_898(dm_build_900 []byte, dm_build_901 string) []byte {
	if dm_build_901 == "UTF-8" {
		return dm_build_900
	}

	if e := dm_build_907(dm_build_901); e != nil {

		tmp, err := ioutil.ReadAll(
			transform.NewReader(bytes.NewReader(dm_build_900), e.NewDecoder()),
		)
		if err != nil {

			panic("Charset To UTF8 error!")
		}

		return tmp
	}

	panic("Unsupported Charset!")

}

func (Dm_build_903 *dm_build_649) Dm_build_902(dm_build_904 []byte, dm_build_905 string, dm_build_906 *DmConnection) string {
	return string(Dm_build_903.Dm_build_898(dm_build_904, dm_build_905))
}

func dm_build_907(dm_build_908 string) encoding.Encoding {
	if e, err := ianaindex.MIB.Encoding(dm_build_908); err == nil && e != nil {
		return e
	}
	return nil
}

type Dm_build_909 struct {
	dm_build_910 io.Reader
	dm_build_911 transform.Transformer
	dm_build_912 error

	dm_build_913               []byte
	dm_build_914, dm_build_915 int

	dm_build_916               []byte
	dm_build_917, dm_build_918 int

	dm_build_919 bool
}

const dm_build_920 = 4096

func Dm_build_921(dm_build_922 io.Reader, dm_build_923 transform.Transformer, dm_build_924 []byte, dm_build_925 []byte) *Dm_build_909 {
	dm_build_923.Reset()
	return &Dm_build_909{
		dm_build_910: dm_build_922,
		dm_build_911: dm_build_923,
		dm_build_913: dm_build_924,
		dm_build_916: dm_build_925,
	}
}

func (dm_build_927 *Dm_build_909) Read(dm_build_928 []byte) (int, error) {
	dm_build_929, dm_build_930 := 0, error(nil)
	for {

		if dm_build_927.dm_build_914 != dm_build_927.dm_build_915 {
			dm_build_929 = copy(dm_build_928, dm_build_927.dm_build_913[dm_build_927.dm_build_914:dm_build_927.dm_build_915])
			dm_build_927.dm_build_914 += dm_build_929
			if dm_build_927.dm_build_914 == dm_build_927.dm_build_915 && dm_build_927.dm_build_919 {
				return dm_build_929, dm_build_927.dm_build_912
			}
			return dm_build_929, nil
		} else if dm_build_927.dm_build_919 {
			return 0, dm_build_927.dm_build_912
		}

		if dm_build_927.dm_build_917 != dm_build_927.dm_build_918 || dm_build_927.dm_build_912 != nil {
			dm_build_927.dm_build_914 = 0
			dm_build_927.dm_build_915, dm_build_929, dm_build_930 = dm_build_927.dm_build_911.Transform(dm_build_927.dm_build_913, dm_build_927.dm_build_916[dm_build_927.dm_build_917:dm_build_927.dm_build_918], dm_build_927.dm_build_912 == io.EOF)
			dm_build_927.dm_build_917 += dm_build_929

			switch {
			case dm_build_930 == nil:
				if dm_build_927.dm_build_917 != dm_build_927.dm_build_918 {
					dm_build_927.dm_build_912 = nil
				}

				dm_build_927.dm_build_919 = dm_build_927.dm_build_912 != nil
				continue
			case dm_build_930 == transform.ErrShortDst && (dm_build_927.dm_build_915 != 0 || dm_build_929 != 0):

				continue
			case dm_build_930 == transform.ErrShortSrc && dm_build_927.dm_build_918-dm_build_927.dm_build_917 != len(dm_build_927.dm_build_916) && dm_build_927.dm_build_912 == nil:

			default:
				dm_build_927.dm_build_919 = true

				if dm_build_927.dm_build_912 == nil || dm_build_927.dm_build_912 == io.EOF {
					dm_build_927.dm_build_912 = dm_build_930
				}
				continue
			}
		}

		if dm_build_927.dm_build_917 != 0 {
			dm_build_927.dm_build_917, dm_build_927.dm_build_918 = 0, copy(dm_build_927.dm_build_916, dm_build_927.dm_build_916[dm_build_927.dm_build_917:dm_build_927.dm_build_918])
		}
		dm_build_929, dm_build_927.dm_build_912 = dm_build_927.dm_build_910.Read(dm_build_927.dm_build_916[dm_build_927.dm_build_918:])
		dm_build_927.dm_build_918 += dm_build_929
	}
}
