package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 应用配置，全部来自环境变量（docker-compose 的 .env 注入）
type Config struct {
	Addr               string   // HTTP 监听地址
	DBPath             string   // SQLite 数据文件路径
	JWTSecret          string   // 本地 JWT 签名密钥（快路径校验，默认与旧系统仓库一致）
	AdminUsernames     []string // 初始管理员用户名（旧系统登录名）
	LegacyAPIBase      string   // 旧系统 API 地址（委派校验 token）
	LegacyValidatePath string   // 旧系统用于校验 token 的接口（带 token 访问返回 200/401）
	TokenCacheMinutes  int      // token 委派校验结果的缓存分钟数
}

// Load 读取环境变量并填充默认值
func Load() *Config {
	return &Config{
		Addr:               getEnv("ADDR", ":8080"),
		DBPath:             getEnv("DB_PATH", "/data/interview.db"),
		JWTSecret:          getEnv("JWT_SECRET", "RecruitmentSystem"),
		AdminUsernames:     splitCSV(os.Getenv("ADMIN_USERNAMES")),
		LegacyAPIBase:      getEnv("LEGACY_API_BASE", "https://api.glimmer.org.cn"),
		LegacyValidatePath: getEnv("LEGACY_VALIDATE_PATH", "/problem"),
		TokenCacheMinutes:  getEnvInt("TOKEN_CACHE_MINUTES", 10),
	}
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
