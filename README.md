# go-id-alloc
Golang+Mysql实现的分布式ID生成服务

# 特性
* 高性能：分配ID只访问内存
* 分布式：横向扩展，理论无上限
* 高可靠：Mysql持久化，故障恢复快
* 唯一性：生成64位整形，整体递增，永不重复
* 易用性：可自定义ID起始位置，对外HTTP服务
* 可运维性：提供健康检查接口，通过负载均衡自动摘除故障节点

# 编译项目
* 进入$GOPATH目录（至少包含src子目录），下载源码

```
go get -u github.com/owenliang/go-id-alloc
```

* 安装mysql依赖

```
go get -u github.com/go-sql-driver/mysql
```

* 进入目录

```
$GOPATH/src/github.com/owenliang/go-id-alloc
```

* 编译项目

```
go build
```

# 初始化数据库

```
create database id_alloc_db;

use id_alloc_db;

CREATE TABLE `segments` (
 `biz_tag` varchar(32) NOT NULL,
 `max_id` bigint NOT NULL,
 `step` bigint NOT NULL,
 `description` varchar(1024) DEFAULT '' NOT NULL,
 `update_time` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
 PRIMARY KEY (`biz_tag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO segments(`biz_tag`, `max_id`, `step`, `description`) VALUES('test', 0, 100000, "test业务ID池");
```

# 启动程序

```
./go-id-alloc -config ./alloc.json
```

# 请求分配ID

```
http://localhost:8880/alloc?biz_tag=test

{"errno":0,"msg":"success","id":201}
```

# 健康检查

```
http://localhost:8880/health?biz_tag=test

{"errno":0,"msg":"success","left":2}
```

# 原理

## 核心SQL

```
Begin
UPDATE table SET max_id=max_id+step WHERE biz_tag=xxx
SELECT tag, max_id, step FROM table WHERE biz_tag=xxx
Commit
```

## 设计参考

[Leaf——美团点评分布式ID生成系统](https://tech.meituan.com/MT_Leaf.html)
