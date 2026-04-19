-- MySQL 8.0+

SET NAMES utf8mb4;
SET time_zone = '+00:00';

-- =========================================
-- Admins
-- =========================================
CREATE TABLE IF NOT EXISTS admins (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100) NOT NULL,
    specialization VARCHAR(100) NOT NULL DEFAULT '',
    must_reset_password TINYINT(1) NOT NULL DEFAULT 0,
    two_factor_secret VARCHAR(255) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_admins_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Seed initial administrator account
-- PostgreSQL 的 crypt('12345678', gen_salt('bf')) 在 MySQL 没有同等内置 bcrypt。
-- 这里用 SHA2 示例（生产建议：由应用层生成 bcrypt/argon2 后再写入 password_hash）。
INSERT INTO admins (username, password_hash, nickname, specialization, must_reset_password)
VALUES ('1', SHA2('x', 256), 'x', '全栈工程师', 1)
ON DUPLICATE KEY UPDATE username = username;

-- Admin recovery codes
CREATE TABLE IF NOT EXISTS admin_recovery_codes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    admin_id BIGINT UNSIGNED NOT NULL,
    code_hash VARCHAR(255) NOT NULL,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_admin_recovery_admin_code (admin_id, code_hash),
    KEY idx_admin_recovery_admin_id (admin_id),
    CONSTRAINT fk_admin_recovery_admin
        FOREIGN KEY (admin_id) REFERENCES admins (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Users
-- =========================================
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar VARCHAR(500) NOT NULL DEFAULT '/avatar.png',
    bio TEXT NOT NULL,
    status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
    email_verified TINYINT(1) NOT NULL DEFAULT 0,
    region VARCHAR(255) NULL,
    blog_url VARCHAR(500) NULL,
    auth_provider VARCHAR(50) NULL,
    auth_openid VARCHAR(255) NULL,
    show_full_profile TINYINT(1) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),

    -- email: MySQL 的 UNIQUE 允许多个 NULL（等价于 PG 的 “partial unique where email is not null”）
    UNIQUE KEY uk_users_email (email),

    -- auth provider constraints
    CONSTRAINT ck_users_auth_provider_allowed CHECK (
        auth_provider IS NULL OR auth_provider IN ('qq')
    ),
    UNIQUE KEY uk_users_auth_pair (auth_provider, auth_openid),
    CONSTRAINT ck_users_auth_pair_nullness CHECK (
        (auth_provider IS NULL AND auth_openid IS NULL)
        OR
        (auth_provider IS NOT NULL AND auth_openid IS NOT NULL)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- PG 默认 bio 有中文默认值；MySQL 对 TEXT 默认值限制较多（不同版本/参数不一致），这里改为插入时由应用层填充；
-- 如果你必须在库层兜底，可以改成 VARCHAR(....) 或用触发器/生成列方案。
-- 这里先给一个一致的默认文本（通过把 bio 改成 VARCHAR 来实现）；如果不想改字段类型，请告诉我。
-- （当前保持 TEXT，但不设 DEFAULT）

-- =========================================
-- Refresh token blacklist
-- =========================================
CREATE TABLE IF NOT EXISTS refresh_token_blacklist (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_refresh_token_hash (token_hash),
    KEY idx_refresh_token_blacklist_user_id (user_id),
    KEY idx_refresh_token_blacklist_expires_at (expires_at),
    CONSTRAINT fk_refresh_token_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Categories
-- =========================================
CREATE TABLE IF NOT EXISTS categories (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    post_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_categories_name (name),
    UNIQUE KEY uk_categories_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Tags
-- =========================================
CREATE TABLE IF NOT EXISTS tags (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    post_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_tags_name (name),
    UNIQUE KEY uk_tags_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Posts
-- =========================================
CREATE TABLE IF NOT EXISTS posts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    excerpt TEXT NOT NULL,
    content LONGTEXT NOT NULL,
    featured_image VARCHAR(500) NULL,
    author_id BIGINT UNSIGNED NOT NULL,
    category_id BIGINT UNSIGNED NOT NULL,
    status ENUM('draft', 'published', 'archived') NOT NULL DEFAULT 'draft',
    read_time VARCHAR(20) NOT NULL,
    views INT NOT NULL DEFAULT 0,
    likes INT NOT NULL DEFAULT 0,
    is_featured TINYINT(1) NOT NULL DEFAULT 0,
    meta_title VARCHAR(255) NULL,
    meta_description TEXT NULL,
    published_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_posts_slug (slug),
    KEY idx_posts_author_id (author_id),
    KEY idx_posts_category_id (category_id),

    CONSTRAINT fk_posts_author
        FOREIGN KEY (author_id) REFERENCES admins (id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_posts_category
        FOREIGN KEY (category_id) REFERENCES categories (id)
        ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- PG 的部分索引 (WHERE status='published')：MySQL 没有 partial index
-- 这里用普通联合索引替代（会稍微“更大”，但可用）
CREATE INDEX idx_posts_category_published_at
ON posts (category_id, status, published_at DESC);

CREATE INDEX idx_posts_featured_published_at
ON posts (is_featured, status, published_at DESC);

-- PG 的 trigram GIN：MySQL 用 FULLTEXT 近似（分词/效果不同）
-- 注意：FULLTEXT 对中文通常需要 ngram parser 或外部搜索（ES/OpenSearch）。
ALTER TABLE posts
    ADD FULLTEXT KEY ft_posts_search (title, excerpt, content);

-- =========================================
-- Post tags join
-- =========================================
CREATE TABLE IF NOT EXISTS post_tags (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    post_id BIGINT UNSIGNED NOT NULL,
    tag_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_post_tags_pair (post_id, tag_id),
    KEY idx_post_tags_tag_id (tag_id),
    CONSTRAINT fk_post_tags_post
        FOREIGN KEY (post_id) REFERENCES posts (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_post_tags_tag
        FOREIGN KEY (tag_id) REFERENCES tags (id)
        ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Post likes
-- =========================================
CREATE TABLE IF NOT EXISTS post_likes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    post_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    liked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_post_likes_pair (post_id, user_id),
    KEY idx_post_likes_user_id (user_id),
    CONSTRAINT fk_post_likes_post
        FOREIGN KEY (post_id) REFERENCES posts (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_post_likes_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Post views
-- =========================================
CREATE TABLE IF NOT EXISTS post_views (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    post_id BIGINT UNSIGNED NOT NULL,
    ip_address VARCHAR(45) NOT NULL, -- PG INET -> MySQL 用 VARCHAR(45) 存 IPv4/IPv6
    user_agent TEXT NULL,
    referer VARCHAR(500) NULL,
    viewed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_post_views_post_id (post_id),
    CONSTRAINT fk_post_views_post
        FOREIGN KEY (post_id) REFERENCES posts (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Comments
-- =========================================
CREATE TABLE IF NOT EXISTS comments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    post_id BIGINT UNSIGNED NOT NULL,
    parent_id BIGINT UNSIGNED NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    content TEXT NOT NULL,
    status ENUM('pending', 'approved', 'rejected', 'spam') NOT NULL DEFAULT 'pending',
    likes INT NOT NULL DEFAULT 0,
    ip_address VARCHAR(45) NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_comments_post_id (post_id),
    KEY idx_comments_parent_id (parent_id),
    KEY idx_comments_user_id (user_id),
    CONSTRAINT fk_comments_post
        FOREIGN KEY (post_id) REFERENCES posts (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_comments_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_comments_parent
        FOREIGN KEY (parent_id) REFERENCES comments (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Comment likes
CREATE TABLE IF NOT EXISTS comment_likes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    comment_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    liked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_comment_likes_pair (comment_id, user_id),
    KEY idx_comment_likes_user_id (user_id),
    CONSTRAINT fk_comment_likes_comment
        FOREIGN KEY (comment_id) REFERENCES comments (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_comment_likes_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Site settings
-- =========================================
CREATE TABLE IF NOT EXISTS site_settings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    setting_key VARCHAR(100) NOT NULL,
    setting_value TEXT NOT NULL,
    setting_type ENUM('string', 'number', 'boolean', 'json') NOT NULL DEFAULT 'string',
    description TEXT NULL,
    is_public TINYINT(1) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_site_settings_key (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Seed site_settings
INSERT INTO site_settings (setting_key, setting_value, setting_type, description, is_public)
VALUES
    ('site.name', 'Nimbus Blog', 'string', '站点名称', 1),
    ('site.title', 'Nimbus Blog - 现代技术博客', 'string', '站点标题', 1),
    ('site.description', '专注分享前端、后端与云原生的技术文章与实践', 'string', '��点描述', 1),
    ('site.slogan', 'Where Thoughts Leave Their Trace.', 'string', '站点标语', 1),
    ('site.hero', '聚焦现代 Web 技术栈与工程实践。\n记录架构设计、性能优化与开发经验，共同成长。', 'string', '首页 Hero 介绍', 1),
    ('site.icp_record', '', 'string', 'ICP 备案号', 1),
    ('site.police_record', '', 'string', '公安备案号', 1),
    ('site.faq', '[{"title":"如何开始使用这个博客？","content":"这是一个简单的博客系统，您可以浏览文章、查看分类和标签。如果您想要更多功能，请联系管理员。"},{"title":"如何搜索文章？","content":"您可以使用导航栏中的搜索框来搜索文章。支持按标题、内容和标签进行搜索。"},{"title":"如何订阅RSS？","content":"点击导航栏中的''RSS订阅''按钮，或者直接访问 /rss.xml 来获取RSS订阅源。"},{"title":"网站支持哪些浏览器？","content":"本网站支持所有现代浏览器，包括Chrome、Firefox、Safari、Edge等。建议使用最新版本以获得最佳体验。"},{"title":"如何切换主题？","content":"点击导航栏右上角的主题切换按钮，可以在浅色模式和深色模式之间切换。"},{"title":"移动端体验如何？","content":"本网站采用响应式设计，完全适配移动设备。您可以在手机和平板上获得良好的浏览体验。"}]', 'json', '常见问题', 1),
    ('profile.name', '博主', 'string', '个人昵称', 1),
    ('profile.avatar', '/author.png', 'string', '个人头像', 1),
    ('profile.bio', '我是一名热爱开源与技术分享的开发者。\n关注前端、后端与云原生，记录实践经验与学习心得，欢迎交流。', 'string', '个人简介', 1),
    ('profile.tech_stack', '["Go","Fiber","PostgreSQL","Redis","Docker","Nginx","React","Next.js","TypeScript","MinIO"]', 'json', '技术栈', 1),
    ('profile.work_experiences', '[{"title":"Web 开发工程师","company":"互联网公司","period":"2019 - 2021","description":"参与 Web 应用开发与维护，积累工程实践。"},{"title":"全栈工程师","company":"技术团队","period":"2021 - 2023","description":"负责前后端开发与部署，推动工程效率提升。"},{"title":"技术顾问","company":"开源社区/企业","period":"2023 - 至今","description":"分享技术经验与最佳实践，参与社区建设。"}]', 'json', '工作经历', 1),
    ('profile.project_experiences', '[{"name":"内容管理平台","description":"用于管理文章、分类与标签的 CMS 系统","tech":["React","TypeScript","PostgreSQL"]},{"name":"技术博客站点","description":"基于现代前端框架构建的个人/团队博客","tech":["Next.js","Tailwind CSS","HeroUI"]},{"name":"数据分析工具","description":"用于指标采集与可视化的应用","tech":["Go","Docker","Grafana"]}]', 'json', '项目经历', 1),
    ('profile.github_url', 'https://github.com/yourname', 'string', 'GitHub 链接', 1),
    ('profile.bilibili_url', 'https://space.bilibili.com/000000000', 'string', 'Bilibili 链接', 1),
    ('profile.qq_group_url', 'https://qm.qq.com/q/XXXXXXXXXX', 'string', 'QQ 群链接', 1),
    ('profile.email', 'contact@example.com', 'string', '联系邮箱', 1)
ON DUPLICATE KEY UPDATE setting_key = setting_key;

-- =========================================
-- Feedbacks
-- =========================================
CREATE TABLE IF NOT EXISTS feedbacks (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    type ENUM('general', 'bug', 'feature', 'ui') NOT NULL DEFAULT 'general',
    subject VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    status ENUM('pending', 'processing', 'resolved', 'closed') NOT NULL DEFAULT 'pending',
    ip_address VARCHAR(45) NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_feedbacks_status (status),
    KEY idx_feedbacks_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Links
-- =========================================
CREATE TABLE IF NOT EXISTS links (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    url VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    logo VARCHAR(500) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    status ENUM('active', 'inactive') NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_links_status_sort (status, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Notifications
-- =========================================
CREATE TABLE IF NOT EXISTS notifications (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    type ENUM('comment_reply', 'comment_approved', 'admin_message') NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    meta JSON NOT NULL,
    is_read TINYINT(1) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_notifications_user_unread (user_id, is_read, created_at DESC),
    CONSTRAINT fk_notifications_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Default values for JSON in MySQL: 用 DEFAULT 有版本差异，这里用 NOT NULL + 由插入时提供。
-- 为了和 PG 默认 '{}' 一致，这里把 meta 改为 NOT NULL，插入时请写 '{}'。
-- 如果你强依赖 DEFAULT '{}'，我可以给你做成 BEFORE INSERT 触发器补全。

-- =========================================
-- Files
-- =========================================
CREATE TABLE IF NOT EXISTS files (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    object_key VARCHAR(512) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    mime_type VARCHAR(100) NOT NULL,
    file_usage ENUM('post_cover', 'post_content', 'avatar') NOT NULL,
    resource_id BIGINT UNSIGNED NULL,
    uploader_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_files_object_key (object_key),
    KEY idx_files_usage_resource (file_usage, resource_id),
    KEY idx_files_resource_id (resource_id),
    CONSTRAINT fk_files_uploader
        FOREIGN KEY (uploader_id) REFERENCES admins (id)
        ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- =========================================
-- Triggers (NO DELIMITER required)
-- Each trigger body is a single statement.
-- =========================================

-- Maintain categories.post_count based on posts changes
CREATE TRIGGER trg_posts_ai_category_count
AFTER INSERT ON posts
FOR EACH ROW
UPDATE categories
SET post_count = post_count + 1
WHERE id = NEW.category_id;

CREATE TRIGGER trg_posts_ad_category_count
AFTER DELETE ON posts
FOR EACH ROW
UPDATE categories
SET post_count = post_count - 1
WHERE id = OLD.category_id;

CREATE TRIGGER trg_posts_au_category_count
AFTER UPDATE ON posts
FOR EACH ROW
UPDATE categories
SET post_count = post_count +
    CASE
      WHEN id = NEW.category_id AND NOT (OLD.category_id <=> NEW.category_id) THEN 1
      WHEN id = OLD.category_id AND NOT (OLD.category_id <=> NEW.category_id) THEN -1
      ELSE 0
    END
WHERE id IN (OLD.category_id, NEW.category_id);

-- Maintain tags.post_count based on post_tags changes
CREATE TRIGGER trg_post_tags_ai_tag_count
AFTER INSERT ON post_tags
FOR EACH ROW
UPDATE tags
SET post_count = post_count + 1
WHERE id = NEW.tag_id;

CREATE TRIGGER trg_post_tags_ad_tag_count
AFTER DELETE ON post_tags
FOR EACH ROW
UPDATE tags
SET post_count = post_count - 1
WHERE id = OLD.tag_id;

CREATE TRIGGER trg_post_tags_au_tag_count
AFTER UPDATE ON post_tags
FOR EACH ROW
UPDATE tags
SET post_count = post_count +
    CASE
      WHEN id = NEW.tag_id AND NOT (OLD.tag_id <=> NEW.tag_id) THEN 1
      WHEN id = OLD.tag_id AND NOT (OLD.tag_id <=> NEW.tag_id) THEN -1
      ELSE 0
    END
WHERE id IN (OLD.tag_id, NEW.tag_id);

-- =========================================
-- DB-level defaults (strong dependency) via triggers
-- =========================================

-- users.bio default: '该用户尚未填写个人简介。' when NULL or empty/whitespace
CREATE TRIGGER trg_users_bi_default_bio
BEFORE INSERT ON users
FOR EACH ROW
SET NEW.bio = CASE
    WHEN NEW.bio IS NULL OR CHAR_LENGTH(TRIM(NEW.bio)) = 0 THEN '该用户尚未填写个人简介。'
    ELSE NEW.bio
END;

CREATE TRIGGER trg_users_bu_default_bio
BEFORE UPDATE ON users
FOR EACH ROW
SET NEW.bio = CASE
    WHEN NEW.bio IS NULL OR CHAR_LENGTH(TRIM(NEW.bio)) = 0 THEN '该用户尚未填写个人简介。'
    ELSE NEW.bio
END;

-- notifications.meta default: {} when NULL
CREATE TRIGGER trg_notifications_bi_default_meta
BEFORE INSERT ON notifications
FOR EACH ROW
SET NEW.meta = COALESCE(NEW.meta, JSON_OBJECT());

CREATE TRIGGER trg_notifications_bu_default_meta
BEFORE UPDATE ON notifications
FOR EACH ROW
SET NEW.meta = COALESCE(NEW.meta, JSON_OBJECT());
