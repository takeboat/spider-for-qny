package spider

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB 定义数据库连接结构
var DB *gorm.DB

// NewGormDB 创建并初始化GORM数据库连接
func MustInitDB() {
	dsn := "ftyonReader:fTpoWer0323@tcp(rm-2vc0efl9w68888948co.mysql.cn-chengdu.rds.aliyuncs.com:3306)/dcpc_charger?charset=utf8mb4&parseTime=true"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	DB = db
}

func GetDB() *gorm.DB {
	return DB
}

