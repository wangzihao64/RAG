package config

import (
	"os"
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

	MilvusAddress    string
	MilvusCollection string

	EmbeddingProvider string
	EmbeddingBaseURL  string
	EmbeddingModel    string
	EmbeddingDim      int
	EmbeddingAPIKey   string // 从环境变量 DASHSCOPE_API_KEY 读取
	ChunkSize         int
	ChunkOverlap      int
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
func LoadMilvus(file *ini.File) {
	MilvusAddress = file.Section("milvus").Key("Address").MustString("127.0.0.1:19530")
	MilvusCollection = file.Section("milvus").Key("Collection").MustString("rag_chunks")
}
func LoadEmbedding(file *ini.File) {
	EmbeddingProvider = file.Section("embedding").Key("Provider").MustString("dashscope")
	EmbeddingBaseURL = file.Section("embedding").Key("BaseURL").MustString("https://dashscope.aliyuncs.com/compatible-mode/v1")
	EmbeddingModel = file.Section("embedding").Key("Model").MustString("text-embedding-v3")
	EmbeddingDim = file.Section("embedding").Key("Dim").MustInt(1024)
	ChunkSize = file.Section("embedding").Key("ChunkSize").MustInt(500)
	ChunkOverlap = file.Section("embedding").Key("ChunkOverlap").MustInt(80)
	// 敏感信息不入配置文件，仅从环境变量读取
	EmbeddingAPIKey = os.Getenv("DASHSCOPE_API_KEY")
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
	LoadMilvus(file)
	LoadEmbedding(file)
}
