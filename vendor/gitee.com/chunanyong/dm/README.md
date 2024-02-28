# dm

### 介绍
``` 
go get gitee.com/chunanyong/dm 
```  
达梦数据库官方Go驱动,本项目和官方驱动版本同步,方便go mod 使用.  
安装达梦数据库(版本>=8.1.1.126),安装目录下 drivers/go/dm-go-driver.zip    
达梦官方文档:https://eco.dameng.com/docs/zh-cn/app-dev/go-go.html    
资源下载:https://eco.dameng.com/download/    
达梦官方Go驱动包:https://package.dameng.com/eco/adapter/resource/go/dm-go-driver.zip  
达梦官方论坛(提交bug):https://eco.dameng.com/community/question  

### zorm  
Go轻量ORM https://gitee.com/chunanyong/zorm 原生支持达梦数据库  

### DSN  
dm://userName:password@127.0.0.1:5236?schema=DBName  
用户名(userName)默认就是数据库的名称,达梦用户模式和数据库名称是对应的,也可以通过schema参数指定数据库  
建议达梦使用UTF-8字符编码,不区分大小写,建表语句的字段名不要带""双引号      

### bug
- 达梦开启等保参数 COMM_ENCRYPT_NAME = AES128_ECB，导致连接异常

### 版本号  
Go三段位版本号和达梦四段位版本号不兼容,统一使用1.达梦主版本号.发布的小版本号,具体查看标签的备注  

* v1.8.13 来自 达梦8.1.3.62      
* v1.8.12 来自 达梦8.1.3.12  
* v1.8.11 来自 达梦8.1.2.192
* v1.8.10 来自 达梦8.1.2.174
* v1.8.9  来自 达梦8.1.2.162
* v1.8.8  来自 达梦8.1.2.138
* v1.8.7  来自 达梦8.1.2.128 
* v1.8.6  来自 达梦8.1.2.114 
* v1.8.5  来自 达梦8.1.2.94    
* v1.8.4  来自 达梦8.1.2.84 
* v1.8.3  来自 达梦8.1.2.38  
* v1.8.2  来自 达梦8.1.2.18  
* v1.8.1  来自 达梦8.1.1.190  
* v1.8.0  来自 达梦8.1.1.126  




    
    




