package database

import (
	"os"
	"path/filepath"

	"github.com/atticus6/echPlus/apps/desktop/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dbPath string) error {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	// 自动迁移
	return DB.AutoMigrate(&models.User{}, &models.Node{})
}

func GetDB() *gorm.DB {
	return DB
}
