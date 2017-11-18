# go-id-alloc
Golang+Mysql实现的分布式ID发号器

# 特点
* 全局ID唯一，整体趋势递增，永不重复
* 纯内存操作，超高单机性能，Mysql零压力
* 支持集群部署，高可用，无限横向扩展
* Mysql持久化，Master-Slave主备，故障节点人工摘除，可以快速恢复
* 自定义ID起始大小，方便接入现有业务
* HTTP协议接口，使用简单

# 准备环境
* 进入$GOPATH（包含src目录），下载源码
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
打开alloc.json准备进行配置

# 配置示例
假设集群由2台go-id-alloc负载均衡（例如使用haproxy）保障高可用，并且计划对外提供最大40w/s的ID分配QPS。
* 申请2个独立的Mysql实例（出于成本考虑，可以复用已有的Mysql，不会对其造成任何压力）
* 在2个实例上分别建立一个专用数据库，名称任意，不强制相同
* 在Mysql1中执行如下语句：
```
CREATE TABLE `partition_1` (
 `id` bigint(20) NOT NULL AUTO_INCREMENT,
 `meanless` tinyint(4) NOT NULL,
 PRIMARY KEY (`id`),
 UNIQUE KEY `meanless` (`meanless`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `partition_1` SET auto_increment_increment=1;
```
* 在Mysql2中执行如下语句：
```
CREATE TABLE `partition_2` (
 `id` bigint(20) NOT NULL AUTO_INCREMENT,
 `meanless` tinyint(4) NOT NULL,
 PRIMARY KEY (`id`),
 UNIQUE KEY `meanless` (`meanless`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `partition_2` SET auto_increment_increment=2;
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

# Mysql故障修复

* 定期调用alloc接口，若连续失败则应当首先关闭进程，人工介入。
* 若Mysql Master无法恢复，则等待Slave同步完成后，记录partition表中id字段的值为N。
* 将Slave提升为主库，并执行SQL清空表：
```
truncate partition_xxx;
```
* 预估故障时主从延迟，找到X，能够令M = (N + TotalPartition * X) * SegmentSize大于id-alloc-size可能最后分配的1个ID。
* 执行SQL初始化新的起始偏移量：
```
ALTER TABLE `partition_xxx` SET auto_increment_increment=(N+TotalParition*X);
```
* 重新启动go-id-alloc，恢复服务。

