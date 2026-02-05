// Copyright 2023 The frp Authors
//
// Licensed under to Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with License.
// You may obtain a copy of License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See License for the specific language governing permissions and
// limitations under License.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
	"gopkg.in/ini.v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/fatedier/frp/pkg/config/legacy"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/util"
)

var glbEnvs map[string]string

// init 初始化全局环境变量映射
func init() {
	glbEnvs = make(map[string]string)
	envs := os.Environ()
	for _, env := range envs {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}
		glbEnvs[pair[0]] = pair[1]
	}
}

// Values 值结构体，包含环境变量
type Values struct {
	Envs map[string]string // 环境变量
}

// GetValues 获取值结构体
func GetValues() *Values {
	return &Values{
		Envs: glbEnvs,
	}
}

// DetectLegacyINIFormat 检测是否为旧版 INI 格式
func DetectLegacyINIFormat(content []byte) bool {
	f, err := ini.Load(content)
	if err != nil {
		return false
	}
	if _, err := f.GetSection("common"); err == nil {
		return true
	}
	return false
}

// DetectLegacyINIFormatFromFile 从文件检测是否为旧版 INI 格式
func DetectLegacyINIFormatFromFile(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return DetectLegacyINIFormat(b)
}

// RenderWithTemplate 使用模板渲染内容
func RenderWithTemplate(in []byte, values *Values) ([]byte, error) {
	tmpl, err := template.New("frp").Funcs(template.FuncMap{
		"parseNumberRange":     parseNumberRange,
		"parseNumberRangePair": parseNumberRangePair,
	}).Parse(string(in))
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBufferString("")
	if err := tmpl.Execute(buffer, values); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// LoadFileContentWithTemplate 使用模板加载文件内容
func LoadFileContentWithTemplate(path string, values *Values) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return RenderWithTemplate(b, values)
}

// LoadConfigureFromFile 从文件加载配置
func LoadConfigureFromFile(path string, c any, strict bool) error {
	content, err := LoadFileContentWithTemplate(path, GetValues())
	if err != nil {
		return err
	}
	return LoadConfigure(content, c, strict)
}

// parseYAMLWithDotFieldsHandling 解析带有点号前缀字段处理的 YAML
// 此函数高效处理两种情况：有或没有点号字段
func parseYAMLWithDotFieldsHandling(content []byte, target any) error {
	var temp any
	if err := yaml.Unmarshal(content, &temp); err != nil {
		return err
	}

	// 如果是映射，则删除点号字段
	if tempMap, ok := temp.(map[string]any); ok {
		for key := range tempMap {
			if strings.HasPrefix(key, ".") {
				delete(tempMap, key)
			}
		}
	}

	// 转换为 JSON 并使用严格验证进行解码
	jsonBytes, err := json.Marshal(temp)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

// LoadConfigure 从字节加载配置并反序列化到 c
// 现在支持 json、yaml 和 toml 格式
func LoadConfigure(b []byte, c any, strict bool) error {
	v1.DisallowUnknownFieldsMu.Lock()
	defer v1.DisallowUnknownFieldsMu.Unlock()
	v1.DisallowUnknownFields = strict

	var tomlObj any
	// 首先尝试反序列化为 TOML；忽略该操作的错误（假设它不是有效的 TOML）
	if err := toml.Unmarshal(b, &tomlObj); err == nil {
		b, err = json.Marshal(&tomlObj)
		if err != nil {
			return err
		}
	}
	// 如果缓冲区看起来像 JSON（第一个非空白字符是 '{'），则直接反序列化为 JSON
	if yaml.IsJSONBuffer(b) {
		decoder := json.NewDecoder(bytes.NewBuffer(b))
		if strict {
			decoder.DisallowUnknownFields()
		}
		return decoder.Decode(c)
	}

	// 处理 YAML 内容
	if strict {
		// 在严格模式下，始终使用我们的自定义处理程序以支持 YAML 合并
		return parseYAMLWithDotFieldsHandling(b, c)
	}
	// 非严格模式，正常解析
	return yaml.Unmarshal(b, c)
}

// NewProxyConfigurerFromMsg 从消息创建代理配置器
func NewProxyConfigurerFromMsg(m *msg.NewProxy, serverCfg *v1.ServerConfig) (v1.ProxyConfigurer, error) {
	m.ProxyType = util.EmptyOr(m.ProxyType, string(v1.ProxyTypeTCP))

	configurer := v1.NewProxyConfigurerByType(v1.ProxyType(m.ProxyType))
	if configurer == nil {
		return nil, fmt.Errorf("未知的代理类型: %s", m.ProxyType)
	}

	configurer.UnmarshalFromMsg(m)
	configurer.Complete("")

	if err := validation.ValidateProxyConfigurerForServer(configurer, serverCfg); err != nil {
		return nil, err
	}
	return configurer, nil
}

// LoadServerConfig 加载服务器配置
func LoadServerConfig(path string, strict bool) (*v1.ServerConfig, bool, error) {
	var (
		svrCfg         *v1.ServerConfig
		isLegacyFormat bool
	)
	// 检测旧版 ini 格式
	if DetectLegacyINIFormatFromFile(path) {
		content, err := legacy.GetRenderedConfFromFile(path)
		if err != nil {
			return nil, true, err
		}
		legacyCfg, err := legacy.UnmarshalServerConfFromIni(content)
		if err != nil {
			return nil, true, err
		}
		svrCfg = legacy.Convert_ServerCommonConf_To_v1(&legacyCfg)
		isLegacyFormat = true
	} else {
		svrCfg = &v1.ServerConfig{}
		if err := LoadConfigureFromFile(path, svrCfg, strict); err != nil {
			return nil, false, err
		}
	}
	if svrCfg != nil {
		if err := svrCfg.Complete(); err != nil {
			return nil, isLegacyFormat, err
		}
	}
	return svrCfg, isLegacyFormat, nil
}

// LoadClientConfig 加载客户端配置
func LoadClientConfig(path string, strict bool) (
	*v1.ClientCommonConfig,
	[]v1.ProxyConfigurer,
	[]v1.VisitorConfigurer,
	bool, error,
) {
	var (
		cliCfg         *v1.ClientCommonConfig
		proxyCfgs      = make([]v1.ProxyConfigurer, 0)
		visitorCfgs    = make([]v1.VisitorConfigurer, 0)
		isLegacyFormat bool
	)

	if DetectLegacyINIFormatFromFile(path) {
		legacyCommon, legacyProxyCfgs, legacyVisitorCfgs, err := legacy.ParseClientConfig(path)
		if err != nil {
			return nil, nil, nil, true, err
		}
		cliCfg = legacy.Convert_ClientCommonConf_To_v1(&legacyCommon)
		for _, c := range legacyProxyCfgs {
			proxyCfgs = append(proxyCfgs, legacy.Convert_ProxyConf_To_v1(c))
		}
		for _, c := range legacyVisitorCfgs {
			visitorCfgs = append(visitorCfgs, legacy.Convert_VisitorConf_To_v1(c))
		}
		isLegacyFormat = true
	} else {
		allCfg := v1.ClientConfig{}
		if err := LoadConfigureFromFile(path, &allCfg, strict); err != nil {
			return nil, nil, nil, false, err
		}
		cliCfg = &allCfg.ClientCommonConfig
		for _, c := range allCfg.Proxies {
			proxyCfgs = append(proxyCfgs, c.ProxyConfigurer)
		}
		for _, c := range allCfg.Visitors {
			visitorCfgs = append(visitorCfgs, c.VisitorConfigurer)
		}
	}

	// 从包含的文件加载额外配置
	// 旧版 ini 格式已在 ParseClientConfig 中处理
	if len(cliCfg.IncludeConfigFiles) > 0 && !isLegacyFormat {
		extProxyCfgs, extVisitorCfgs, err := LoadAdditionalClientConfigs(cliCfg.IncludeConfigFiles, isLegacyFormat, strict)
		if err != nil {
			return nil, nil, nil, isLegacyFormat, err
		}
		proxyCfgs = append(proxyCfgs, extProxyCfgs...)
		visitorCfgs = append(visitorCfgs, extVisitorCfgs...)
	}

	// 按启动项过滤
	if len(cliCfg.Start) > 0 {
		startSet := sets.New(cliCfg.Start...)
		proxyCfgs = lo.Filter(proxyCfgs, func(c v1.ProxyConfigurer, _ int) bool {
			return startSet.Has(c.GetBaseConfig().Name)
		})
		visitorCfgs = lo.Filter(visitorCfgs, func(c v1.VisitorConfigurer, _ int) bool {
			return startSet.Has(c.GetBaseConfig().Name)
		})
	}

	// 按每个代理中的 enabled 字段过滤
	// nil 或 true 表示启用，false 表示禁用
	proxyCfgs = lo.Filter(proxyCfgs, func(c v1.ProxyConfigurer, _ int) bool {
		enabled := c.GetBaseConfig().Enabled
		return enabled == nil || *enabled
	})
	visitorCfgs = lo.Filter(visitorCfgs, func(c v1.VisitorConfigurer, _ int) bool {
		enabled := c.GetBaseConfig().Enabled
		return enabled == nil || *enabled
	})

	if cliCfg != nil {
		if err := cliCfg.Complete(); err != nil {
			return nil, nil, nil, isLegacyFormat, err
		}
	}
	for _, c := range proxyCfgs {
		c.Complete(cliCfg.User)
	}
	for _, c := range visitorCfgs {
		c.Complete(cliCfg)
	}
	return cliCfg, proxyCfgs, visitorCfgs, isLegacyFormat, nil
}

// LoadAdditionalClientConfigs 加载额外的客户端配置
func LoadAdditionalClientConfigs(paths []string, isLegacyFormat bool, strict bool) ([]v1.ProxyConfigurer, []v1.VisitorConfigurer, error) {
	proxyCfgs := make([]v1.ProxyConfigurer, 0)
	visitorCfgs := make([]v1.VisitorConfigurer, 0)
	for _, path := range paths {
		absDir, err := filepath.Abs(filepath.Dir(path))
		if err != nil {
			return nil, nil, err
		}
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			return nil, nil, err
		}
		files, err := os.ReadDir(absDir)
		if err != nil {
			return nil, nil, err
		}
		for _, fi := range files {
			if fi.IsDir() {
				continue
			}
			absFile := filepath.Join(absDir, fi.Name())
			if matched, _ := filepath.Match(filepath.Join(absDir, filepath.Base(path)), absFile); matched {
				// 支持 yaml/json/toml
				cfg := v1.ClientConfig{}
				if err := LoadConfigureFromFile(absFile, &cfg, strict); err != nil {
					return nil, nil, fmt.Errorf("从 %s 加载额外配置错误: %v", absFile, err)
				}
				for _, c := range cfg.Proxies {
					proxyCfgs = append(proxyCfgs, c.ProxyConfigurer)
				}
				for _, c := range cfg.Visitors {
					visitorCfgs = append(visitorCfgs, c.VisitorConfigurer)
				}
			}
		}
	}
	return proxyCfgs, visitorCfgs, nil
}
