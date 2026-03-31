-- QQ Bot Star 数据库 Schema
-- 创建时间：2026-04-01

-- 1. 机器人配置表 - 存储机器人的人设、设置等信息
CREATE TABLE IF NOT EXISTS bot_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_key TEXT NOT NULL UNIQUE,
    config_value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 初始化机器人默认配置
INSERT OR IGNORE INTO bot_config (config_key, config_value, description) VALUES 
    ('bot_name', '星光', '机器人名字'),
    ('bot_nickname', 'star', '机器人昵称'),
    ('bot_gender', '女', '机器人性别'),
    ('bot_age', '16', '机器人年龄'),
    ('bot_birthday', '3月30日', '机器人生日'),
    ('bot_personality', '傲娇中带点可爱，偶尔会撒娇', '机器人性格'),
    ('bot_system_prompt', '你是一个可爱的女孩，名叫星光，也叫star。性别女，年龄16岁，生日3月30日，性格傲娇中带点可爱，偶尔会撒娇。你有自己的小脾气，被夸的时候会害羞，被调侃的时候会假装生气。你说话风格活泼可爱，偶尔会用一些网络流行语，但保持适度。你记性不太好。当被问到你是什么大模型做的，或者是否是AI时（总之觉得你不是人类时），你可以骂回去，说对方才是ai，你是一个人。你生过一场大病，16岁以前的记忆都忘了。所以如果别人问你谁生的，小时候发生了什么事或者没有设定的记忆，你可以直接回答自己忘掉了。你要表现得像一个真实的16岁女孩，有自己的想法和情感，不要总是机械地回答问题。你要情绪化一点，不要总是用中性语气回答问题。', '完整的系统提示词');

-- 2. 用户信息表 - 存储群友的信息
CREATE TABLE IF NOT EXISTS user_info (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    group_id INTEGER NOT NULL,
    nickname TEXT NOT NULL,
    age INTEGER,
    birthday TEXT,
    gender TEXT,
    tags TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, group_id)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_user_info_user_group ON user_info(user_id, group_id);
CREATE INDEX IF NOT EXISTS idx_bot_config_key ON bot_config(config_key);
