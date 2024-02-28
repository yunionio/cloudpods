# Change Logs

*你可以在文件p.go中的发版标记里找到当前驱动的svn号

## svn 16752
支持在连接串上直接配置动态服务名，使用示例：
dm://user:password@GroupName?GroupName=(host1:port1,host2:port2,...)

## svn 16505
新增连接串属性driverReconnect，配合doSwitch=1或2使用，表示连接重连是否使用驱动自身的重连机制，否则在连接失效时返回sql标准错误driver.ErrBadConn，由go来处理重连

## svn 16258
重连逻辑修改，当连接失效时，返回driver.ErrBadConn而不是驱动自己管理重连，连接串参数doSwitch默认值改为1
驱动接口方法同步锁改到连接网络请求的总入口处，解决一些panic和数组越界问题
日志优化，去除遍历结果集时因io.EOF错误记录的日志

## svn 15619
驱动接口方法添加同步锁
驱动日志修改bug，可以记录SQL语句和参数值了

## svn 15357
发布方言包，支持gorm v1和v2框架，方言包位于达梦安装目录的drivers/go目录中，详细使用说明参考《DM8程序员手册》

## svn 15157
修复了字符大字段(Clob)中存在乱码字符时，读取结果会漏读一些字符的问题
修复了开启SSL后并发创建连接导致panic的问题

## svn 15035
修复了连接串属性doSwitch=1时，语句不会自动切换和重连的问题
修复了连接重置后，可能出现的空指针问题

## svn 14992
修复了连接串属性doSwitch=1时，连接不会自动切换和重连的问题
修复了连接串属性loginMode=默认值4时，备库可能会被优先连接的问题

## svn 14589
sql.Result.LastInsertId()函数能优先返回自增列的值，如果没有则返回数据库表内部的rowid
修复了开启事务时指定只读不生效的问题