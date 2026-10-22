package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	// 内嵌时区数据库，容器内无需安装 tzdata 即可使用 Asia/Shanghai
	_ "time/tzdata"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"glimmer-interview-hub/backend/internal/auth"
	"glimmer-interview-hub/backend/internal/config"
	"glimmer-interview-hub/backend/internal/handler"
	"glimmer-interview-hub/backend/internal/model"
	"glimmer-interview-hub/backend/internal/repository"

	// 纯 Go SQLite 驱动（无需 CGO）
	"github.com/glebarez/sqlite"
)

func main() {
	cfg := config.Load()

	if dir := filepath.Dir(cfg.DBPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("创建数据目录失败: %v", err)
		}
	}

	// WAL 模式提升读写并发，busy_timeout 避免写入锁竞争
	dsn := "file:" + cfg.DBPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(4) // SQLite 单文件写串行，控制连接数
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := db.AutoMigrate(
		&model.Admin{},
		&model.User{},
		&model.UserDirection{},
		&model.Session{},
		&model.Slot{},
		&model.Preference{},
		&model.AssignmentVersion{},
		&model.Assignment{},
		&model.Invite{},
		&model.InviteRedemption{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 复合唯一约束（幂等）
	for _, sql := range model.UniqueIndexes {
		if err := db.Exec(sql).Error; err != nil {
			log.Fatalf("创建唯一索引失败: %v", err)
		}
	}

	seedAdmins(db, cfg.AdminUsernames)

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.LoggerWithWriter(gin.DefaultWriter), gin.Recovery())

	repo := repository.New(db)
	validator := auth.NewLegacyValidator(cfg.LegacyAPIBase, cfg.LegacyValidatePath, cfg.TokenCacheMinutes)
	handler.New(repo, cfg.JWTSecret, validator).Register(r)

	log.Printf("微光面试预约平台后端启动，监听 %s", cfg.Addr)
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// seedAdmins 幂等地把配置中的初始管理员写入名单
func seedAdmins(db *gorm.DB, usernames []string) {
	for _, u := range usernames {
		var count int64
		db.Model(&model.Admin{}).Where("username = ?", u).Count(&count)
		if count == 0 {
			if err := db.Create(&model.Admin{Username: u, CreatedAt: time.Now()}).Error; err != nil {
				log.Printf("种子化管理员 %s 失败: %v", u, err)
				continue
			}
			log.Printf("已添加初始管理员: %s", u)
		}
	}
}
