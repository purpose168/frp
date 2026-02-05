package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRandId 测试 RandID 函数
func TestRandId(t *testing.T) {
	require := require.New(t)
	// 生成随机 ID
	id, err := RandID()
	require.NoError(err)
	// 输出 ID
	t.Log(id)
	// 验证 ID 长度为 16
	require.Equal(16, len(id))
}

// TestGetAuthKey 测试 GetAuthKey 函数
func TestGetAuthKey(t *testing.T) {
	require := require.New(t)
	// 生成认证密钥
	key := GetAuthKey("1234", 1488720000)
	// 验证密钥是否正确
	require.Equal("6df41a43725f0c770fd56379e12acf8c", key)
}

// TestParseRangeNumbers 测试 ParseRangeNumbers 函数
func TestParseRangeNumbers(t *testing.T) {
	require := require.New(t)
	// 测试范围数字解析
	numbers, err := ParseRangeNumbers("2-5")
	require.NoError(err)
	require.Equal([]int64{2, 3, 4, 5}, numbers)

	// 测试单个数字解析
	numbers, err = ParseRangeNumbers("1")
	require.NoError(err)
	require.Equal([]int64{1}, numbers)

	// 测试组合范围和单个数字解析
	numbers, err = ParseRangeNumbers("3-5,8")
	require.NoError(err)
	require.Equal([]int64{3, 4, 5, 8}, numbers)

	// 测试带空格的组合范围和单个数字解析
	numbers, err = ParseRangeNumbers(" 3-5,8, 10-12 ")
	require.NoError(err)
	require.Equal([]int64{3, 4, 5, 8, 10, 11, 12}, numbers)

	// 测试无效的数字格式
	_, err = ParseRangeNumbers("3-a")
	require.Error(err)
}
