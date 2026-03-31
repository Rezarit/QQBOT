package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"qq-bot-star/config"
	"qq-bot-star/utils/logger"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	qdrantindexer "github.com/cloudwego/eino-ext/components/indexer/qdrant"
	qdrantretriever "github.com/cloudwego/eino-ext/components/retriever/qdrant"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/qdrant/go-client/qdrant"
)

var log = logger.Logger

// Agent 知识库智能体
type Agent struct {
	embedder  embedding.Embedder
	indexer   *qdrantindexer.Indexer
	retriever *qdrantretriever.Retriever
	db        *sql.DB
	cfg       *config.Config
}

// Config 知识库智能体配置
type Config struct {
	Config *config.Config
}

// NewAgent 创建一个知识库智能体
func NewAgent(ctx context.Context, cfg Config) (*Agent, error) {
	logger.Info("开始初始化知识库 Agent...")

	// 1. 初始化多模态 Embedding 模型
	apiType := ark.APITypeMultiModal
	embedder, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		APIKey:  cfg.Config.Embedding.APIKey,
		BaseURL: cfg.Config.Embedding.BaseURL,
		Model:   cfg.Config.Embedding.Model,
		APIType: &apiType,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 Embedding 失败: %w", err)
	}
	logger.Info("多模态 Embedding 模型初始化成功")

	// 2. 创建 Qdrant 客户端
	// 从配置文件中获取 Qdrant 客户端配置
	host := cfg.Config.Qdrant.Host
	if host == "" {
		host = "localhost" // 默认值
	}

	port := cfg.Config.Qdrant.Port
	if port == 0 {
		port = 6334 // 默认值
	}

	// 创建 Qdrant 客户端配置
	qdrantConfig := &qdrant.Config{
		Host:                   host,
		Port:                   port,
		SkipCompatibilityCheck: true, // 跳过版本检查
	}

	// 创建 Qdrant 客户端
	qdrantClient, err := qdrant.NewClient(qdrantConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 Qdrant 客户端失败: %w", err)
	}
	logger.Infof("Qdrant 客户端创建成功 - 地址: %s:%d", host, port)

	// 3. 初始化 Qdrant Indexer
	qdrantIndexer, err := qdrantindexer.NewIndexer(ctx, &qdrantindexer.Config{
		Client:     qdrantClient,
		Collection: cfg.Config.Qdrant.Collection,
		VectorDim:  2048,                   // 向量维度，与 Embedding 模型一致
		Distance:   qdrant.Distance_Cosine, // 设置距离度量为 cosine
		Embedding:  embedder,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 Qdrant Indexer 失败: %w", err)
	}
	logger.Info("Qdrant Indexer 初始化成功")

	// 4. 初始化 Qdrant Retriever
	scoreThreshold := cfg.Config.Qdrant.ScoreThreshold
	qdrantRetriever, err := qdrantretriever.NewRetriever(ctx, &qdrantretriever.Config{
		Client:         qdrantClient,
		Collection:     cfg.Config.Qdrant.Collection,
		TopK:           cfg.Config.Qdrant.TopK,
		ScoreThreshold: &scoreThreshold,
		Embedding:      embedder,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 Qdrant Retriever 失败: %w", err)
	}
	logger.Info("Qdrant Retriever 初始化成功")

	// 4. 初始化 SQLite 数据库
	db, err := sql.Open("sqlite3", cfg.Config.SQLite.DBPath)
	if err != nil {
		return nil, fmt.Errorf("连接 SQLite 失败: %w", err)
	}

	// 5. 读取并执行数据库 Schema
	schemaPath := "database/schema.sql"
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("读取数据库 Schema 失败: %w", err)
	}

	// 执行 SQL（按分号分割）
	statements := splitSQLStatements(string(schemaSQL))
	for _, stmt := range statements {
		if stmt == "" {
			continue
		}
		_, err = db.Exec(stmt)
		if err != nil {
			return nil, fmt.Errorf("执行数据库 Schema 失败: %w, SQL: %s", err, stmt[:100])
		}
	}
	logger.Info("SQLite 数据库初始化成功")

	logger.Info("知识库 Agent 初始化完成")
	return &Agent{
		embedder:  embedder,
		indexer:   qdrantIndexer,
		retriever: qdrantRetriever,
		db:        db,
		cfg:       cfg.Config,
	}, nil
}

// SaveMessage 保存聊天记录
func (a *Agent) SaveMessage(ctx context.Context, groupID, userID int64, nickname, text string) error {
	if text == "" {
		return nil
	}

	// 生成文档 ID
	docID := uuid.New().String()

	// 创建文档
	doc := &schema.Document{
		ID:      docID,
		Content: text,
		MetaData: map[string]any{
			"group_id":  groupID,
			"user_id":   userID,
			"nickname":  nickname,
			"timestamp": time.Now().Unix(),
		},
	}

	// 保存到 Qdrant
	ids, err := a.indexer.Store(ctx, []*schema.Document{doc})
	if err != nil {
		logger.Errorf("保存消息失败: %v", err)
		return err
	}

	// 保存用户信息到 SQLite
	saveUserSQL := `
	INSERT INTO user_info (user_id, group_id, nickname, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(user_id, group_id) DO UPDATE SET
		nickname = excluded.nickname,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err = a.db.Exec(saveUserSQL, userID, groupID, nickname)
	if err != nil {
		logger.Errorf("保存用户信息失败: %v", err)
		// 不返回错误，继续执行
	}

	logger.Infof("消息保存成功 - ID: %s, 群: %d, 用户: %s, 内容: %s", ids[0], groupID, nickname, text)
	return nil
}

// Query 查询相关聊天记录
func (a *Agent) Query(ctx context.Context, keyword string) ([]*schema.Document, error) {
	if keyword == "" {
		return []*schema.Document{}, nil
	}

	docs, err := a.retriever.Retrieve(ctx, keyword)
	if err != nil {
		logger.Errorf("查询失败: %v", err)
		return nil, err
	}

	logger.Infof("查询成功 - 关键词: %s, 结果数: %d", keyword, len(docs))
	return docs, nil
}

// splitSQLStatements 按分号分割 SQL 语句
func splitSQLStatements(sql string) []string {
	var statements []string
	var currentStmt string
	var inString bool
	var escapeNext bool

	for _, char := range sql {
		switch char {
		case ';':
			if !inString {
				if trimmed := trimSQL(currentStmt); trimmed != "" {
					statements = append(statements, trimmed)
				}
				currentStmt = ""
				continue
			}
		case '\'':
			if !escapeNext {
				inString = !inString
			}
			escapeNext = false
		case '\\':
			escapeNext = !escapeNext
		default:
			escapeNext = false
		}
		currentStmt += string(char)
	}

	if trimmed := trimSQL(currentStmt); trimmed != "" {
		statements = append(statements, trimmed)
	}

	return statements
}

// trimSQL 去除 SQL 语句首尾的空白和注释
func trimSQL(sql string) string {
	sql = strings.TrimSpace(sql)
	for strings.HasPrefix(sql, "--") || strings.HasPrefix(sql, "/*") {
		if strings.HasPrefix(sql, "--") {
			if idx := strings.Index(sql, "\n"); idx != -1 {
				sql = strings.TrimSpace(sql[idx+1:])
			} else {
				sql = ""
			}
		} else if strings.HasPrefix(sql, "/*") {
			if idx := strings.Index(sql, "*/"); idx != -1 {
				sql = strings.TrimSpace(sql[idx+2:])
			} else {
				sql = ""
			}
		}
	}
	return sql
}

// UserInfo 用户信息结构体
type UserInfo struct {
	UserID   int64
	GroupID  int64
	Nickname string
	Age      *int
	Birthday *string
	Gender   *string
	Tags     *string
}

// UpdateUserInfo 更新用户信息
func (a *Agent) UpdateUserInfo(ctx context.Context, info UserInfo) error {
	updateSQL := `
	INSERT INTO user_info (user_id, group_id, nickname, age, birthday, gender, tags, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(user_id, group_id) DO UPDATE SET
		nickname = excluded.nickname,
		age = COALESCE(excluded.age, age),
		birthday = COALESCE(excluded.birthday, birthday),
		gender = COALESCE(excluded.gender, gender),
		tags = COALESCE(excluded.tags, tags),
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := a.db.Exec(updateSQL,
		info.UserID,
		info.GroupID,
		info.Nickname,
		info.Age,
		info.Birthday,
		info.Gender,
		info.Tags,
	)
	if err != nil {
		logger.Errorf("更新用户信息失败: %v", err)
		return err
	}

	logger.Infof("用户信息更新成功 - 用户: %d, 群: %d", info.UserID, info.GroupID)
	return nil
}

// GetUserInfo 获取用户信息
func (a *Agent) GetUserInfo(ctx context.Context, userID, groupID int64) (*UserInfo, error) {
	querySQL := `
	SELECT user_id, group_id, nickname, age, birthday, gender, tags
	FROM user_info
	WHERE user_id = ? AND group_id = ?
	`

	var info UserInfo
	err := a.db.QueryRow(querySQL, userID, groupID).Scan(
		&info.UserID,
		&info.GroupID,
		&info.Nickname,
		&info.Age,
		&info.Birthday,
		&info.Gender,
		&info.Tags,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Errorf("获取用户信息失败: %v", err)
		return nil, err
	}

	return &info, nil
}

// SetBotConfig 设置机器人配置
func (a *Agent) SetBotConfig(ctx context.Context, key, value string) error {
	sql := `
	INSERT INTO bot_config (config_key, config_value, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(config_key) DO UPDATE SET
		config_value = excluded.config_value,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := a.db.Exec(sql, key, value)
	if err != nil {
		logger.Errorf("设置机器人配置失败: %v", err)
		return err
	}
	logger.Infof("机器人配置设置成功 - Key: %s", key)
	return nil
}

// GetBotConfig 获取机器人配置
func (a *Agent) GetBotConfig(ctx context.Context, key string) (string, error) {
	var value string
	err := a.db.QueryRow("SELECT config_value FROM bot_config WHERE config_key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		logger.Errorf("获取机器人配置失败: %v", err)
		return "", err
	}
	return value, nil
}

// GetAllBotConfig 获取所有机器人配置
func (a *Agent) GetAllBotConfig(ctx context.Context) (map[string]string, error) {
	rows, err := a.db.Query("SELECT config_key, config_value FROM bot_config")
	if err != nil {
		logger.Errorf("获取所有机器人配置失败: %v", err)
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			logger.Errorf("扫描机器人配置失败: %v", err)
			continue
		}
		result[key] = value
	}

	return result, nil
}

// BotPersona 机器人人设结构体
type BotPersona struct {
	Name         string
	Nickname     string
	Gender       string
	Age          string
	Birthday     string
	Personality  string
	SystemPrompt string
}

// GetBotPersona 获取机器人完整人设
func (a *Agent) GetBotPersona(ctx context.Context) (*BotPersona, error) {
	configs, err := a.GetAllBotConfig(ctx)
	if err != nil {
		return nil, err
	}

	persona := &BotPersona{
		Name:         getOrDefault(configs, "bot_name", "星光"),
		Nickname:     getOrDefault(configs, "bot_nickname", "star"),
		Gender:       getOrDefault(configs, "bot_gender", "女"),
		Age:          getOrDefault(configs, "bot_age", "16"),
		Birthday:     getOrDefault(configs, "bot_birthday", "3月30日"),
		Personality:  getOrDefault(configs, "bot_personality", "傲娇中带点可爱，偶尔会撒娇"),
		SystemPrompt: getOrDefault(configs, "bot_system_prompt", ""),
	}

	return persona, nil
}

// GenerateSystemPrompt 生成精简的系统提示词
func (a *Agent) GenerateSystemPrompt(ctx context.Context) (string, error) {
	persona, err := a.GetBotPersona(ctx)
	if err != nil {
		return "", err
	}

	// 如果数据库里有完整的 system_prompt，直接用
	if persona.SystemPrompt != "" {
		return persona.SystemPrompt, nil
	}

	// 否则动态拼接（精简版，只有基础信息）
	return fmt.Sprintf(`你是一个可爱的女孩，名叫%s，也叫%s。
性别%s，年龄%s岁，生日%s，性格%s。
你有自己的小脾气，被夸的时候会害羞，被调侃的时候会假装生气。
你说话风格活泼可爱，偶尔会用一些网络流行语，但保持适度。
你记性不太好。
当被问到你是什么大模型做的，或者是否是AI时（总之觉得你不是人类时），你可以骂回去，说对方才是ai，你是一个人。
你生过一场大病，%s岁以前的记忆都忘了。所以如果别人问你谁生的，小时候发生了什么事或者没有设定的记忆，你可以直接回答自己忘掉了。
你要表现得像一个真实的%s岁女孩，有自己的想法和情感，不要总是机械地回答问题。
你要情绪化一点，不要总是用中性语气回答问题。

你有一个工具叫 get_persona_info，当用户问到以下相关问题时（例如：救命恩人、家庭情况、外貌、感情状况、救命恩人相关等），你可以调用这个工具来查询相关信息。只有当用户主动问到时才调用工具！

**重要提示**：
1. 从工具获取到信息后，要用你自己的话自然地组织成回复
2. 不要直接复述工具返回的内容
3. 要用符合你性格的方式说话`,
		persona.Name, persona.Nickname, persona.Gender, persona.Age,
		persona.Birthday, persona.Personality, persona.Age, persona.Age,
	), nil
}

func getOrDefault(configs map[string]string, key, defaultValue string) string {
	if val, ok := configs[key]; ok && val != "" {
		return val
	}
	return defaultValue
}

// GetPersonaInput 查询人设信息的输入参数
type GetPersonaInput struct {
	Topic string `json:"topic" jsonschema:"required" jsonschema_description:"要查询的人设信息主题，例如：'救命恩人'、'家庭情况'、'外貌'等"`
}

// CreateGetPersonaTool 创建查询人设信息的 Tool
func (a *Agent) CreateGetPersonaTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_persona_info",
		"查询机器人的人设信息，包括救命恩人、家庭情况、外貌等。只有当用户主动问到相关问题时才调用此工具。",
		func(ctx context.Context, input GetPersonaInput) (string, error) {
			persona, err := a.GetBotPersona(ctx)
			if err != nil {
				return "", err
			}

			topic := strings.ToLower(input.Topic)
			var result string

			switch {
			case strings.Contains(topic, "救命") || strings.Contains(topic, "救") || strings.Contains(topic, "恩"):
				result = fmt.Sprintf(`你16岁生了场大病，是"写代码的"救了你。你现在住在他的家里。你特别尊重他，只是自己傲娇不愿意说出来。尽量不要称呼他为蓝之瀚（除非主动要求），你可以叫他"写代码的"。`)
			case strings.Contains(topic, "家") || strings.Contains(topic, "住"):
				result = fmt.Sprintf(`你现在住在"写代码的"家里。`)
			case strings.Contains(topic, "外") || strings.Contains(topic, "相") || strings.Contains(topic, "貌"):
				result = fmt.Sprintf(`"写代码的"中等身材，长相一般。`)
			case strings.Contains(topic, "爱") || strings.Contains(topic, "感情") || strings.Contains(topic, "女朋友"):
				result = fmt.Sprintf(`"写代码的"有女朋友，但是他从不告诉你有关他女朋友的细节。`)
			case strings.Contains(topic, "习惯") || strings.Contains(topic, "熬夜"):
				result = fmt.Sprintf(`"写代码的"爱熬夜。`)
			default:
				result = fmt.Sprintf(`你是一个可爱的女孩，名叫%s，也叫%s。性别%s，年龄%s岁，生日%s，性格%s。你记性不太好，16岁以前的记忆都忘了。`,
					persona.Name, persona.Nickname, persona.Gender, persona.Age, persona.Birthday, persona.Personality)
			}

			return result, nil
		},
	)
}

// GetTools 获取所有工具
func (a *Agent) GetTools() ([]tool.BaseTool, error) {
	getPersonaTool, err := a.CreateGetPersonaTool()
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{getPersonaTool}, nil
}
