package mime

func FileContentType(fileType string) string {
	switch fileType {
	case "pdf":
		return "application/pdf"
	case "md":
		return "text/markdown; charset=utf-8"
	case "txt":
		return "text/plain; charset=utf-8"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "html":
		return "text/html; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
