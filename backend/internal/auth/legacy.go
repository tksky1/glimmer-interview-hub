package auth

import (
	"net/http"
	"sync"
	"time"
)

// LegacyValidator 通过旧系统 API 委派校验 token。
// 旧系统部署版本的 JWT 密钥可能与本仓库配置不同，直接委派可对任意密钥生效，
// 也无需关心旧系统将来轮换密钥。
type LegacyValidator struct {
	baseURL      string
	validatePath string
	ttl          time.Duration
	client       *http.Client

	mu    sync.RWMutex
	cache map[string]time.Time // token -> 缓存过期时间
}

// NewLegacyValidator 构造委派校验器
func NewLegacyValidator(baseURL, validatePath string, ttlMinutes int) *LegacyValidator {
	if ttlMinutes <= 0 {
		ttlMinutes = 10
	}
	return &LegacyValidator{
		baseURL:      baseURL,
		validatePath: validatePath,
		ttl:          time.Duration(ttlMinutes) * time.Minute,
		client:       &http.Client{Timeout: 8 * time.Second},
		cache:        make(map[string]time.Time),
	}
}

// Validate 校验 token 是否被旧系统认可。
// 返回 (valid, reachable)：reachable 为 false 表示旧 API 网络不可达（而非拒绝）。
func (v *LegacyValidator) Validate(token string) (valid bool, reachable bool) {
	// 缓存命中直接通过
	v.mu.RLock()
	exp, ok := v.cache[token]
	v.mu.RUnlock()
	if ok && time.Now().Before(exp) {
		return true, true
	}

	req, err := http.NewRequest(http.MethodGet, v.baseURL+v.validatePath, nil)
	if err != nil {
		return false, false
	}
	req.Header.Set("token", token)

	resp, err := v.client.Do(req)
	if err != nil {
		// 网络不可达：交由调用方决定放行策略
		return false, false
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		v.mu.Lock()
		v.cache[token] = time.Now().Add(v.ttl)
		// 顺手清理过期项，避免缓存无限增长
		for t, e := range v.cache {
			if time.Now().After(e) {
				delete(v.cache, t)
			}
		}
		v.mu.Unlock()
		return true, true
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, true
	default:
		// 旧 API 异常（5xx 等）：不视为拒绝，交由调用方决定
		return false, false
	}
}
