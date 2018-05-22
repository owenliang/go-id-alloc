package core

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"context"
	"time"
	"errors"
)

/**
建表：

	CREATE TABLE `segments` (
	 `biz_tag` varchar(32) NOT NULL,
	 `max_id` bigint NOT NULL,
	 `step` bigint NOT NULL,
	 `desc` varchar(1024) DEFAULT '' NOT NULL,
	 `update_time` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	 PRIMARY KEY (`biz_tag`)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8;
 */

type Mysql struct {
	db *sql.DB
}

var GMysql *Mysql

func InitMysql() error {
	db, err := sql.Open("mysql", GConf.DSN)
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0)
	GMysql = &Mysql{db: db}
	return nil
}

func (mysql *Mysql)NextId(bizTag string) (maxId int64, step int64, err error) {
	var (
		tx *sql.Tx
		query string
		stmt *sql.Stmt
		result sql.Result
		rowsAffected int64
	)

	// 总耗时小于2秒
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(2000) * time.Millisecond)
	defer cancelFunc()

	// 开启事务
	if tx, err = mysql.db.BeginTx(ctx, nil); err != nil {
		return
	}

	// 1, 前进一个步长, 即占用一个号段(更新操作是悲观行锁)
	query = "UPDATE " + GConf.Table + " SET max_id=max_id+step WHERE biz_tag=?"
	if stmt, err = tx.PrepareContext(ctx, query); err != nil {
		goto ROLLBACK
	}

	if result, err = stmt.ExecContext(ctx, bizTag); err != nil {
		goto ROLLBACK
	}

	if rowsAffected, err = result.RowsAffected(); err != nil { // 失败
		goto ROLLBACK
	} else if rowsAffected == 0 { // 记录不存在
		err = errors.New("biz_tag not found")
		goto ROLLBACK
	}

	// 2, 查询更新后的最新max_id, 此时仍在事务内, 行锁保护下
	query = "SELECT max_id, step FROM " + GConf.Table + " WHERE biz_tag=?"
	if stmt, err = tx.PrepareContext(ctx, query); err != nil {
		goto ROLLBACK
	}
	if err = stmt.QueryRowContext(ctx, bizTag).Scan(&maxId, &step); err != nil {
		goto ROLLBACK
	}

	// 3, 提交事务
	err = tx.Commit()
	return

ROLLBACK:
	tx.Rollback()
	return
}
