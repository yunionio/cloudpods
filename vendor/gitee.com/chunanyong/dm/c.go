/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"io"
	"math"
)

type Dm_build_1009 struct {
	dm_build_1010 []byte
	dm_build_1011 int
}

func Dm_build_1012(dm_build_1013 int) *Dm_build_1009 {
	return &Dm_build_1009{make([]byte, 0, dm_build_1013), 0}
}

func Dm_build_1014(dm_build_1015 []byte) *Dm_build_1009 {
	return &Dm_build_1009{dm_build_1015, 0}
}

func (dm_build_1017 *Dm_build_1009) dm_build_1016(dm_build_1018 int) *Dm_build_1009 {

	dm_build_1019 := len(dm_build_1017.dm_build_1010)
	dm_build_1020 := cap(dm_build_1017.dm_build_1010)

	if dm_build_1019+dm_build_1018 <= dm_build_1020 {
		dm_build_1017.dm_build_1010 = dm_build_1017.dm_build_1010[:dm_build_1019+dm_build_1018]
	} else {

		var calCap = int64(math.Max(float64(2*dm_build_1020), float64(dm_build_1018+dm_build_1019)))

		nbuf := make([]byte, dm_build_1018+dm_build_1019, calCap)
		copy(nbuf, dm_build_1017.dm_build_1010)
		dm_build_1017.dm_build_1010 = nbuf
	}

	return dm_build_1017
}

func (dm_build_1022 *Dm_build_1009) Dm_build_1021() int {
	return len(dm_build_1022.dm_build_1010)
}

func (dm_build_1024 *Dm_build_1009) Dm_build_1023(dm_build_1025 int) *Dm_build_1009 {
	for i := dm_build_1025; i < len(dm_build_1024.dm_build_1010); i++ {
		dm_build_1024.dm_build_1010[i] = 0
	}
	dm_build_1024.dm_build_1010 = dm_build_1024.dm_build_1010[:dm_build_1025]
	return dm_build_1024
}

func (dm_build_1027 *Dm_build_1009) Dm_build_1026(dm_build_1028 int) *Dm_build_1009 {
	dm_build_1027.dm_build_1011 = dm_build_1028
	return dm_build_1027
}

func (dm_build_1030 *Dm_build_1009) Dm_build_1029() int {
	return dm_build_1030.dm_build_1011
}

func (dm_build_1032 *Dm_build_1009) Dm_build_1031(dm_build_1033 bool) int {
	return len(dm_build_1032.dm_build_1010) - dm_build_1032.dm_build_1011
}

func (dm_build_1035 *Dm_build_1009) Dm_build_1034(dm_build_1036 int, dm_build_1037 bool, dm_build_1038 bool) *Dm_build_1009 {

	if dm_build_1037 {
		if dm_build_1038 {
			dm_build_1035.dm_build_1016(dm_build_1036)
		} else {
			dm_build_1035.dm_build_1010 = dm_build_1035.dm_build_1010[:len(dm_build_1035.dm_build_1010)-dm_build_1036]
		}
	} else {
		if dm_build_1038 {
			dm_build_1035.dm_build_1011 += dm_build_1036
		} else {
			dm_build_1035.dm_build_1011 -= dm_build_1036
		}
	}

	return dm_build_1035
}

func (dm_build_1040 *Dm_build_1009) Dm_build_1039(dm_build_1041 io.Reader, dm_build_1042 int) (int, error) {
	dm_build_1043 := len(dm_build_1040.dm_build_1010)
	dm_build_1040.dm_build_1016(dm_build_1042)
	dm_build_1044 := 0
	for dm_build_1042 > 0 {
		n, err := dm_build_1041.Read(dm_build_1040.dm_build_1010[dm_build_1043+dm_build_1044:])
		if n > 0 && err == io.EOF {
			dm_build_1044 += n
			dm_build_1040.dm_build_1010 = dm_build_1040.dm_build_1010[:dm_build_1043+dm_build_1044]
			return dm_build_1044, nil
		} else if n > 0 && err == nil {
			dm_build_1042 -= n
			dm_build_1044 += n
		} else if n == 0 && err != nil {
			return -1, ECGO_COMMUNITION_ERROR.addDetailln(err.Error()).throw()
		}
	}

	return dm_build_1044, nil
}

func (dm_build_1046 *Dm_build_1009) Dm_build_1045(dm_build_1047 io.Writer) (*Dm_build_1009, error) {
	if _, err := dm_build_1047.Write(dm_build_1046.dm_build_1010); err != nil {
		return nil, ECGO_COMMUNITION_ERROR.addDetailln(err.Error()).throw()
	}
	return dm_build_1046, nil
}

func (dm_build_1049 *Dm_build_1009) Dm_build_1048(dm_build_1050 bool) int {
	dm_build_1051 := len(dm_build_1049.dm_build_1010)
	dm_build_1049.dm_build_1016(1)

	if dm_build_1050 {
		return copy(dm_build_1049.dm_build_1010[dm_build_1051:], []byte{1})
	} else {
		return copy(dm_build_1049.dm_build_1010[dm_build_1051:], []byte{0})
	}
}

func (dm_build_1053 *Dm_build_1009) Dm_build_1052(dm_build_1054 byte) int {
	dm_build_1055 := len(dm_build_1053.dm_build_1010)
	dm_build_1053.dm_build_1016(1)

	return copy(dm_build_1053.dm_build_1010[dm_build_1055:], Dm_build_650.Dm_build_828(dm_build_1054))
}

func (dm_build_1057 *Dm_build_1009) Dm_build_1056(dm_build_1058 int8) int {
	dm_build_1059 := len(dm_build_1057.dm_build_1010)
	dm_build_1057.dm_build_1016(1)

	return copy(dm_build_1057.dm_build_1010[dm_build_1059:], Dm_build_650.Dm_build_831(dm_build_1058))
}

func (dm_build_1061 *Dm_build_1009) Dm_build_1060(dm_build_1062 int16) int {
	dm_build_1063 := len(dm_build_1061.dm_build_1010)
	dm_build_1061.dm_build_1016(2)

	return copy(dm_build_1061.dm_build_1010[dm_build_1063:], Dm_build_650.Dm_build_834(dm_build_1062))
}

func (dm_build_1065 *Dm_build_1009) Dm_build_1064(dm_build_1066 int32) int {
	dm_build_1067 := len(dm_build_1065.dm_build_1010)
	dm_build_1065.dm_build_1016(4)

	return copy(dm_build_1065.dm_build_1010[dm_build_1067:], Dm_build_650.Dm_build_837(dm_build_1066))
}

func (dm_build_1069 *Dm_build_1009) Dm_build_1068(dm_build_1070 uint8) int {
	dm_build_1071 := len(dm_build_1069.dm_build_1010)
	dm_build_1069.dm_build_1016(1)

	return copy(dm_build_1069.dm_build_1010[dm_build_1071:], Dm_build_650.Dm_build_849(dm_build_1070))
}

func (dm_build_1073 *Dm_build_1009) Dm_build_1072(dm_build_1074 uint16) int {
	dm_build_1075 := len(dm_build_1073.dm_build_1010)
	dm_build_1073.dm_build_1016(2)

	return copy(dm_build_1073.dm_build_1010[dm_build_1075:], Dm_build_650.Dm_build_852(dm_build_1074))
}

func (dm_build_1077 *Dm_build_1009) Dm_build_1076(dm_build_1078 uint32) int {
	dm_build_1079 := len(dm_build_1077.dm_build_1010)
	dm_build_1077.dm_build_1016(4)

	return copy(dm_build_1077.dm_build_1010[dm_build_1079:], Dm_build_650.Dm_build_855(dm_build_1078))
}

func (dm_build_1081 *Dm_build_1009) Dm_build_1080(dm_build_1082 uint64) int {
	dm_build_1083 := len(dm_build_1081.dm_build_1010)
	dm_build_1081.dm_build_1016(8)

	return copy(dm_build_1081.dm_build_1010[dm_build_1083:], Dm_build_650.Dm_build_858(dm_build_1082))
}

func (dm_build_1085 *Dm_build_1009) Dm_build_1084(dm_build_1086 float32) int {
	dm_build_1087 := len(dm_build_1085.dm_build_1010)
	dm_build_1085.dm_build_1016(4)

	return copy(dm_build_1085.dm_build_1010[dm_build_1087:], Dm_build_650.Dm_build_855(math.Float32bits(dm_build_1086)))
}

func (dm_build_1089 *Dm_build_1009) Dm_build_1088(dm_build_1090 float64) int {
	dm_build_1091 := len(dm_build_1089.dm_build_1010)
	dm_build_1089.dm_build_1016(8)

	return copy(dm_build_1089.dm_build_1010[dm_build_1091:], Dm_build_650.Dm_build_858(math.Float64bits(dm_build_1090)))
}

func (dm_build_1093 *Dm_build_1009) Dm_build_1092(dm_build_1094 []byte) int {
	dm_build_1095 := len(dm_build_1093.dm_build_1010)
	dm_build_1093.dm_build_1016(len(dm_build_1094))
	return copy(dm_build_1093.dm_build_1010[dm_build_1095:], dm_build_1094)
}

func (dm_build_1097 *Dm_build_1009) Dm_build_1096(dm_build_1098 []byte) int {
	return dm_build_1097.Dm_build_1064(int32(len(dm_build_1098))) + dm_build_1097.Dm_build_1092(dm_build_1098)
}

func (dm_build_1100 *Dm_build_1009) Dm_build_1099(dm_build_1101 []byte) int {
	return dm_build_1100.Dm_build_1068(uint8(len(dm_build_1101))) + dm_build_1100.Dm_build_1092(dm_build_1101)
}

func (dm_build_1103 *Dm_build_1009) Dm_build_1102(dm_build_1104 []byte) int {
	return dm_build_1103.Dm_build_1072(uint16(len(dm_build_1104))) + dm_build_1103.Dm_build_1092(dm_build_1104)
}

func (dm_build_1106 *Dm_build_1009) Dm_build_1105(dm_build_1107 []byte) int {
	return dm_build_1106.Dm_build_1092(dm_build_1107) + dm_build_1106.Dm_build_1052(0)
}

func (dm_build_1109 *Dm_build_1009) Dm_build_1108(dm_build_1110 string, dm_build_1111 string, dm_build_1112 *DmConnection) int {
	dm_build_1113 := Dm_build_650.Dm_build_866(dm_build_1110, dm_build_1111, dm_build_1112)
	return dm_build_1109.Dm_build_1096(dm_build_1113)
}

func (dm_build_1115 *Dm_build_1009) Dm_build_1114(dm_build_1116 string, dm_build_1117 string, dm_build_1118 *DmConnection) int {
	dm_build_1119 := Dm_build_650.Dm_build_866(dm_build_1116, dm_build_1117, dm_build_1118)
	return dm_build_1115.Dm_build_1099(dm_build_1119)
}

func (dm_build_1121 *Dm_build_1009) Dm_build_1120(dm_build_1122 string, dm_build_1123 string, dm_build_1124 *DmConnection) int {
	dm_build_1125 := Dm_build_650.Dm_build_866(dm_build_1122, dm_build_1123, dm_build_1124)
	return dm_build_1121.Dm_build_1102(dm_build_1125)
}

func (dm_build_1127 *Dm_build_1009) Dm_build_1126(dm_build_1128 string, dm_build_1129 string, dm_build_1130 *DmConnection) int {
	dm_build_1131 := Dm_build_650.Dm_build_866(dm_build_1128, dm_build_1129, dm_build_1130)
	return dm_build_1127.Dm_build_1105(dm_build_1131)
}

func (dm_build_1133 *Dm_build_1009) Dm_build_1132() byte {
	dm_build_1134 := Dm_build_650.Dm_build_743(dm_build_1133.dm_build_1010, dm_build_1133.dm_build_1011)
	dm_build_1133.dm_build_1011++
	return dm_build_1134
}

func (dm_build_1136 *Dm_build_1009) Dm_build_1135() int16 {
	dm_build_1137 := Dm_build_650.Dm_build_747(dm_build_1136.dm_build_1010, dm_build_1136.dm_build_1011)
	dm_build_1136.dm_build_1011 += 2
	return dm_build_1137
}

func (dm_build_1139 *Dm_build_1009) Dm_build_1138() int32 {
	dm_build_1140 := Dm_build_650.Dm_build_752(dm_build_1139.dm_build_1010, dm_build_1139.dm_build_1011)
	dm_build_1139.dm_build_1011 += 4
	return dm_build_1140
}

func (dm_build_1142 *Dm_build_1009) Dm_build_1141() int64 {
	dm_build_1143 := Dm_build_650.Dm_build_757(dm_build_1142.dm_build_1010, dm_build_1142.dm_build_1011)
	dm_build_1142.dm_build_1011 += 8
	return dm_build_1143
}

func (dm_build_1145 *Dm_build_1009) Dm_build_1144() float32 {
	dm_build_1146 := Dm_build_650.Dm_build_762(dm_build_1145.dm_build_1010, dm_build_1145.dm_build_1011)
	dm_build_1145.dm_build_1011 += 4
	return dm_build_1146
}

func (dm_build_1148 *Dm_build_1009) Dm_build_1147() float64 {
	dm_build_1149 := Dm_build_650.Dm_build_766(dm_build_1148.dm_build_1010, dm_build_1148.dm_build_1011)
	dm_build_1148.dm_build_1011 += 8
	return dm_build_1149
}

func (dm_build_1151 *Dm_build_1009) Dm_build_1150() uint8 {
	dm_build_1152 := Dm_build_650.Dm_build_770(dm_build_1151.dm_build_1010, dm_build_1151.dm_build_1011)
	dm_build_1151.dm_build_1011 += 1
	return dm_build_1152
}

func (dm_build_1154 *Dm_build_1009) Dm_build_1153() uint16 {
	dm_build_1155 := Dm_build_650.Dm_build_774(dm_build_1154.dm_build_1010, dm_build_1154.dm_build_1011)
	dm_build_1154.dm_build_1011 += 2
	return dm_build_1155
}

func (dm_build_1157 *Dm_build_1009) Dm_build_1156() uint32 {
	dm_build_1158 := Dm_build_650.Dm_build_779(dm_build_1157.dm_build_1010, dm_build_1157.dm_build_1011)
	dm_build_1157.dm_build_1011 += 4
	return dm_build_1158
}

func (dm_build_1160 *Dm_build_1009) Dm_build_1159(dm_build_1161 int) []byte {
	dm_build_1162 := Dm_build_650.Dm_build_801(dm_build_1160.dm_build_1010, dm_build_1160.dm_build_1011, dm_build_1161)
	dm_build_1160.dm_build_1011 += dm_build_1161
	return dm_build_1162
}

func (dm_build_1164 *Dm_build_1009) Dm_build_1163() []byte {
	return dm_build_1164.Dm_build_1159(int(dm_build_1164.Dm_build_1138()))
}

func (dm_build_1166 *Dm_build_1009) Dm_build_1165() []byte {
	return dm_build_1166.Dm_build_1159(int(dm_build_1166.Dm_build_1132()))
}

func (dm_build_1168 *Dm_build_1009) Dm_build_1167() []byte {
	return dm_build_1168.Dm_build_1159(int(dm_build_1168.Dm_build_1135()))
}

func (dm_build_1170 *Dm_build_1009) Dm_build_1169(dm_build_1171 int) []byte {
	return dm_build_1170.Dm_build_1159(dm_build_1171)
}

func (dm_build_1173 *Dm_build_1009) Dm_build_1172() []byte {
	dm_build_1174 := 0
	for dm_build_1173.Dm_build_1132() != 0 {
		dm_build_1174++
	}
	dm_build_1173.Dm_build_1034(dm_build_1174, false, false)
	return dm_build_1173.Dm_build_1159(dm_build_1174)
}

func (dm_build_1176 *Dm_build_1009) Dm_build_1175(dm_build_1177 int, dm_build_1178 string, dm_build_1179 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1176.Dm_build_1159(dm_build_1177), dm_build_1178, dm_build_1179)
}

func (dm_build_1181 *Dm_build_1009) Dm_build_1180(dm_build_1182 string, dm_build_1183 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1181.Dm_build_1163(), dm_build_1182, dm_build_1183)
}

func (dm_build_1185 *Dm_build_1009) Dm_build_1184(dm_build_1186 string, dm_build_1187 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1185.Dm_build_1165(), dm_build_1186, dm_build_1187)
}

func (dm_build_1189 *Dm_build_1009) Dm_build_1188(dm_build_1190 string, dm_build_1191 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1189.Dm_build_1167(), dm_build_1190, dm_build_1191)
}

func (dm_build_1193 *Dm_build_1009) Dm_build_1192(dm_build_1194 string, dm_build_1195 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1193.Dm_build_1172(), dm_build_1194, dm_build_1195)
}

func (dm_build_1197 *Dm_build_1009) Dm_build_1196(dm_build_1198 int, dm_build_1199 byte) int {
	return dm_build_1197.Dm_build_1232(dm_build_1198, Dm_build_650.Dm_build_828(dm_build_1199))
}

func (dm_build_1201 *Dm_build_1009) Dm_build_1200(dm_build_1202 int, dm_build_1203 int16) int {
	return dm_build_1201.Dm_build_1232(dm_build_1202, Dm_build_650.Dm_build_834(dm_build_1203))
}

func (dm_build_1205 *Dm_build_1009) Dm_build_1204(dm_build_1206 int, dm_build_1207 int32) int {
	return dm_build_1205.Dm_build_1232(dm_build_1206, Dm_build_650.Dm_build_837(dm_build_1207))
}

func (dm_build_1209 *Dm_build_1009) Dm_build_1208(dm_build_1210 int, dm_build_1211 int64) int {
	return dm_build_1209.Dm_build_1232(dm_build_1210, Dm_build_650.Dm_build_840(dm_build_1211))
}

func (dm_build_1213 *Dm_build_1009) Dm_build_1212(dm_build_1214 int, dm_build_1215 float32) int {
	return dm_build_1213.Dm_build_1232(dm_build_1214, Dm_build_650.Dm_build_843(dm_build_1215))
}

func (dm_build_1217 *Dm_build_1009) Dm_build_1216(dm_build_1218 int, dm_build_1219 float64) int {
	return dm_build_1217.Dm_build_1232(dm_build_1218, Dm_build_650.Dm_build_846(dm_build_1219))
}

func (dm_build_1221 *Dm_build_1009) Dm_build_1220(dm_build_1222 int, dm_build_1223 uint8) int {
	return dm_build_1221.Dm_build_1232(dm_build_1222, Dm_build_650.Dm_build_849(dm_build_1223))
}

func (dm_build_1225 *Dm_build_1009) Dm_build_1224(dm_build_1226 int, dm_build_1227 uint16) int {
	return dm_build_1225.Dm_build_1232(dm_build_1226, Dm_build_650.Dm_build_852(dm_build_1227))
}

func (dm_build_1229 *Dm_build_1009) Dm_build_1228(dm_build_1230 int, dm_build_1231 uint32) int {
	return dm_build_1229.Dm_build_1232(dm_build_1230, Dm_build_650.Dm_build_855(dm_build_1231))
}

func (dm_build_1233 *Dm_build_1009) Dm_build_1232(dm_build_1234 int, dm_build_1235 []byte) int {
	return copy(dm_build_1233.dm_build_1010[dm_build_1234:], dm_build_1235)
}

func (dm_build_1237 *Dm_build_1009) Dm_build_1236(dm_build_1238 int, dm_build_1239 []byte) int {
	return dm_build_1237.Dm_build_1204(dm_build_1238, int32(len(dm_build_1239))) + dm_build_1237.Dm_build_1232(dm_build_1238+4, dm_build_1239)
}

func (dm_build_1241 *Dm_build_1009) Dm_build_1240(dm_build_1242 int, dm_build_1243 []byte) int {
	return dm_build_1241.Dm_build_1196(dm_build_1242, byte(len(dm_build_1243))) + dm_build_1241.Dm_build_1232(dm_build_1242+1, dm_build_1243)
}

func (dm_build_1245 *Dm_build_1009) Dm_build_1244(dm_build_1246 int, dm_build_1247 []byte) int {
	return dm_build_1245.Dm_build_1200(dm_build_1246, int16(len(dm_build_1247))) + dm_build_1245.Dm_build_1232(dm_build_1246+2, dm_build_1247)
}

func (dm_build_1249 *Dm_build_1009) Dm_build_1248(dm_build_1250 int, dm_build_1251 []byte) int {
	return dm_build_1249.Dm_build_1232(dm_build_1250, dm_build_1251) + dm_build_1249.Dm_build_1196(dm_build_1250+len(dm_build_1251), 0)
}

func (dm_build_1253 *Dm_build_1009) Dm_build_1252(dm_build_1254 int, dm_build_1255 string, dm_build_1256 string, dm_build_1257 *DmConnection) int {
	return dm_build_1253.Dm_build_1236(dm_build_1254, Dm_build_650.Dm_build_866(dm_build_1255, dm_build_1256, dm_build_1257))
}

func (dm_build_1259 *Dm_build_1009) Dm_build_1258(dm_build_1260 int, dm_build_1261 string, dm_build_1262 string, dm_build_1263 *DmConnection) int {
	return dm_build_1259.Dm_build_1240(dm_build_1260, Dm_build_650.Dm_build_866(dm_build_1261, dm_build_1262, dm_build_1263))
}

func (dm_build_1265 *Dm_build_1009) Dm_build_1264(dm_build_1266 int, dm_build_1267 string, dm_build_1268 string, dm_build_1269 *DmConnection) int {
	return dm_build_1265.Dm_build_1244(dm_build_1266, Dm_build_650.Dm_build_866(dm_build_1267, dm_build_1268, dm_build_1269))
}

func (dm_build_1271 *Dm_build_1009) Dm_build_1270(dm_build_1272 int, dm_build_1273 string, dm_build_1274 string, dm_build_1275 *DmConnection) int {
	return dm_build_1271.Dm_build_1248(dm_build_1272, Dm_build_650.Dm_build_866(dm_build_1273, dm_build_1274, dm_build_1275))
}

func (dm_build_1277 *Dm_build_1009) Dm_build_1276(dm_build_1278 int) byte {
	return Dm_build_650.Dm_build_871(dm_build_1277.Dm_build_1303(dm_build_1278, 1))
}

func (dm_build_1280 *Dm_build_1009) Dm_build_1279(dm_build_1281 int) int16 {
	return Dm_build_650.Dm_build_874(dm_build_1280.Dm_build_1303(dm_build_1281, 2))
}

func (dm_build_1283 *Dm_build_1009) Dm_build_1282(dm_build_1284 int) int32 {
	return Dm_build_650.Dm_build_877(dm_build_1283.Dm_build_1303(dm_build_1284, 4))
}

func (dm_build_1286 *Dm_build_1009) Dm_build_1285(dm_build_1287 int) int64 {
	return Dm_build_650.Dm_build_880(dm_build_1286.Dm_build_1303(dm_build_1287, 8))
}

func (dm_build_1289 *Dm_build_1009) Dm_build_1288(dm_build_1290 int) float32 {
	return Dm_build_650.Dm_build_883(dm_build_1289.Dm_build_1303(dm_build_1290, 4))
}

func (dm_build_1292 *Dm_build_1009) Dm_build_1291(dm_build_1293 int) float64 {
	return Dm_build_650.Dm_build_886(dm_build_1292.Dm_build_1303(dm_build_1293, 8))
}

func (dm_build_1295 *Dm_build_1009) Dm_build_1294(dm_build_1296 int) uint8 {
	return Dm_build_650.Dm_build_889(dm_build_1295.Dm_build_1303(dm_build_1296, 1))
}

func (dm_build_1298 *Dm_build_1009) Dm_build_1297(dm_build_1299 int) uint16 {
	return Dm_build_650.Dm_build_892(dm_build_1298.Dm_build_1303(dm_build_1299, 2))
}

func (dm_build_1301 *Dm_build_1009) Dm_build_1300(dm_build_1302 int) uint32 {
	return Dm_build_650.Dm_build_895(dm_build_1301.Dm_build_1303(dm_build_1302, 4))
}

func (dm_build_1304 *Dm_build_1009) Dm_build_1303(dm_build_1305 int, dm_build_1306 int) []byte {
	return dm_build_1304.dm_build_1010[dm_build_1305 : dm_build_1305+dm_build_1306]
}

func (dm_build_1308 *Dm_build_1009) Dm_build_1307(dm_build_1309 int) []byte {
	dm_build_1310 := dm_build_1308.Dm_build_1282(dm_build_1309)
	return dm_build_1308.Dm_build_1303(dm_build_1309+4, int(dm_build_1310))
}

func (dm_build_1312 *Dm_build_1009) Dm_build_1311(dm_build_1313 int) []byte {
	dm_build_1314 := dm_build_1312.Dm_build_1276(dm_build_1313)
	return dm_build_1312.Dm_build_1303(dm_build_1313+1, int(dm_build_1314))
}

func (dm_build_1316 *Dm_build_1009) Dm_build_1315(dm_build_1317 int) []byte {
	dm_build_1318 := dm_build_1316.Dm_build_1279(dm_build_1317)
	return dm_build_1316.Dm_build_1303(dm_build_1317+2, int(dm_build_1318))
}

func (dm_build_1320 *Dm_build_1009) Dm_build_1319(dm_build_1321 int) []byte {
	dm_build_1322 := 0
	for dm_build_1320.Dm_build_1276(dm_build_1321) != 0 {
		dm_build_1321++
		dm_build_1322++
	}

	return dm_build_1320.Dm_build_1303(dm_build_1321-dm_build_1322, int(dm_build_1322))
}

func (dm_build_1324 *Dm_build_1009) Dm_build_1323(dm_build_1325 int, dm_build_1326 string, dm_build_1327 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1324.Dm_build_1307(dm_build_1325), dm_build_1326, dm_build_1327)
}

func (dm_build_1329 *Dm_build_1009) Dm_build_1328(dm_build_1330 int, dm_build_1331 string, dm_build_1332 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1329.Dm_build_1311(dm_build_1330), dm_build_1331, dm_build_1332)
}

func (dm_build_1334 *Dm_build_1009) Dm_build_1333(dm_build_1335 int, dm_build_1336 string, dm_build_1337 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1334.Dm_build_1315(dm_build_1335), dm_build_1336, dm_build_1337)
}

func (dm_build_1339 *Dm_build_1009) Dm_build_1338(dm_build_1340 int, dm_build_1341 string, dm_build_1342 *DmConnection) string {
	return Dm_build_650.Dm_build_902(dm_build_1339.Dm_build_1319(dm_build_1340), dm_build_1341, dm_build_1342)
}
