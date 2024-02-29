/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package util

const (
	LINE_SEPARATOR = "\n"
)

// 执行f并忽略panic
func AbsorbPanic(f func()){
	defer func() {
		if p := recover(); p != nil {
			// TODO do something
		}
	}()
	f()
}

func SliceEquals(src []byte, dest []byte) bool {
	if len(src) != len(dest) {
		return false
	}

	for i, _ := range src {
		if src[i] != dest[i] {
			return false
		}
	}

	return true
}

// 获取两个数的最大公约数，由调用者确保m、n>=0；如果m或n为0，返回1
func GCD(m int32, n int32) int32 {
	if m == 0 || n == 0 {
		return 1
	}
	r := m % n
	m = n
	n = r
	if r == 0 {
		return m
	} else {
		return GCD(m, n)
	}
}

// 返回切片中所有数的累加值
func Sum(arr []int32) int32 {
	var sum int32 = 0
	for _, i := range arr {
		sum += i
	}
	return sum
}
