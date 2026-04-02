//go:build wireinject
// +build wireinject

package app

import (
	"context"
	"server/config"
	"server/internal/repo"
	"server/internal/repo/cache"
	"server/internal/repo/messaging"

	httpctrl "server/internal/controller/http"
	reponotif "server/internal/repo/notification"
	"server/internal/repo/persistence"
	"server/internal/repo/storage"
	"server/internal/repo/viewbuffer"
	"server/internal/repo/webapi"
	"server/internal/usecase"
	"server/internal/usecase/auth"
	authadmin "server/internal/usecase/auth/admin"
	authuser "server/internal/usecase/auth/user"
	"server/internal/usecase/captcha"
	"server/internal/usecase/comment"
	"server/internal/usecase/content"
	"server/internal/usecase/email"
	"server/internal/usecase/feedback"
	"server/internal/usecase/file"
	"server/internal/usecase/link"
	"server/internal/usecase/notification"
	"server/internal/usecase/setting"
	"server/internal/usecase/user"
	"server/pkg/httpserver"
	"server/pkg/logger"
	"server/pkg/mysql"
	"server/pkg/redis"
	"server/pkg/ssehub"
	"time"

	minioSDK "github.com/minio/minio-go/v7"

	"github.com/google/wire"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// App 应用容器。
type App struct {
	Info AppInfo

	Logger     logger.Interface
	MySQL      *mysql.Mysql
	Redis      *redis.Redis
	HTTPServer *httpserver.Server
}

// AppInfo 应用信息。
type AppInfo struct {
	Name    string
	Version string
}

// NewApp 创建 App。
func NewApp(info AppInfo, l logger.Interface, db *mysql.Mysql, r *redis.Redis, srv *httpserver.Server) *App {
	return &App{
		Info:       info,
		Logger:     l,
		MySQL:      db,
		Redis:      r,
		HTTPServer: srv,
	}
}

// NewAppInfo 创建 AppInfo。
func NewAppInfo(cfg *config.Config) AppInfo {
	return AppInfo{Name: cfg.App.Name, Version: cfg.App.Version}
}

// NewLogger 创建 Logger。
func NewLogger(cfg *config.Config) logger.Interface {
	return logger.New(cfg.Log.Level)
}

// NewMySQL 创建 MySQL 连接并返回 cleanup。
func NewMySQL(cfg *config.Config) (*mysql.Mysql, func(), error) {
	db, err := mysql.New(
		cfg.Mysql.Host,
		cfg.Mysql.Port,
		cfg.Mysql.User,
		cfg.Mysql.Password,
		cfg.Mysql.DBName,
		cfg.Mysql.Config,
	)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { db.Close() }
	return db, cleanup, nil
}

// NewRedis 创建 Redis 连接并返回 cleanup。
func NewRedis(cfg *config.Config) (*redis.Redis, func(), error) {
	rdb, err := redis.New(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { rdb.Close() }
	return rdb, cleanup, nil
}

// NewMinioClient 创建 MinIO Client 并确保默认 bucket 存在。
func NewMinioClient(cfg *config.Config) (*minio.Client, error) {
	cli, err := minio.New(
		cfg.MinIO.Endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
			Secure: cfg.MinIO.UseSSL,
			Region: cfg.MinIO.Region,
		},
	)
	if err != nil {
		return nil, err
	}
	// 如果配置了桶，自动创建（不存在则创建，存在则跳过）
	if cfg.MinIO.Bucket != "" {
		// 检查桶是否存在
		_, err = cli.BucketExists(context.Background(), cfg.MinIO.Bucket)
		if err != nil {
			return nil, err
		}
	}
	return cli, nil
}

// Persistence Repo（MySQL）。

func NewAdminRepo(db *mysql.Mysql) repo.AdminRepo {
	return persistence.NewAdminRepo(db.DB)
}

func NewUserRepo(db *mysql.Mysql) repo.UserRepo {
	return persistence.NewUserRepo(db.DB)
}

func NewPostRepo(db *mysql.Mysql) repo.PostRepo {
	return persistence.NewPostRepo(db.DB)
}

func NewTagRepo(db *mysql.Mysql) repo.TagRepo {
	return persistence.NewTagRepo(db.DB)
}

func NewCategoryRepo(db *mysql.Mysql) repo.CategoryRepo {
	return persistence.NewCategoryRepo(db.DB)
}

func NewCommentRepo(db *mysql.Mysql) repo.CommentRepo {
	return persistence.NewCommentRepo(db.DB)
}

func NewPostLikeRepo(db *mysql.Mysql) repo.PostLikeRepo {
	return persistence.NewPostLikeRepo(db.DB)
}

func NewCommentLikeRepo(db *mysql.Mysql) repo.CommentLikeRepo {
	return persistence.NewCommentLikeRepo(db.DB)
}

func NewFeedbackRepo(db *mysql.Mysql) repo.FeedbackRepo {
	return persistence.NewFeedbackRepo(db.DB)
}

func NewLinkRepo(db *mysql.Mysql) repo.LinkRepo {
	return persistence.NewLinkRepo(db.DB)
}

func NewSiteSettingRepo(db *mysql.Mysql) repo.SiteSettingRepo {
	return persistence.NewSiteSettingRepo(db.DB)
}

func NewFileRepo(db *mysql.Mysql) repo.FileRepo {
	return persistence.NewFileRepo(db.DB)
}

func NewNotificationRepo(db *mysql.Mysql) repo.NotificationRepo {
	return persistence.NewNotificationRepo(db.DB)
}

func NewRefreshTokenBlacklistRepo(db *mysql.Mysql) repo.RefreshTokenBlacklistRepo {
	return persistence.NewRefreshTokenBlacklistPostgres(db.DB)
}

// ViewBuffer Repo（浏览量缓冲）。
func NewPostViewRepo(db *mysql.Mysql, l logger.Interface) (repo.PostViewRepo, func()) {
	return viewbuffer.New(db.DB, l)
}

// Cache Repo（Redis）。
func NewCaptchaStore(r *redis.Redis) repo.CaptchaStore {
	return cache.NewCaptchaRedisStore(r, 5*time.Minute)
}

func NewEmailCodeStore(r *redis.Redis) repo.EmailCodeStore {
	return cache.NewEmailCodeRedisStore(r, 10*time.Minute)
}

func NewRefreshTokenStore(r *redis.Redis) repo.RefreshTokenStore {
	return cache.NewRefreshTokenRedisStore(r)
}

func NewAdminTwoFASetupStore(r *redis.Redis) repo.AdminTwoFASetupStore {
	return cache.NewAdminTwoFARedisStore(r)
}

// Storage Repo（MinIO）。

func NewObjectStore(cli *minioSDK.Client) repo.ObjectStore {
	return storage.NewMinioStore(cli)
}

// Messaging Repo（SMTP）。

func NewEmailSender(cfg *config.Config) repo.EmailSender {
	return messaging.NewSMTPEmailSender(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
		cfg.SMTP.From,
	)
}

// WebAPI Repo（外部 API）。

func NewTranslationWebAPI() repo.TranslationWebAPI {
	return webapi.NewTranslationWebAPI()
}

func NewLLMWebAPI(cfg *config.Config) repo.LLMWebAPI {
	return webapi.NewLLMWebAPI(cfg.OpenAI.APIKey, cfg.OpenAI.BaseURL, cfg.OpenAI.Model)
}

// Auth UseCase。
func NewTokenSigner(cfg *config.Config) (authuser.TokenSigner, error) {
	return authuser.NewTokenSigner(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
		cfg.JWT.Issuer,
	)
}

func NewAdminAuthUseCase(cfg *config.Config, adminRepo repo.AdminRepo, twoFASetupStore repo.AdminTwoFASetupStore) usecase.AdminAuth {
	totpCfg := authadmin.TOTPConfig{QRWidth: cfg.TwoFA.QRWidth, QRHeight: cfg.TwoFA.QRHeight}
	enc := authadmin.NewEncryptorFromSecret(cfg.TwoFA.EncryptionKey)
	return authadmin.New(adminRepo, twoFASetupStore, cfg.App.Name, authadmin.NewTOTPProviderWithConfig(totpCfg), enc)
}

func NewUserAuthUseCase(userRepo repo.UserRepo, signer authuser.TokenSigner, refreshStore repo.RefreshTokenStore, refreshBlacklist repo.RefreshTokenBlacklistRepo) usecase.UserAuth {
	return authuser.New(userRepo, signer, refreshStore, refreshBlacklist)
}

func NewAuthUseCase(adminAuth usecase.AdminAuth, userAuth usecase.UserAuth) usecase.Auth {
	return auth.New(adminAuth, userAuth)
}

// Captcha UseCase。

func NewCaptchaGenerator(cfg *config.Config) captcha.Generator {
	return captcha.NewBase64Generator(captcha.Config{
		Height:   cfg.Captcha.Height,
		Width:    cfg.Captcha.Width,
		Length:   cfg.Captcha.Length,
		MaxSkew:  cfg.Captcha.MaxSkew,
		DotCount: cfg.Captcha.DotCount,
	})
}

func NewCaptchaUseCase(gen captcha.Generator, store repo.CaptchaStore) usecase.Captcha {
	return captcha.New(store, gen)
}

// Email UseCase。

func NewEmailUseCase(sender repo.EmailSender, codeStore repo.EmailCodeStore) usecase.Email {
	return email.New(sender, codeStore)
}

// File UseCase。

func NewFileUseCase(cfg *config.Config, objectStore repo.ObjectStore, fileRepo repo.FileRepo) usecase.File {
	return file.New(objectStore, fileRepo, cfg.MinIO.Bucket)
}

// User UseCase。

func NewUserUseCase(userRepo repo.UserRepo) usecase.User {
	return user.New(userRepo)
}

// Content UseCase。

func NewContentUseCase(translationAPI repo.TranslationWebAPI, llmAPI repo.LLMWebAPI, adminRepo repo.AdminRepo, postRepo repo.PostRepo, tagRepo repo.TagRepo, categoryRepo repo.CategoryRepo, postLikeRepo repo.PostLikeRepo, fileRepo repo.FileRepo, postViewRepo repo.PostViewRepo) usecase.Content {
	return content.New(translationAPI, llmAPI, adminRepo, postRepo, tagRepo, categoryRepo, postLikeRepo, fileRepo, postViewRepo, content.NewCalculator())
}

// Comment UseCase。

func NewCommentUseCase(commentRepo repo.CommentRepo, commentLikeRepo repo.CommentLikeRepo, userRepo repo.UserRepo, postRepo repo.PostRepo, notifier repo.Notifier) usecase.Comment {
	return comment.New(commentRepo, commentLikeRepo, userRepo, postRepo, notifier)
}

// Feedback UseCase。

func NewFeedbackUseCase(feedbackRepo repo.FeedbackRepo) usecase.Feedback {
	return feedback.New(feedbackRepo)
}

// Link UseCase。

func NewLinkUseCase(linkRepo repo.LinkRepo) usecase.Link {
	return link.New(linkRepo)
}

// Setting UseCase。

func NewSettingUseCase(settingRepo repo.SiteSettingRepo) usecase.Setting {
	return setting.New(settingRepo)
}

// SSEHub SSE Hub。

func NewSSEHub() *ssehub.Hub {
	return ssehub.New()
}

// Notifier 通知推送实现。

func NewNotifier(notificationRepo repo.NotificationRepo, hub *ssehub.Hub) repo.Notifier {
	return reponotif.NewNotifier(notificationRepo, hub)
}

// Notification UseCase。

func NewNotificationUseCase(notificationRepo repo.NotificationRepo, notifier repo.Notifier, hub *ssehub.Hub) usecase.Notification {
	return notification.New(notificationRepo, notifier, hub)
}

// HTTP Server。
func SetupHTTPServer(
	cfg *config.Config, l logger.Interface,
	auth usecase.Auth, captchaUC usecase.Captcha, emailUC usecase.Email, signer authuser.TokenSigner,
	fileUC usecase.File, userUC usecase.User, contentUC usecase.Content, commentUC usecase.Comment,
	feedbackUC usecase.Feedback, linkUC usecase.Link, settingUC usecase.Setting,
	notificationUC usecase.Notification,
) *httpserver.Server {
	srv := httpserver.New(l, httpserver.WithPort(cfg.HTTP.Port), httpserver.WithPrefork(cfg.HTTP.UsePreforkMode))
	httpctrl.NewRouter(srv.App, cfg, l, auth, captchaUC, emailUC, signer, fileUC, userUC, contentUC, commentUC, feedbackUC, linkUC, settingUC, notificationUC)
	return srv
}

// ProviderSet Wire ProviderSet。
var ProviderSet = wire.NewSet(
	// App 应用容器。
	NewAppInfo,
	NewLogger,
	NewApp,
	// Infrastructure 基础设施。
	NewMySQL,
	NewRedis,
	NewMinioClient,
	// RepoPersistence Postgres Repo。
	NewAdminRepo,
	NewUserRepo,
	NewPostRepo,
	NewTagRepo,
	NewCategoryRepo,
	NewCommentRepo,
	NewPostLikeRepo,
	NewCommentLikeRepo,
	NewFeedbackRepo,
	NewLinkRepo,
	NewSiteSettingRepo,
	NewFileRepo,
	NewNotificationRepo,
	NewRefreshTokenBlacklistRepo,
	// RepoViewBuffer 浏览量缓冲。
	NewPostViewRepo,
	// RepoCache Redis Repo。
	NewCaptchaStore,
	NewEmailCodeStore,
	NewRefreshTokenStore,
	NewAdminTwoFASetupStore,
	// RepoStorage MinIO Repo。
	NewObjectStore,
	// RepoMessaging SMTP Repo。
	NewEmailSender,
	// RepoWebAPI 外部 API。
	NewTranslationWebAPI,
	NewLLMWebAPI,
	// UseCaseAuth 认证用例。
	NewTokenSigner,
	NewAdminAuthUseCase,
	NewUserAuthUseCase,
	NewAuthUseCase,
	// UseCaseCaptcha 验证码用例。
	NewCaptchaGenerator,
	NewCaptchaUseCase,
	// UseCaseEmail 邮件用例。
	NewEmailUseCase,
	// UseCaseFile 文件用例。
	NewFileUseCase,
	// UseCaseUser 用户用例。
	NewUserUseCase,
	// UseCaseContent 内容用例。
	NewContentUseCase,
	// UseCaseComment 评论用例。
	NewCommentUseCase,
	// UseCaseFeedback 反馈用例。
	NewFeedbackUseCase,
	// UseCaseLink 友链用例。
	NewLinkUseCase,
	// UseCaseSetting 设置用例。
	NewSettingUseCase,
	// Pkg 基础包。
	NewSSEHub,
	// RepoNotifier 通知推送。
	NewNotifier,
	// UseCaseNotification 通知用例。
	NewNotificationUseCase,
	// HTTP Server。
	SetupHTTPServer,
)

// InitializeApp 初始化 App 并返回 cleanup。
func InitializeApp(cfg *config.Config) (*App, func(), error) {
	wire.Build(ProviderSet)
	return nil, nil, nil
}
