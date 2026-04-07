package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"schema"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
}

type Config struct {
	DB        MySQLConfig `yaml:"MySQL"`
	Redis     RedisConfig `yaml:"Redis"`
	JWTSecret string      `yaml:"jwt_secret"`
}

var AppConfig *Config

func Init() {
	// 读取配置文件
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		panic("读取配置文件失败: " + err.Error())
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		panic("解析配置文件失败: " + err.Error())
	}

	// 环境变量覆盖（生产环境用）
	// if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
	// 	cfg.DB.Host = dbHost
	// }
	// if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
	// 	cfg.DB.Password = dbPassword
	// }
	// if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
	// 	cfg.JWTSecret = jwtSecret
	// }

	AppConfig = &cfg
}

// 方便使用的方法
func (c *MySQLConfig) DSN() string {
	return c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + c.Port + ")/" + c.Database + "?charset=utf8mb4&parseTime=True"
}
