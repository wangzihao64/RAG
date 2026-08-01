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

	LLMProvider string
	LLMBaseURL  string
	LLMModel    string
	LLMAPIKey   string // 复用环境变量 DASHSCOPE_API_KEY
	QueryTopK   int
)

func LoadServer(file *ini.File) {
	section := file.Section("service")
	AppModel = envOrConfig("APP_MODEL", section.Key("AppModel").String())
	HttpPort = envOrConfig("HTTP_PORT", section.Key("HttpPort").String())
}
func LoadPostgreSQL(file *ini.File) {
	section := file.Section("postgresql")
	DB = envOrConfig("DB_DRIVER", section.Key("DB").String())
	DbHost = envOrConfig("DB_HOST", section.Key("DbHost").String())
	DbPort = envOrConfig("DB_PORT", section.Key("DbPort").String())
	DbUser = envOrConfig("DB_USER", section.Key("DbUser").String())
	DbPassword = envOrConfig("DB_PASSWORD", section.Key("DbPassword").String())
	DbName = envOrConfig("DB_NAME", section.Key("DbName").String())
}
func LoadJWT(file *ini.File) {
	JwtSecret = envOrConfig("JWT_SECRET", file.Section("jwt").Key("Secret").String())
	JwtExpireHours = file.Section("jwt").Key("ExpireHours").MustInt(72)
}
func LoadStorage(file *ini.File) {
	UploadDir = envOrConfig("UPLOAD_DIR", file.Section("storage").Key("UploadDir").MustString("./uploads"))
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
	section := file.Section("milvus")
	MilvusAddress = envOrConfig("MILVUS_ADDRESS", section.Key("Address").MustString("127.0.0.1:19530"))
	MilvusCollection = envOrConfig("MILVUS_COLLECTION", section.Key("Collection").MustString("rag_chunks"))
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
func LoadLLM(file *ini.File) {
	LLMProvider = file.Section("llm").Key("Provider").MustString("dashscope")
	LLMBaseURL = file.Section("llm").Key("BaseURL").MustString("https://dashscope.aliyuncs.com/compatible-mode/v1")
	LLMModel = file.Section("llm").Key("Model").MustString("qwen-plus")
	QueryTopK = file.Section("llm").Key("QueryTopK").MustInt(5)
	// 与 embedding 共用同一个 DashScope key
	LLMAPIKey = os.Getenv("DASHSCOPE_API_KEY")
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
	LoadLLM(file)
}

func envOrConfig(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
