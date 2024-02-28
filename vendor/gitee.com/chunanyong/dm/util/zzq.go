/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package util

func Split(s string, sep string) []string {
	var foot = make([]int, len(s)) // 足够的元素个数
	var count, sLen, sepLen = 0, len(s), len(sep)
	for i := 0; i < sLen; i++ {
		// 处理 s == “-9999-1" && seperators == "-"情况
		if i == 0 && sLen >= sepLen {
			if s[0:sepLen] == sep {
				i += sepLen - 1
				continue
			}
		}
		for j := 0; j < sepLen; j++ {
			if s[i] == sep[j] {
				foot[count] = i
				count++
				break
			}
		}
	}
	var ret = make([]string, count+1)
	if count == 0 {
		ret[0] = s
		return ret
	}
	ret[0] = s[0:foot[0]]
	for i := 1; i < count; i++ {
		ret[i] = s[foot[i-1]+1 : foot[i]]
	}
	ret[count] = s[foot[count-1]+1:]
	return ret
}
