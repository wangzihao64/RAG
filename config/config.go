package config

import (
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
func Init() {
	file, err := ini.Load("./config/config.ini")
	if err != nil {
		panic(err)
	}
	LoadServer(file)
	LoadPostgreSQL(file)
}
