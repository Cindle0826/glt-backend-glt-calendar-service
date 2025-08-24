package env

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
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

	if mode := viper.GetString("gin.mode"); mode == gin.TestMode || mode == gin.ReleaseMode {
		if err := loadSSMParameters(context.Background()); err != nil {
			// 根據需求可改為 panic 或只是 log，這裡選擇 panic 讓部署立即發現錯誤
			panic(fmt.Sprintf("load ssm parameters failed: %v", err))
		}
	}

	config := Config{
		ServerConfig: ServerConfig{Port: viper.GetString("server.port")},
		SigningConfig: SigningConfig{
			TTL: viper.GetInt("signin.ttl"),
		},
		GinConfig: GinConfig{Mode: viper.GetString("gin.mode")},
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
				} else {
					viper.Set(key, "")
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

func loadSSMParameters(ctx context.Context) error {
	// 從 viper 讀取 mappings
	mappings := viper.GetStringMapString("ssm.mappings")
	// 修正 S1009: 直接檢查長度，不需要先檢查是否為 nil
	if len(mappings) == 0 {
		log.Println("ssm.enabled is true but ssm.mappings is empty, skip")
		return nil
	}

	// 反向索引 paramName -> viperKey
	paramToViper := make(map[string]string, len(mappings))
	names := make([]string, 0, len(mappings))
	for viperKey, paramName := range mappings {
		paramToViper[paramName] = viperKey
		names = append(names, paramName)
	}

	// 讀取可選的自訂 endpoint（用於本地模擬器）
	endpoint := strings.TrimSpace(viper.GetString("ssm.endpoint"))

	// 初始化 AWS client（使用預設 credential/provider）
	awsRegion := viper.GetString("ssm.region")
	var cfgOpts []func(*awsconfig.LoadOptions) error
	if awsRegion != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithRegion(awsRegion))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	// 建立 SSM client，若指定 endpoint，使用自訂 endpoint resolver（使用新的 BaseEndpoint 方式）
	var client *ssm.Client
	if endpoint != "" {
		client = ssm.NewFromConfig(awsCfg, func(o *ssm.Options) {
			// 使用新的 BaseEndpoint 方式取代已棄用的 EndpointResolver
			o.BaseEndpoint = aws.String(endpoint)
		})
	} else {
		client = ssm.NewFromConfig(awsCfg)
	}

	// SSM GetParameters 一次最多 10 個
	for i := 0; i < len(names); i += 10 {
		end := i + 10
		if end > len(names) {
			end = len(names)
		}
		batch := names[i:end]

		out, err := client.GetParameters(ctx, &ssm.GetParametersInput{
			Names:          batch,
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return fmt.Errorf("ssm GetParameters failed: %w", err)
		}

		// 設定回 viper
		for _, p := range out.Parameters {
			paramName := aws.ToString(p.Name)
			value := aws.ToString(p.Value)
			if vkey, ok := paramToViper[paramName]; ok {
				viper.Set(vkey, value)
				log.Printf("ssm param loaded: %s -> viper.%s\n", paramName, vkey)
			}
		}

		// log 未找到的 parameters（Optional）
		if len(out.InvalidParameters) > 0 {
			log.Printf("ssm invalid parameters: %v\n", out.InvalidParameters)
		}
	}

	return nil
}
