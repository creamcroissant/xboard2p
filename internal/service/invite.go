package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/repository"
	"github.com/google/uuid"
)

// InviteService 负责邀请相关交互。
type InviteService interface {
	TrackVisit(ctx context.Context, code string) error
	GenerateBatch(ctx context.Context, count int, limit int64, expireAt int64, userID int64) error
	FindByCode(ctx context.Context, code string) (*repository.InviteCode, error)
	Fetch(ctx context.Context, limit, offset int) ([]*repository.InviteCode, int64, error)
	Validate(ctx context.Context, code string) (*repository.InviteCode, error)
	Consume(ctx context.Context, code string) error
}

type inviteService struct {
	repo     repository.InviteCodeRepository
	userRepo repository.UserRepository
}

// NewInviteService 构造基于仓储的 InviteService。
func NewInviteService(repo repository.InviteCodeRepository, userRepo repository.UserRepository) InviteService {
	return &inviteService{
		repo:     repo,
		userRepo: userRepo,
	}
}

func (s *inviteService) TrackVisit(ctx context.Context, code string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return nil
	}
	return s.repo.IncrementPV(ctx, trimmed)
}

func (s *inviteService) GenerateBatch(ctx context.Context, count int, limit int64, expireAt int64, userID int64) error {
	if count <= 0 {
		return nil
	}
	if limit <= 0 {
		limit = 1
	}

	if userID > 0 {
		user, err := s.userRepo.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		if user.InviteLimit > -1 {
			existingCount, err := s.repo.CountByUser(ctx, userID)
			if err != nil {
				return err
			}
			if existingCount+int64(count) > user.InviteLimit {
				return fmt.Errorf("invite limit exceeded / 邀请次数超出限制")
			}
		}
	}

	codes := make([]*repository.InviteCode, 0, count)
	for i := 0; i < count; i++ {
		codes = append(codes, &repository.InviteCode{
			UserID:    userID,
			Code:      generateInviteCode(),
			Status:    0,
			PV:        0,
			Limit:     limit,
			ExpireAt:  expireAt,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		})
	}
	return s.repo.CreateBatch(ctx, codes)
}

func (s *inviteService) FindByCode(ctx context.Context, code string) (*repository.InviteCode, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("invite service not configured / 邀请服务未配置")
	}
	return s.repo.FindByCode(ctx, code)
}

func (s *inviteService) Fetch(ctx context.Context, limit, offset int) ([]*repository.InviteCode, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, fmt.Errorf("invite service not configured / 邀请服务未配置")
	}
	codes, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountAll(ctx)
	if err != nil {
		return nil, 0, err
	}
	return codes, total, nil
}

func (s *inviteService) Validate(ctx context.Context, code string) (*repository.InviteCode, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("invite service not configured / 邀请服务未配置")
	}
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return nil, ErrInvalidInviteCode
	}
	invite, err := s.repo.FindByCode(ctx, trimmed)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidInviteCode
		}
		return nil, err
	}
	if invite.Status != 0 {
		return nil, ErrInvalidInviteCode
	}
	if invite.ExpireAt > 0 && invite.ExpireAt < time.Now().Unix() {
		return nil, ErrInvalidInviteCode
	}
	return invite, nil
}

func (s *inviteService) Consume(ctx context.Context, code string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("invite service not configured / 邀请服务未配置")
	}
	invite, err := s.Validate(ctx, code)
	if err != nil {
		return err
	}
	return s.repo.MarkUsed(ctx, invite.ID)
}

func generateInviteCode() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
}