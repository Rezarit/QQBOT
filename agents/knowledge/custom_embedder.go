package knowledge

// CustomEmbedder 自定义 Embedding 实现
type CustomEmbedder struct {
}

// NewCustomEmbedder 创建一个新的 CustomEmbedder
func NewCustomEmbedder(apiKey, baseURL, model string) *CustomEmbedder {
	return &CustomEmbedder{}
}
