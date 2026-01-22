-- 为 grammar_errors 表添加 error_type 字段
-- 执行此脚本来更新现有数据库

-- 1. 添加 error_type 字段
ALTER TABLE grammar_errors
    ADD COLUMN error_type VARCHAR(50) NOT NULL DEFAULT '错误1'
        AFTER llm_explanation;

-- 2. 为 error_type 字段添加索引以提高查询性能
ALTER TABLE grammar_errors
    ADD INDEX idx_error_type (error_type);

-- 3. 如果需要,可以更新现有数据
-- 例如:根据某些规则将现有错误分类
-- UPDATE grammar_errors SET error_type = '错误2' WHERE 某个条件;

-- 4. 验证更改
SELECT
    error_type,
    COUNT(*) as count
FROM grammar_errors
GROUP BY error_type;

-- 查看表结构
DESCRIBE grammar_errors;