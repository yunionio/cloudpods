go-pinyin
=========

[![Build Status](https://github.com/mozillazg/go-pinyin/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/mozillazg/go-pinyin/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/mozillazg/go-pinyin/badge.svg?branch=master)](https://coveralls.io/r/mozillazg/go-pinyin?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/mozillazg/go-pinyin)](https://goreportcard.com/report/github.com/mozillazg/go-pinyin)
[![GoDoc](https://godoc.org/github.com/mozillazg/go-pinyin?status.svg)](https://godoc.org/github.com/mozillazg/go-pinyin)

汉语拼音转换工具 Go 版。


Installation
------------

```
go get -u github.com/mozillazg/go-pinyin
```

install CLI tool:

```
go get -u github.com/mozillazg/go-pinyin/cmd/pinyin
$ pinyin 中国人
zhōng guó rén
```


Documentation
--------------

API documentation can be found here:
https://godoc.org/github.com/mozillazg/go-pinyin


Usage
------

```go
package main

import (
	"fmt"
	"github.com/mozillazg/go-pinyin"
)

func main() {
	hans := "中国人"

	// 默认
	a := pinyin.NewArgs()
	fmt.Println(pinyin.Pinyin(hans, a))
	// [[zhong] [guo] [ren]]

	// 包含声调
	a.Style = pinyin.Tone
	fmt.Println(pinyin.Pinyin(hans, a))
	// [[zhōng] [guó] [rén]]

	// 声调用数字表示
	a.Style = pinyin.Tone2
	fmt.Println(pinyin.Pinyin(hans, a))
	// [[zho1ng] [guo2] [re2n]]

	// 开启多音字模式
	a = pinyin.NewArgs()
	a.Heteronym = true
	fmt.Println(pinyin.Pinyin(hans, a))
	// [[zhong zhong] [guo] [ren]]
	a.Style = pinyin.Tone2
	fmt.Println(pinyin.Pinyin(hans, a))
	// [[zho1ng zho4ng] [guo2] [re2n]]

	fmt.Println(pinyin.LazyPinyin(hans, pinyin.NewArgs()))
	// [zhong guo ren]

	fmt.Println(pinyin.Convert(hans, nil))
	// [[zhong] [guo] [ren]]

	fmt.Println(pinyin.LazyConvert(hans, nil))
	// [zhong guo ren]
}
```

注意：

* 默认情况下会忽略没有拼音的字符（可以通过自定义 `Fallback` 参数的值来自定义如何处理没有拼音的字符，
  详见 [示例](https://godoc.org/github.com/mozillazg/go-pinyin#example-Pinyin--FallbackCustom1)）。
* 根据 [《汉语拼音方案》](http://www.moe.gov.cn/s78/A19/yxs_left/moe_810/s230/195802/t19580201_186000.html) y，w，ü (yu) 都不是声母，
  以及不是所有拼音都有声母，如果这不是你预期的话，你可能需要的是首字母风格 `FirstLetter`
（ [详细信息](https://github.com/mozillazg/python-pinyin#%E4%B8%BA%E4%BB%80%E4%B9%88%E6%B2%A1%E6%9C%89-y-w-yu-%E5%87%A0%E4%B8%AA%E5%A3%B0%E6%AF%8D) ）。


Related Projects
-----------------

* [hotoo/pinyin](https://github.com/hotoo/pinyin): 汉语拼音转换工具 Node.js/JavaScript 版。
* [mozillazg/python-pinyin](https://github.com/mozillazg/python-pinyin): 汉语拼音转换工具 Python 版。
* [mozillazg/rust-pinyin](https://github.com/mozillazg/rust-pinyin): 汉语拼音转换工具 Rust 版。


pinyin data
-----------------

* 使用 [pinyin-data](https://github.com/mozillazg/pinyin-data) 的拼音数据


License
---------

Under the MIT License.
