package validation

import (
	"fmt"

	"github.com/purpose168/frp/pkg/policy/security"
)

// ConfigValidator 保存配置验证的上下文依赖
type ConfigValidator struct {
	unsafeFeatures *security.UnsafeFeatures
}

// NewConfigValidator 创建一个新的 ConfigValidator 实例
// 参数 unsafeFeatures 为不安全特性配置对象
// 返回新创建的 ConfigValidator 实例
func NewConfigValidator(unsafeFeatures *security.UnsafeFeatures) *ConfigValidator {
	return &ConfigValidator{
		unsafeFeatures: unsafeFeatures,
	}
}

// ValidateUnsafeFeature 检查特定不安全特性是否已启用
// 参数 feature 为要检查的特性名称
// 返回验证过程中产生的错误信息
func (v *ConfigValidator) ValidateUnsafeFeature(feature string) error {
	// 检查特性是否已启用
	if !v.unsafeFeatures.IsEnabled(feature) {
		return fmt.Errorf("不安全特性 %q 未启用。"+
			"要启用它，请确保在配置或命令行标志中允许它", feature)
	}
	return nil
}
