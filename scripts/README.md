Github Pull Request Helper Scripts
======================================

这里提供一些脚本辅助github CI机器人，方便标签和合并代码：


* approve.sh

合并一个PR，使用方法：

    ./scripts/approve.sh <PRN> [check_reviewers]

合并一个PR之前，将会做如下检查：

1. 该PR的状态为open
2. 该PR的mergeable状态为true (如果有冲突，则mergeable=false)
3. 该PR的所有CI检查都通过
4. 如果命令行的check_reviewers为非空字符串，则还会检查是否所有requested reviewers都lgtm了这个PR

需要注意的是，在执行脚本之前，请确认已经人肉review过代码，并且认为可以合并了再执行。脚本只是为了方便合并，并且确保合并前没有忽略的检查，并不是为了替代人肉code review。


* approve_all.sh

合并一组PR，使用方法：

    ./scripts/approve_all.sh <PRN>

这组PR由master上的主PR和backport到各个分支的cherry pick PR组成。对Master上的PR会做所有4项检查，对其他PR只做前3项检查。


* lgtm.sh

给一个PR打上lgtm的标签。使用方法：

    ./scripts/lgtm.sh <PRN>
