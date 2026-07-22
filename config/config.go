package config

import (
	"strings"

	"gopkg.in/ini.v1"
)

var (
	AppModel string
	HttpPort string

	DB         string
	DbHost     string
	DbPort     string
	DbUser     string
	DbPassword string
	DbName     string

	JwtSecret      string
	JwtExpireHours int

	UploadDir     string
	MaxFileSizeMB int64
	AllowedTypes  []string
)

func LoadServer(file *ini.File) {
	AppModel = file.Section("service").Key("AppModel").String()
	HttpPort = file.Section("service").Key("HttpPort").String()
}
func LoadPostgreSQL(file *ini.File) {
	DB = file.Section("postgresql").Key("DB").String()
	DbHost = file.Section("postgresql").Key("DbHost").String()
	DbPort = file.Section("postgresql").Key("DbPort").String()
	DbUser = file.Section("postgresql").Key("DbUser").String()
	DbPassword = file.Section("postgresql").Key("DbPassword").String()
	DbName = file.Section("postgresql").Key("DbName").String()
}
func LoadJWT(file *ini.File) {
	JwtSecret = file.Section("jwt").Key("Secret").String()
	JwtExpireHours = file.Section("jwt").Key("ExpireHours").MustInt(72)
}
func LoadStorage(file *ini.File) {
	UploadDir = file.Section("storage").Key("UploadDir").MustString("./uploads")
	MaxFileSizeMB = file.Section("storage").Key("MaxFileSizeMB").MustInt64(50)
	// 允许的文件类型，逗号分隔，统一转小写
	raw := file.Section("storage").Key("AllowedTypes").MustString("pdf,md,txt,docx")
	AllowedTypes = nil
	for _, t := range strings.Split(raw, ",") {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			AllowedTypes = append(AllowedTypes, t)
		}
	}
}
func Init() {
	file, err := ini.Load("./config/config.ini")
	if err != nil {
		panic(err)
	}
	LoadServer(file)
	LoadPostgreSQL(file)
	LoadJWT(file)
	LoadStorage(file)
}
