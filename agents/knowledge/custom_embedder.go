package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudwego/eino/components/embedding"
)

// CustomEmbedder 自定义 Embedding 实现，支持多模态模型
type CustomEmbedder struct {
	APIKey  string
	BaseURL string
	Model   string
	client  *http.Client
}

// NewCustomEmbedder 创建一个新的 CustomEmbedder
func NewCustomEmbedder(apiKey, baseURL, model string) *CustomEmbedder {
	return &CustomEmbedder{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// MultimodalInput 多模态输入结构
type MultimodalInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MultimodalEmbeddingRequest 多模态向量化请求结构
type MultimodalEmbeddingRequest struct {
	Model          string            `json:"model"`
	EncodingFormat string            `json:"encoding_format"`
	Input          []MultimodalInput `json:"input"`
}

// MultimodalEmbeddingResponse 多模态向量化响应结构
type MultimodalEmbeddingResponse struct {
	Data struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed 实现 schema.Embedder 接口
func (e *CustomEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	// 构建请求体
	input := make([]MultimodalInput, len(texts))
	for i, text := range texts {
		input[i] = MultimodalInput{
			Type: "text",
			Text: text,
		}
	}

	request := MultimodalEmbeddingRequest{
		Model:          e.Model,
		EncodingFormat: "float",
		Input:          input,
	}

	// 序列化请求体
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+"/embeddings/multimodal", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.APIKey)

	// 发送请求
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 调用失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var response MultimodalEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取嵌入向量并转换为 float64
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		float64Embedding := make([]float64, len(response.Data.Embedding))
		for j, val := range response.Data.Embedding {
			float64Embedding[j] = float64(val)
		}
		embeddings[i] = float64Embedding
	}

	return embeddings, nil
}

// EmbedQuery 实现 schema.Embedder 接口
func (e *CustomEmbedder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("嵌入向量为空")
	}
	return embeddings[0], nil
}

// EmbedStrings 实现 embedding.Embedder 接口
func (e *CustomEmbedder) EmbedStrings(ctx context.Context, texts []string, options ...embedding.Option) ([][]float64, error) {
	// 调用 Embed 方法获取 float64 类型的嵌入向量
	float64Embeddings, err := e.Embed(ctx, texts)
	if err != nil {
		return nil, err
	}

	return float64Embeddings, nil
}
