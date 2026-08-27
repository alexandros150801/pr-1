package utils

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config — конфигурация приложения.
type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Name     string `yaml:"name"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"database"`
	Log struct {
		File       string `yaml:"file"`
		Level      string `yaml:"level"`
		MaxSizeMB  int    `yaml:"maxSizeMB"`
		MaxBackups int    `yaml:"maxBackups"`
	} `yaml:"log"`
	Export struct {
		DefaultDir string `yaml:"defaultDir"`
	} `yaml:"export"`
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.Name = "order_constructor"
	cfg.Database.User = "postgres"
	cfg.Database.Password = ""
	cfg.Log.File = "app.log"
	cfg.Log.Level = "info"
	cfg.Log.MaxSizeMB = 10
	cfg.Log.MaxBackups = 3
	cfg.Export.DefaultDir = ""
	return cfg
}

// AppDir возвращает каталог данных приложения (рядом с исполняемым файлом).
func AppDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// LoadConfig читает конфигурацию из config.yaml, при отсутствии — создаёт файл по умолчанию.
// Переменные окружения имеют приоритет: OC_DB_HOST, OC_DB_PORT, OC_DB_NAME, OC_DB_USER, OC_DB_PASSWORD.
func LoadConfig() (*Config, string, error) {
	cfg := DefaultConfig()

	// поиск файла конфигурации: рядом с исполняемым файлом или в текущем каталоге
	path := "config.yaml"
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "config.yaml")
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
		} else if _, err := os.Stat(path); err != nil {
			// файла нет нигде — создаём рядом с исполняемым файлом (если возможно), иначе в текущем каталоге
			if f, err := os.Create(candidate); err == nil {
				_ = f.Close()
				path = candidate
			}
		}
	}

	data, err := os.ReadFile(path)
	if err == nil {
		_ = yaml.Unmarshal(data, cfg)
	} else {
		// создаём файл с дефолтными значениями (игнорируем ошибки записи)
		if out, merr := yaml.Marshal(cfg); merr == nil {
			_ = os.WriteFile(path, out, 0o644)
		}
	}

	// переменные окружения
	if v := os.Getenv("OC_DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("OC_DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = p
		}
	}
	if v := os.Getenv("OC_DB_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("OC_DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("OC_DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}

	return cfg, path, nil
}
