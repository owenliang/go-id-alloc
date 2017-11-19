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
$GOPATH/src/github.com/owenliang/go-id-alloc/main
```

* 编译项目

```
go build
```

# 集群配置

go-id-alloc依赖Mysql持久化ID，若Mysql故障则go-id-alloc无法继续服务。

为了保障集群高可用，应该为每个go-id-alloc分别创建不同的Mysql实例（服务器），从而保障个别节点故障，其他节点仍可以对外服务。

但go-id-alloc并不强制与Mysql实例1:1配置，完全可以部分go-id-alloc共用同一个Mysql实例，只不过此时若Mysql故障的话，所有使用它的go-id-alloc均会不可用。

假设集群由2台go-id-alloc组成，前台使用负载均衡（例如：haproxy）反向代理。

按业务需求分析，计划对外提供最大QPS = 40w/s的ID分配能力，那么进行如下配置即可：

* 申请2个独立的Mysql实例（可以复用已有Mysql实例）
* 在2个实例上分别建立一个专用数据库id_alloc_db
* 在Mysql1中建立如下table：

```
CREATE TABLE `partition_1` (
 `id` bigint(20) NOT NULL AUTO_INCREMENT,
 `meanless` tinyint(4) NOT NULL,
 PRIMARY KEY (`id`),
 UNIQUE KEY `meanless` (`meanless`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `partition_1` auto_increment=1;
```

* 在Mysql2中建立如下table：

```
CREATE TABLE `partition_2` (
 `id` bigint(20) NOT NULL AUTO_INCREMENT,
 `meanless` tinyint(4) NOT NULL,
 PRIMARY KEY (`id`),
 UNIQUE KEY `meanless` (`meanless`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `partition_2` auto_increment=2;
```

* 修改go-id-alloc进程1的配置如下：

```
{
  "PartitionIdx": 1,
  "TotalPartition" : 2,
  "SegmentSize": 200000,
  "DSN": "user:password@tcp(Mysql1:3306)/id_alloc_db",

  "HttpPort": 8880,
  "HttpReadTimeout": 5000,
  "HttpWriteTimeout": 5000
}
```

* 修改go-id-alloc进程2的配置如下：

```
{
  "PartitionIdx": 2,
  "TotalPartition" : 2,
  "SegmentSize": 200000,
  "DSN": "user:password@123@tcp(Mysql2:3306)/id_alloc_db",

  "HttpPort": 8880,
  "HttpReadTimeout": 5000,
  "HttpWriteTimeout": 5000
}
```

* 分别启动2个go-id-alloc服务：

```
go run main.go -config ./alloc.json
```

# 请求示例
```
request: http://localhost:8880/alloc

response:
{
Errno: 0,
Msg: "",
Id: 130000001
}
```

# 健康检查接口
一般推荐使用haproxy，它可以访问特定接口进行健康检查。

其他负载均衡无需特别配置，/alloc接口在故障时同样会返回500错误码。

若Mysql存在故障，健康接口将返回的HTTP Code = 500

```
request: http://localhost:8080/health

response:
{
Left: 399994
}

```

# Mysql故障修复

* 通常Mysql故障后，对应的go-id-alloc服务应当会被负载均衡自动摘除。
* 通过Slave查看对应partition表的id大小，记录为N。
* 在新Master上清空partition表：

```
truncate partition_xxx;
```

* 因为Master-Slave可能存在主从延迟，所以数字N可能偏小，为了保险起见应该选择一个数字M = N + TotalParition * X，建议X >= 10以跳过可能已经在Master上分配出去的号段：

```
ALTER TABLE `partition_xxx` auto_increment=M;
```

* 重新启动go-id-alloc，恢复服务。

