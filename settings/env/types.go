package env

type ServerConfig struct {
	Port string
}

type SigningConfig struct {
	TTL int
}

type GinConfig struct {
	Mode string
}

type DynamodbConfig struct {
	DynamodbLocal *struct {
		Endpoint    string
		AccessKey   string
		AccessKeyId string
	}
	Region string
}

type GoogleOAuth2 struct {
	ClientID     string
	ClientSecret string
}

type HttpAllows struct {
	Origins []string
}

type LogConfig struct {
	Level string
}

type Config struct {
	ServerConfig   ServerConfig
	SigningConfig  SigningConfig
	GinConfig      GinConfig
	DynamodbConfig DynamodbConfig
	GoogleOAuth2   GoogleOAuth2
	HttpAllows     HttpAllows
	LogConfig      LogConfig
}
