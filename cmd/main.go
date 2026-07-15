package main

import (
	"rag/config"
	"rag/internal/dao"
)

func Init() {
	config.Init()
	dao.InitPostgreSQL()
}

func main() {
	Init()
}
