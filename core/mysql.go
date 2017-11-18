package core

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"strings"
	"context"
	"time"
)

type Mysql struct {
	db *sql.DB
}

var GMysql *Mysql

func InitMysql() error {
	options := []string {
		"auto_increment_increment=" + strconv.Itoa(GConf.TotalPartition),
	}
	db, err := sql.Open("mysql", GConf.DSN + "?" + strings.Join(options, "&"))
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	GMysql = &Mysql{db: db}
	return nil
}

func (mysql *Mysql)NextId() (nextId int64, err error) {
	var (
		conn *sql.Conn
		stmt *sql.Stmt
		res sql.Result
	)

	// 总耗时小于2秒
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(2000) * time.Millisecond)
	defer cancelFunc()

	if conn, err = mysql.db.Conn(ctx); err != nil {
		return
	}
	defer conn.Close()

	sql := "replace into partition_" + strconv.Itoa(GConf.PartitionIdx)  + "(`meanless`) values(0)"
	if stmt, err = conn.PrepareContext(ctx, sql); err != nil {
		return
	}
	if res, err = stmt.ExecContext(ctx); err != nil {
		return
	}
	if nextId, err = res.LastInsertId(); err != nil {
		return
	}
	return
}