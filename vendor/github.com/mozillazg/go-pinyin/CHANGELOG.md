# Changelog

## [0.19.0] (2021-12-11)
* **Changed** 使用 [pinyin-data][pinyin-data] v0.12.0 的拼音数据


## [0.18.0] (2020-06-13)
* **Changed** 使用 [pinyin-data][pinyin-data] v0.9.0 的拼音数据
* **Bugfixed** 修复自定义的 Fallback 函数可能会导致结果乱码的问题 Fixes [#35]

## [0.17.0] (2020-04-09)

* **Changed** 因为依赖的 gojieba 经常出现安装异常，撤销 v0.16.0 的修改，撤销后 v0.17.0 的代码跟 v0.15.0 基本是一样的。
  如果有需要使用 v0.16.0 新增的 ``func Paragraph(p string) string`` 功能的请使用 v0.16.0 版本或者通过 v0.16.0 中相关代码实现类似的需求。

## [0.16.0] (2019-12-05)

* **NEW** 增加 ``func Paragraph(p string) string`` 用于便捷处理大段文字
(thanks [@huacnlee] via [#37][#37])

## [0.15.0] (2019-04-06)

* **Changed** 使用 [pinyin-data][pinyin-data] v0.7.0 的拼音数据
* **NEW** 添加 go.mod 文件


## [0.14.0] (2018-08-05)

* **Changed** 使用 [pinyin-data][pinyin-data] v0.6.1 的拼音数据
* **Changed** 命令行工具移到 `cmd/pinyin/` 目录下，现在需要改为使用
  `go get -u github.com/mozillazg/go-pinyin/cmd/pinyin` 来安装命令行工具。


## [0.13.0] (2018-04-29)

* **Changed** 使用 [pinyin-data][pinyin-data] v0.5.1 的拼音数据 (via [#30])
* **Changed** 修改命令行工具 `-s` 参数的值(thanks [@wdscxsj][@wdscxsj] via [#19][#19]):
    * `Normal` 改为 `zhao`
    * `Tone` 改为 `zh4ao`
    * `Tone2` 改为 `zha4o`
    * `Tone3` 改为 `zhao4`
    * `Initials` 改为 `zh`
    * `FirstLetter` 改为 `z`
    * `Finals` 改为 `ao`
    * `FinalsTone` 改为 `4ao`
    * `FinalsTone2` 改为 `a4o`
    * `FinalsTone3` 改为 `ao4`
* **Changed** 严格限制命令行参数中 `-s` 选项的值(thanks [@wdscxsj][@wdscxsj] via [#20][#20]):


## [0.12.0] (2017-04-25)


* **NEW** 命令行程序支持通过 -s 指定新增的 `Tone3` 和 `FinalsTone3` 拼音风格

        $ pinyin -s Tone3 请至少输入一个汉字
        qing3 zhi4 shao3 shu1 ru4 yi1 ge4 han4 zi4

        $ pinyin -s FinalsTone3 请至少输入一个汉字
        ing3 i4 ao3 u1 u4 i1 e4 an4 i4

* **Changed** use [pinyin-data](https://github.com/mozillazg/pinyin-data) v0.4.1


## [0.11.0] (2016-10-28)

* **Changed** 不再使用 `0` 表示轻声（因为之前并没有正确的实现这个功能, 同时也觉得这个功能没必要）。
  顺便修复了 Tone2 中 `ü` 标轻声的问题（像 `侵略 -> qi1n lv0e4` ）
* **NEW** 新增 `Tone3` 和 `FinalsTone3` 拼音风格。

        hans := "中国人"
        args := pinyin.NewArgs()
        args.Style = pinyin.Tone3
        fmt.Println("Tone3:", pinyin.Pinyin(hans, args))
        // Output: Tone3: [[zhong1] [guo2] [ren2]]

        args.Style = pinyin.FinalsTone3
        fmt.Println("FinalsTone3:", pinyin.Pinyin(hans, args))
        // Output: FinalsTone3: [[ong1] [uo2] [en2]]



## [0.10.0] (2016-10-18)

* **Changed** use [pinyin-data](https://github.com/mozillazg/pinyin-data) v0.4.0


## [0.9.0] (2016-09-04):

* **NEW** 新增 `func Convert(s string, a *Args) [][]string`
* **NEW** 新增 `func LazyConvert(s string, a *Args) []string`

之所以增加这两个函数是希望 `a` 参数支持 `nil`



## [0.8.0] (2016-08-19)

* **Changed** use [pinyin-data](https://github.com/mozillazg/pinyin-data) v0.3.0
  * Fixed [#13](https://github.com/mozillazg/go-pinyin/issues/13) . thanks [@aisq2008](https://github.com/aisq2008)
  * Fixed pinyin of 罗


## [0.7.0] (2016-08-02)

* **Changed** use [pinyin-data](https://github.com/mozillazg/pinyin-data) v0.2.0
* **Improved** golint and gofmt


## [0.6.0] (2016-05-14)

* **NEW** 命令行程序支持指定拼音风格:

  ```shell
  $ pinyin -s Normal 你好
  ni hao
  ```
* **Bugfixed** 解决韵母 i, u, ü 的问题：根据以下拼音方案，还原出正确的韵母
   [#8](https://github.com/mozillazg/go-pinyin/pull/8),  [python-pinyin#26](https://github.com/mozillazg/python-pinyin/pull/26)

    > i 行的韵母，前面没有声母的时候，写成：yi（衣），yɑ（呀），ye（耶），
    > yɑo（腰），you（忧），yɑn（烟），yin（因），yɑnɡ（央），yinɡ（英），yonɡ（雍）。
    >
    > u 行的韵母，前面没有声母的时候，写成wu（乌），wɑ（蛙），wo（窝），
    > wɑi（歪），wei（威），wɑn（弯），wen（温），wɑnɡ（汪），wenɡ（翁）。
    >
    > ü行的韵母跟声母j，q，x拼的时候，写成ju（居），qu（区），xu（虚），
    > ü上两点也省略；但是跟声母l，n拼的时候，仍然写成lü（吕），nü（女）。

    **注意** `y` 既不是声母也不是韵母。详见 [汉语拼音方案](http://www.edu.cn/20011114/3009777.shtml)

* **Bugfixed** 解决未正确处理鼻音 ḿ, ń, ň, ǹ 的问题：包含鼻音的拼音不应该有声母



## [0.5.0] (2016-03-12)

* **CHANGE** 改为使用来自 [pinyin-data](https://github.com/mozillazg/pinyin-data) 的拼音数据。
* **NEW** 命令行程序支持从标准输入读取数据（支持管道和重定向输入）:

  ```shell
  $ echo "你好" | pinyin
  nǐ hǎo
  $ pinyin < hello.txt
  nǐ hǎo
  ```


## [0.4.0] (2016-01-29)

* **NEW** `Args` 结构体新增 field: `Fallback func(r rune, a Args) []string`
  用于处理没有拼音的字符（默认忽略没有拼音的字符）:
  ```go
  a := pinyin.NewArgs()
  a.Fallback = func(r rune, a pinyin.Args) []string {
      return []string{string(r + 1)}
  }
  fmt.Println(pinyin.Pinyin("中国人abc", a))
  // Output: [[zhong] [guo] [ren] [b] [c] [d]]

  // or
  pinyin.Fallback = func(r rune, a pinyin.Args) []string {
      return []string{string(r)}
  }
  fmt.Println(pinyin.Pinyin("中国人abc", pinyin.NewArgs()))
  // Output: [[zhong] [guo] [ren] [a] [b] [c]]
  ```


## [0.3.0] (2015-12-29)

* fix "当字符串中有非中文的时候，会出现下标越界的情况"(影响 `pinyin.LazyPinyin` 和 `pinyin.Slug` ([#1](https://github.com/mozillazg/go-pinyin/issues/1)))
* 调整对非中文字符的处理：当遇到没有拼音的字符时，直接忽略
  ```go
  // before
  fmt.Println(pinyin.Pinyin("中国人abc", pinyin.NewArgs()))
  [[zhong] [guo] [ren] [] [] []]

  // after
  fmt.Println(pinyin.Pinyin("中国人abc", pinyin.NewArgs()))
  [[zhong] [guo] [ren]]
  ```


## [0.2.1] (2015-08-26)

* `yu`, `y`, `w` 不是声母


## [0.2.0] (2015-01-04)

* 新增 `func NewArgs() Args`
* 解决 `Args.Separator` 无法赋值为 `""` 的 BUG
* 规范命名:
    * `NORMAL` -> `Normal`
    * `TONE` -> `Tone`
    * `TONE2` -> `Tone2`
    * `INITIALS` -> `Initials`
    * `FIRST_LETTER` -> `FirstLetter`
    * `FINALS` -> `Finals`
    * `FINALS_TONE` -> `FinalsTone`
    * `FINALS_TONE2` -> `FinalsTone2`

## [0.1.1] (2014-12-07)
* 更新拼音库


## 0.1.0 (2014-11-23)
* Initial Release


[pinyin-data]: https://github.com/mozillazg/pinyin-data
[@wdscxsj]: https://github.com/wdscxsj
[@huacnlee]: https://github.com/huacnlee
[#19]: https://github.com/mozillazg/go-pinyin/pull/19
[#20]: https://github.com/mozillazg/go-pinyin/pull/20
[#30]: https://github.com/mozillazg/go-pinyin/pull/30
[#37]: https://github.com/mozillazg/go-pinyin/pull/37
[#35]: https://github.com/mozillazg/go-pinyin/issues/35

[0.1.1]: https://github.com/mozillazg/go-pinyin/compare/v0.1.0...v0.1.1
[0.2.0]: https://github.com/mozillazg/go-pinyin/compare/v0.1.1...v0.2.0
[0.2.1]: https://github.com/mozillazg/go-pinyin/compare/v0.2.0...v0.2.1
[0.3.0]: https://github.com/mozillazg/go-pinyin/compare/v0.2.1...v0.3.0
[0.4.0]: https://github.com/mozillazg/go-pinyin/compare/v0.3.0...v0.4.0
[0.5.0]: https://github.com/mozillazg/go-pinyin/compare/v0.4.0...v0.5.0
[0.6.0]: https://github.com/mozillazg/go-pinyin/compare/v0.5.0...v0.6.0
[0.7.0]: https://github.com/mozillazg/go-pinyin/compare/v0.6.0...v0.7.0
[0.8.0]: https://github.com/mozillazg/go-pinyin/compare/v0.7.0...v0.8.0
[0.9.0]: https://github.com/mozillazg/go-pinyin/compare/v0.8.0...v0.9.0
[0.10.0]: https://github.com/mozillazg/go-pinyin/compare/v0.9.0...v0.10.0
[0.11.0]: https://github.com/mozillazg/go-pinyin/compare/v0.10.0...v0.11.0
[0.12.0]: https://github.com/mozillazg/go-pinyin/compare/v0.11.0...v0.12.0
[0.13.0]: https://github.com/mozillazg/go-pinyin/compare/v0.12.0...v0.13.0
[0.14.0]: https://github.com/mozillazg/go-pinyin/compare/v0.13.0...v0.14.0
[0.15.0]: https://github.com/mozillazg/go-pinyin/compare/v0.14.0...v0.15.0
[0.16.0]: https://github.com/mozillazg/go-pinyin/compare/v0.15.0...v0.16.0
[0.17.0]: https://github.com/mozillazg/go-pinyin/compare/v0.16.0...v0.17.0
[0.18.0]: https://github.com/mozillazg/go-pinyin/compare/v0.17.0...v0.18.0
[0.19.0]: https://github.com/mozillazg/go-pinyin/compare/v0.18.0...v0.19.0
