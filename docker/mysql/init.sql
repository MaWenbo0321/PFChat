-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS im_system CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE im_system;

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
                                       `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                       `username` varchar(255) NOT NULL,
    `password` varchar(255) NOT NULL,
    `country` varchar(100) NOT NULL,
    `role` varchar(20) DEFAULT 'user',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `deleted_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`),
    KEY `idx_users_deleted_at` (`deleted_at`),
    KEY `idx_role` (`role`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 消息表
CREATE TABLE IF NOT EXISTS `messages` (
                                          `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                          `sender_id` bigint unsigned NOT NULL,
                                          `receiver_id` bigint unsigned NOT NULL,
                                          `content` text NOT NULL,
                                          `is_read` tinyint(1) DEFAULT '0',
    `sent_at` datetime(3) DEFAULT NULL,
    `created_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_sender_id` (`sender_id`),
    KEY `idx_receiver_id` (`receiver_id`),
    KEY `idx_sent_at` (`sent_at`),
    CONSTRAINT `fk_messages_sender` FOREIGN KEY (`sender_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_messages_receiver` FOREIGN KEY (`receiver_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 语法错误记录表
CREATE TABLE IF NOT EXISTS `grammar_errors` (
                                                `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                                `user_id` bigint unsigned NOT NULL,
                                                `message_id` bigint unsigned NULL,
                                                `original_text` text NOT NULL,
                                                `llm_suggestion` text NOT NULL,
                                                `llm_explanation` text,
                                                `created_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_message_id` (`message_id`),
    KEY `idx_created_at` (`created_at`),
    CONSTRAINT `fk_grammar_errors_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_grammar_errors_message` FOREIGN KEY (`message_id`) REFERENCES `messages` (`id`) ON DELETE SET NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建索引优化查询性能
CREATE INDEX idx_messages_conversation ON messages(sender_id, receiver_id, sent_at);
CREATE INDEX idx_grammar_errors_user_created ON grammar_errors(user_id, created_at DESC);