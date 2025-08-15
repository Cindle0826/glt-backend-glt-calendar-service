package env

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"os"
	"regexp"
	"sync"
)

// InitConfig reference : https://medium.com/@fdev777/%E5%A5%97%E4%BB%B6%E5%B7%A5%E5%85%B7%E7%AF%87-viper-%E7%92%B0%E5%A2%83%E8%A8%AD%E5%AE%9A%E7%AE%A1%E7%90%86%E4%B8%BB%E6%B5%81%E7%A5%9E%E5%99%A8-24e57ff06246
func InitConfig() *Config {
	viper.SetConfigName("config")
	viper.AddConfigPath("./settings/env/")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config is fail %v \n", err))
	}

	replaceEnvVariablesWithOptionalDefault()

	config := Config{
		ServerConfig: ServerConfig{Port: viper.GetString("server.port")},
		GinConfig:    GinConfig{Mode: viper.GetString("gin.mode")},
		DynamodbConfig: DynamodbConfig{
			DynamodbLocal: &struct {
				Endpoint    string
				AccessKey   string
				AccessKeyId string
			}{
				Endpoint:    viper.GetString("dynamodb.local.endpoint"),
				AccessKey:   viper.GetString("dynamodb.local.access_key"),
				AccessKeyId: viper.GetString("dynamodb.local.access_key_id"),
			},
			Region: viper.GetString("dynamodb.region"),
		},
		GoogleOAuth2: GoogleOAuth2{
			ClientID:     viper.GetString("google.oauth2.client_id"),
			ClientSecret: viper.GetString("google.oauth2.client_secret"),
		},
		HttpAllows: HttpAllows{
			Origins: viper.GetStringSlice("allow.origins"),
		},
		LogConfig: LogConfig{
			Level: viper.GetString("log.level"),
		},
	}

	return &config
}

var (
	config     *Config
	configOnce sync.Once
)

func GetConfig() *Config {
	configOnce.Do(func() {
		config = InitConfig() // 初始化配置
	})
	return config
}

// 替換 YAML 中的環境變數並提供預設值
func replaceEnvVariablesWithOptionalDefault() {
	// 修訂正則表達式以支持特殊字符（如 "-"）
	pattern := regexp.MustCompile(`\${([a-zA-Z0-9_\-]+)(?::([^}]+))?}`)

	for _, key := range viper.AllKeys() {
		value := viper.Get(key)

		switch v := value.(type) {
		case string:
			// 如果是字符串，檢查是否包含佔位符
			matches := pattern.FindStringSubmatch(v)
			if len(matches) > 1 {
				envKey := matches[1] // 環境變數名稱
				defaultValue := ""
				if len(matches) == 3 {
					defaultValue = matches[2] // 預設值（如果有）
				}
				envValue := os.Getenv(envKey)
				if envValue != "" {
					viper.Set(key, envValue)
				} else if defaultValue != "" {
					viper.Set(key, defaultValue)
				}
			}

		case []interface{}:
			// 如果是 slice，逐一檢查每個元素是否包含佔位符
			for i, item := range v {
				if strItem, ok := item.(string); ok {
					matches := pattern.FindStringSubmatch(strItem)
					if len(matches) > 1 {
						envKey := matches[1] // 環境變數名稱
						defaultValue := ""
						if len(matches) == 3 {
							defaultValue = matches[2] // 預設值（如果有）
						}
						envValue := os.Getenv(envKey)
						if envValue != "" {
							v[i] = envValue
						} else if defaultValue != "" {
							v[i] = defaultValue
						}
					}
				}
			}
			// 更新 slice 的值
			viper.Set(key, v)

		default:
			// 其他類型不處理
		}
	}
}

func getGinMode(mode string) string {
	switch mode {
	case gin.ReleaseMode:
		return gin.ReleaseMode
	case gin.TestMode:
		return gin.TestMode
	default:
		return gin.DebugMode
	}
}
