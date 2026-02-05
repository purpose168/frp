package security

const (
	// TokenSourceExec 表示令牌来源为执行命令
	TokenSourceExec = "TokenSourceExec"
)

var (
	// ClientUnsafeFeatures 定义客户端不安全特性列表
	ClientUnsafeFeatures = []string{
		TokenSourceExec,
	}

	// ServerUnsafeFeatures 定义服务端不安全特性列表
	ServerUnsafeFeatures = []string{
		TokenSourceExec,
	}
)

// UnsafeFeatures 管理不安全特性的启用状态
type UnsafeFeatures struct {
	// features 保存特性名称到启用状态的映射
	features map[string]bool
}

// NewUnsafeFeatures 创建一个新的 UnsafeFeatures 实例
// 参数 allowed 是允许启用的特性名称列表
func NewUnsafeFeatures(allowed []string) *UnsafeFeatures {
	features := make(map[string]bool)
	for _, f := range allowed {
		features[f] = true
	}
	return &UnsafeFeatures{features: features}
}

// IsEnabled 检查指定的不安全特性是否已启用
// 参数 feature 是要检查的特性名称
// 返回值表示该特性是否已启用
func (u *UnsafeFeatures) IsEnabled(feature string) bool {
	if u == nil {
		return false
	}
	return u.features[feature]
}
