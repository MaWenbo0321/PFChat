-- 数据库迁移脚本：添加用户角色功能
-- 执行前请备份数据库

USE im_system;

-- 1. 检查是否已经有 role 字段，如果没有则添加
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role varchar(20) DEFAULT 'user' AFTER country;

-- 2. 更新现有用户的角色为普通用户（如果字段为空）
UPDATE users SET role = 'user' WHERE role IS NULL OR role = '';

-- 3. 添加角色索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 4. 创建默认管理员账号（如果不存在）
-- 注意：密码是 'admin123456' 的 bcrypt 哈希值，请在首次登录后修改
INSERT IGNORE INTO users (username, password, country, role, created_at, updated_at)
VALUES (
    'admin',
    '$2a$10$3k5zKFxNxEQO9zOOmF5jFe8YeOCvKFxGKOXZZxGQ5Q6QOXZGQxGQ6',
    'CN',
    'admin',
    NOW(),
    NOW()
);

-- 5. 验证数据
SELECT 'Migration completed. Admin account created.' as status;
SELECT username, role, created_at FROM users WHERE role = 'admin';

-- 迁移完成后的说明：
-- 1. 默认管理员账号：admin / admin123456
-- 2. 请立即登录并修改管理员密码
-- 3. 所有现有用户都被设置为普通用户角色