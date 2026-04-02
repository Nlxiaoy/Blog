package usecase

import (
	"context"
	"server/internal/usecase/input"
	"server/internal/usecase/output"
	"server/pkg/ssehub"
	"time"
)

// Auth 提供认证相关的用例入口。
//
// 通常由 usecase 层实现，并在 handler 层中被调用。
type Auth interface {
	// Admin 返回管理员认证相关操作集合。
	Admin() AdminAuth
	// User 返回普通用户认证相关操作集合。
	User() UserAuth
}

// AdminAuth 定义管理员账号的认证与资料相关操作。
// 同时包含 2FA（TOTP）开启/验证以及恢复码等流程。
type AdminAuth interface {
	// Login 校验用户名与密码，成功时返回管理员详情。
	Login(ctx context.Context, username, password string) (*output.AdminDetail, error)
	// GetAdminByID 根据管理员 ID 获取管理员详情。
	GetAdminByID(ctx context.Context, id int64) (*output.AdminDetail, error)
	// ChangePassword 修改管理员密码（一般需要校验旧密码）。
	ChangePassword(ctx context.Context, params input.ChangePassword) error

	// StartTwoFactorSetup 开始 2FA 设置流程，返回设置所需信息（如二维码/密钥等）。
	StartTwoFactorSetup(ctx context.Context, id int64) (*output.TwoFASetupStart, error)
	// VerifyTwoFactorSetup 校验待完成的 2FA 设置流程中的验证码。
	// 成功时通常会确认本次设置并返回验证结果。
	VerifyTwoFactorSetup(ctx context.Context, id int64, setupID string, code string) (*output.TwoFAVerifyResult, error)
	// ValidateTOTP 使用管理员当前 2FA 密钥校验 TOTP 验证码。
	ValidateTOTP(ctx context.Context, id int64, code string) (bool, error)
	// SetTwoFactorSecret 保存/启用管理员的 2FA 密钥。
	SetTwoFactorSecret(ctx context.Context, id int64, secret string) error
	// ClearTwoFactorSecret 清除已保存的 2FA 密钥，用于关闭 2FA。
	ClearTwoFactorSecret(ctx context.Context, id int64) error

	// VerifyAndUseRecoveryCode 校验恢复码，并在成功使用后将其置为失效。
	VerifyAndUseRecoveryCode(ctx context.Context, id int64, code string) (bool, error)
	// InvalidateRecoveryCodes 使管理员所有恢复码失效。
	InvalidateRecoveryCodes(ctx context.Context, id int64) error
	// ResetRecoveryCodes 生成并保存一组新的恢复码，返回明文恢复码列表。
	ResetRecoveryCodes(ctx context.Context, id int64) ([]string, error)

	// GetProfile 获取管理员个人资料。
	GetProfile(ctx context.Context, id int64) (*output.AdminProfile, error)
	// UpdateProfile 更新管理员可修改的资料字段。
	UpdateProfile(ctx context.Context, params input.UpdateAdminProfile) error
}

// UserAuth 定义普通用户的认证与会话（token）相关操作。
type UserAuth interface {
	// Register 注册新用户账号，成功时返回用户详情。
	Register(ctx context.Context, username, email, password string) (*output.UserDetail, error)
	// Login 校验邮箱与密码，成功时返回 access/refresh token 对。
	Login(ctx context.Context, email, password string) (*output.TokenPair, error)
	// Refresh 校验 refresh token 并签发新的 token 对。
	Refresh(ctx context.Context, refreshToken string) (*output.TokenPair, error)
	// ChangePassword 修改用户密码（一般需要校验旧密码）。
	ChangePassword(ctx context.Context, id int64, oldPassword, newPassword string) error
	// ResetPasswordByEmail 根据邮箱重置密码（通常发生在完成邮箱验证之后）。
	ResetPasswordByEmail(ctx context.Context, email, newPassword string) error
	// RevokeUserRefreshToken 吊销用户 refresh token（通常用于强制重新登录）。
	RevokeUserRefreshToken(ctx context.Context, userID int64) error
	// ValidateSession 校验指定用户的 refresh token 是否仍然有效。
	ValidateSession(ctx context.Context, userID int64, refreshToken string) error
}

type Captcha interface {
	Generate(ctx context.Context) (string, string, error)
	Verify(ctx context.Context, id, answer string) (bool, error)
}

type Email interface {
	SendCode(ctx context.Context, to string) error
	VerifyCode(ctx context.Context, to, code string) (bool, error)
}

type File interface {
	// Object Storage (MinIO)
	GenerateUploadURL(ctx context.Context, key string, expires time.Duration, contentType string) (string, error)
	GetFileURL(ctx context.Context, key string, expires time.Duration) (string, error)
	DeleteObject(ctx context.Context, key string) error

	// File Metadata (DB)
	SaveMeta(ctx context.Context, params input.SaveFileMeta) (int64, error)
	GetMeta(ctx context.Context, objectKey string) (*output.FileDetail, error)
	ListFiles(ctx context.Context, params input.ListFiles) (*output.ListResult[output.FileDetail], error)
	ListMetaByResource(ctx context.Context, usage string, resourceID int64) ([]*output.FileDetail, error)
	BindResource(ctx context.Context, objectKey string, resourceID int64) error
	ClearResourceByUsage(ctx context.Context, usage string, resourceID int64) error
	DeleteWithMeta(ctx context.Context, objectKey string) error
}
type User interface {
	// Admin
	ListUsers(ctx context.Context, params input.ListUsers) (*output.ListResult[output.UserDetail], error)
	UpdateStatus(ctx context.Context, id int64, status string) error

	// Public
	GetUserByID(ctx context.Context, id int64) (*output.UserDetail, error)
	UpdateProfile(ctx context.Context, id int64, params input.UpdateProfile) error
}

type Content interface {
	// Admin
	ListPosts(ctx context.Context, params input.ListPosts) (*output.ListResult[output.PostSummary], error)
	GetPostByID(ctx context.Context, id int64) (*output.PostDetail, error)
	CreatePost(ctx context.Context, params input.CreatePost) (int64, error)
	UpdatePost(ctx context.Context, params input.UpdatePost) error
	DeletePost(ctx context.Context, id int64) error
	ListCategories(ctx context.Context, params input.ListCategories) (*output.ListResult[output.CategoryDetail], error)
	CreateCategory(ctx context.Context, params input.CreateCategory) (int64, error)
	UpdateCategory(ctx context.Context, params input.UpdateCategory) error
	DeleteCategory(ctx context.Context, id int64) error
	ListTags(ctx context.Context, params input.ListTags) (*output.ListResult[output.TagDetail], error)
	CreateTag(ctx context.Context, params input.CreateTag) (int64, error)
	UpdateTag(ctx context.Context, params input.UpdateTag) error
	DeleteTag(ctx context.Context, id int64) error
	GenerateSlug(ctx context.Context, title string) (string, error)

	// Public
	ListPublicPosts(ctx context.Context, params input.ListPublicPosts, userID *int64) (*output.ListResult[output.PostSummary], error)
	GetPublicPostBySlug(ctx context.Context, slug string, userID *int64) (*output.PostDetail, error)
	GetAllPublicCategories(ctx context.Context) (*output.AllResult[output.CategoryDetail], error)
	GetAllPublicTags(ctx context.Context) (*output.AllResult[output.TagDetail], error)
	ToggleLikeOnPost(ctx context.Context, postID int64, userID int64) (bool, int32, error)
	RemoveLikeOnPost(ctx context.Context, postID int64, userID int64) (bool, int32, error)
	RecordView(ctx context.Context, postID int64, ip, userAgent, referer string)
}

type Comment interface {
	// Admin
	ListComments(ctx context.Context, params input.ListComments) (*output.ListResult[output.CommentDetail], error)
	UpdateCommentStatus(ctx context.Context, id int64, status string) error
	DeleteComment(ctx context.Context, id int64) error

	// Public
	GetAllPublicCommentsByPostID(ctx context.Context, postID int64, userID *int64) (*output.AllResult[output.CommentBasic], error)
	SubmitComment(ctx context.Context, params input.SubmitComment) error
	ToggleLikeOnComment(ctx context.Context, commentID int64, userID int64) (bool, int32, error)
	RemoveLikeOnComment(ctx context.Context, commentID int64, userID int64) (bool, int32, error)
	DeleteOwnComment(ctx context.Context, commentID int64, userID int64) error
}

type Feedback interface {
	// Admin
	ListFeedbacks(ctx context.Context, params input.ListFeedbacks) (*output.ListResult[output.FeedbackDetail], error)
	GetFeedbackByID(ctx context.Context, id int64) (*output.FeedbackDetail, error)
	UpdateFeedback(ctx context.Context, params input.UpdateFeedback) error
	DeleteFeedback(ctx context.Context, id int64) error

	// Public
	SubmitFeedback(ctx context.Context, params input.SubmitFeedback) error
}

type Link interface {
	// Admin
	ListLinks(ctx context.Context, params input.ListLinks) (*output.ListResult[output.LinkDetail], error)
	CreateLink(ctx context.Context, params input.CreateLink) (int64, error)
	UpdateLink(ctx context.Context, params input.UpdateLink) error
	DeleteLink(ctx context.Context, id int64) error

	// Public
	GetAllPublicLinks(ctx context.Context) (*output.AllResult[output.LinkDetail], error)
}

type Setting interface {
	// Admin
	GetAllSiteSettings(ctx context.Context) (*output.AllResult[output.SiteSettingDetail], error)
	GetSiteSettingByKey(ctx context.Context, key string) (*output.SiteSettingDetail, error)
	UpsertSiteSetting(ctx context.Context, params input.UpsertSiteSetting) error
}

type Notification interface {
	// V1
	ListMyNotifications(ctx context.Context, params input.ListNotifications) (*output.ListResult[output.NotificationDetail], error)
	GetUnreadCount(ctx context.Context, userID int64) (int64, error)
	MarkRead(ctx context.Context, id, userID int64) error
	MarkAllRead(ctx context.Context, userID int64) error
	DeleteNotification(ctx context.Context, id, userID int64) error

	// Admin
	SendAdminMessage(ctx context.Context, params input.SendAdminNotification) error

	// SSE
	Subscribe(userID int64) chan ssehub.Event
	Unsubscribe(userID int64, ch chan ssehub.Event)
}
