package v3

import (
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"time"
)

// Mapping 定义从源JSON到目标JSON的映射规则
type Mapping struct {
	SourcePath       string                 `yaml:"source_path"`        // 源JSON路径，支持集合通配符和过滤
	TargetPath       string                 `yaml:"target_path"`        // 目标JSON路径
	Type             string                 `yaml:"type"`               // 目标数据类型
	DefaultValue     interface{}            `yaml:"default_value"`      // 默认值
	TimeFormat       string                 `yaml:"time_format"`        // 源时间格式
	TargetTimeFormat string                 `yaml:"target_time_format"` // 目标时间格式
	Timezone         string                 `yaml:"timezone"`           // 时区
	EnumMap          map[string]interface{} `yaml:"enum_map"`           // 枚举值映射表
	EnumIgnoreCase   bool                   `yaml:"enum_ignore_case"`   // 枚举映射是否忽略大小写
	EnumDefault      interface{}            `yaml:"enum_default"`       // 枚举未匹配时的默认值
}

// Config 定义配置文件结构
type Config struct {
	Mappings []Mapping `yaml:"mappings"`
}

// LoadConfig 从YAML文件加载配置
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("配置映射到结构体失败: %v", err)
	}

	return &config, nil
}

// TransformJSON 根据配置转换JSON
func TransformJSON(sourceJSON string, config *Config) (string, error) {
	targetJSON := "{}" // 初始化空JSON对象

	for _, mapping := range config.Mappings {
		// 处理集合字段映射
		if strings.Contains(mapping.SourcePath, ".#.") && strings.Contains(mapping.TargetPath, ".#.") {
			if err := processCollectionMapping(sourceJSON, &targetJSON, mapping); err != nil {
				return "", fmt.Errorf("处理集合映射失败 (路径: %s): %v", mapping.SourcePath, err)
			}
			continue
		}

		// 获取源JSON路径的值
		sourceValue := gjson.Get(sourceJSON, mapping.SourcePath)

		// 如果源路径不存在且有默认值，则使用默认值
		if !sourceValue.Exists() || sourceValue.Type == gjson.Null {
			if mapping.DefaultValue != nil {
				if err := setValue(&targetJSON, mapping, mapping.DefaultValue); err != nil {
					return "", fmt.Errorf("设置默认值失败 (路径: %s): %v", mapping.TargetPath, err)
				}
			}
			continue
		}
		targetValue, err := convertValue(sourceValue, mapping)
		if err != nil {
			return "", fmt.Errorf("转换值失败 (路径: %s): %v", mapping.SourcePath, err)
		}

		if err := setValue(&targetJSON, mapping, targetValue); err != nil {
			return "", fmt.Errorf("设置目标值失败 (路径: %s): %v", mapping.TargetPath, err)
		}
	}

	return targetJSON, nil
}

// processCollectionMapping 处理集合字段映射
func processCollectionMapping(sourceJSON string, targetJSON *string, mapping Mapping) error {
	// 提取集合路径和元素路径
	parts := strings.Split(mapping.SourcePath, ".#")
	if len(parts) < 2 {
		return fmt.Errorf("集合映射路径格式不正确: %s", mapping.SourcePath)
	}

	collectionPath := parts[0]
	elementPath := strings.Join(parts[1:], "#")

	// 获取集合
	collection := gjson.Get(sourceJSON, collectionPath)
	if !collection.IsArray() {
		return fmt.Errorf("集合路径不是数组: %s", collectionPath)
	}

	// 遍历集合元素
	collection.ForEach(func(index, element gjson.Result) bool {
		elementIndex := int(index.Int())
		// 构建元素完整路径
		fullElementPath := fmt.Sprintf("%s.%d%s", collectionPath, int(index.Int()), elementPath)
		targetElementPath := strings.ReplaceAll(mapping.TargetPath, ".#.", fmt.Sprintf(".%d.", elementIndex))

		// 映射元素
		elementMapping := mapping
		elementMapping.SourcePath = fullElementPath
		elementMapping.TargetPath = targetElementPath

		sourceValue := gjson.Get(sourceJSON, elementMapping.SourcePath)
		if !sourceValue.Exists() || sourceValue.Type == gjson.Null {
			if elementMapping.DefaultValue != nil {
				err := setValue(targetJSON, elementMapping, elementMapping.DefaultValue)
				if err != nil {
					return false
				}
			}
			return true
		}

		targetValue, err := convertValue(sourceValue, elementMapping)
		if err != nil {
			fmt.Printf("转换元素值失败 (路径: %s): %v\n", elementMapping.SourcePath, err)
			return true
		}
		err = setValue(targetJSON, elementMapping, targetValue)
		if err != nil {
			return false
		}

		return true
	})
	return nil
}

// convertValue 根据映射规则转换值
func convertValue(sourceValue gjson.Result, mapping Mapping) (interface{}, error) {
	// 先处理枚举映射
	if len(mapping.EnumMap) > 0 {
		if enumValue, err := applyEnumMapping(sourceValue, mapping); err != nil {
			return nil, err
		} else if enumValue != nil {
			return enumValue, nil
		}
	}

	// 根据目标类型进行转换
	switch strings.ToLower(mapping.Type) {
	case "string":
		return sourceValue.String(), nil
	case "int", "integer":
		return sourceValue.Int(), nil
	case "float", "number":
		return sourceValue.Float(), nil
	case "[]string":
		var arr []string
		sourceValue.ForEach(func(_, v gjson.Result) bool {
			arr = append(arr, v.String())
			return true
		})
		return arr, nil
	case "[]int":
		var arr []int64
		sourceValue.ForEach(func(_, v gjson.Result) bool {
			arr = append(arr, v.Int())
			return true
		})
		return arr, nil
	case "bool", "boolean":
		return sourceValue.Bool(), nil
	case "time":
		value, err := convertTime(sourceValue, mapping)
		if err != nil {
			return nil, err
		}
		return value, nil
	case "object":
		return sourceValue.Value(), nil
	case "array":
		return sourceValue.Value(), nil
	default:
		return sourceValue.Value(), nil
	}
}

// applyEnumMapping 应用枚举值映射
func applyEnumMapping(sourceValue gjson.Result, mapping Mapping) (interface{}, error) {
	sourceStr := sourceValue.String()
	lookupKey := sourceStr

	if mapping.EnumIgnoreCase {
		lookupKey = strings.ToLower(lookupKey)
	}

	// 查找映射值
	for key, value := range mapping.EnumMap {
		comparisonKey := key
		if mapping.EnumIgnoreCase {
			comparisonKey = strings.ToLower(comparisonKey)
		}

		if comparisonKey == lookupKey {
			return value, nil
		}
	}

	// 如果未找到映射且设置了默认值，返回默认值
	if mapping.EnumDefault != nil {
		return mapping.EnumDefault, nil
	}

	return nil, fmt.Errorf("枚举值 '%s' 未找到映射且无默认值", sourceStr)
}

// convertTime 转换时间格式
func convertTime(sourceValue gjson.Result, mapping Mapping) (interface{}, error) {
	var t time.Time
	var err error

	// 尝试解析Unix时间戳
	if sourceValue.Type == gjson.Number {
		num := sourceValue.Float()
		if num <= 2147483647 {
			t = time.Unix(int64(num), 0)
		} else {
			t = time.UnixMilli(int64(num))
		}
	} else {
		// 尝试解析时间字符串
		timeStr := sourceValue.String()
		timeFormats := []string{
			mapping.TimeFormat,
			time.RFC3339,
			time.RFC1123,
			time.RFC1123Z,
			time.RFC822,
			time.RFC822Z,
			time.RFC850,
			time.ANSIC,
			time.UnixDate,
			time.RubyDate,
			time.Layout,
			time.Stamp,
			time.StampMilli,
			time.StampMicro,
			time.StampNano,
			"2006-01-02",
			"2006/01/02",
			"02/01/2006",
			"01-02-2006",
			"2006年01月02日",
			"2006-01-02 15:04:05",
			"2006/01/02 15:04:05",
			"02/01/2006 15:04:05",
			"2006-01-02T15:04:05Z07:00",
		}

		found := false
		for _, format := range timeFormats {
			if format == "" {
				continue
			}

			t, err = time.Parse(format, timeStr)
			if err == nil {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("无法解析时间格式: %s", timeStr)
		}
	}

	// 处理时区转换
	if mapping.Timezone != "" {
		loc, err := time.LoadLocation(mapping.Timezone)
		if err != nil {
			return nil, fmt.Errorf("无效的时区: %s", mapping.Timezone)
		}
		t = t.In(loc)
	}

	switch mapping.TargetTimeFormat {
	case "unix":
		// 转换为Unix时间戳（秒）
		return t.Unix(), nil
	case "unix_ms":
		// 转换为Unix时间戳（毫秒）
		return t.UnixMilli(), nil
	default:
		// 格式化目标时间
		if mapping.TargetTimeFormat != "" {
			return t.Format(mapping.TargetTimeFormat), nil
		}
		return t.UnixMilli(), nil
	}
}

// setValue 设置目标JSON的值
func setValue(targetJSON *string, mapping Mapping, value interface{}) error {
	var err error

	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "{") && strings.HasSuffix(v, "}") ||
			strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
			*targetJSON, err = sjson.SetRaw(*targetJSON, mapping.TargetPath, v)
		} else {
			*targetJSON, err = sjson.Set(*targetJSON, mapping.TargetPath, v)
		}
	default:
		*targetJSON, err = sjson.Set(*targetJSON, mapping.TargetPath, v)
	}

	return err
}
