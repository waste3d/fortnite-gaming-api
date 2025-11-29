package grpc_server

import (
	"context"
	"fmt"
	"time"

	"github.com/waste3d/gameplatform-api/services/user-service/internal/domain"
	"github.com/waste3d/gameplatform-api/services/user-service/internal/infrastructure/repository"

	userpb "github.com/waste3d/gameplatform-api/services/user-service/pkg/userpb/proto/user"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServer struct {
	userpb.UnimplementedUserServiceServer
	repo *repository.ProfileRepository
}

func NewUserServer(repo *repository.ProfileRepository) *UserServer {
	return &UserServer{repo: repo}
}

func (s *UserServer) CreateProfile(ctx context.Context, req *userpb.CreateProfileRequest) (*userpb.CreateProfileResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	profile := &domain.Profile{
		ID:       uid,
		Email:    req.Email,
		Username: req.Username,
	}

	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create profile: %v", err)
	}

	return &userpb.CreateProfileResponse{ProfileId: profile.ID.String()}, nil
}

func (s *UserServer) SyncEmail(ctx context.Context, req *userpb.SyncEmailRequest) (*userpb.SyncEmailResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	err := s.repo.UpdateEmail(ctx, uid, req.NewEmail)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to sync email")
	}
	return &userpb.SyncEmailResponse{Success: true}, nil
}

func (s *UserServer) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	// Валидация
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username cannot be empty")
	}

	// Вызываем репозиторий
	err = s.repo.UpdateUsername(ctx, uid, req.Username)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	return &userpb.UpdateProfileResponse{Success: true}, nil
}

func (s *UserServer) SetAvatar(ctx context.Context, req *userpb.SetAvatarRequest) (*userpb.SetAvatarResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	// Проверяем диапазон (у нас 10 пресетов)
	if req.AvatarId < 1 || req.AvatarId > 10 {
		return nil, status.Error(codes.InvalidArgument, "avatar_id must be between 1 and 10")
	}

	// Вызываем репозиторий
	err = s.repo.UpdateAvatar(ctx, uid, int(req.AvatarId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update avatar")
	}

	return &userpb.SetAvatarResponse{
		Success:  true,
		AvatarId: req.AvatarId,
	}, nil
}

func (s *UserServer) UpdateProgress(ctx context.Context, req *userpb.UpdateProgressRequest) (*userpb.UpdateProgressResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	newStatus, err := s.repo.UpdateProgress(ctx, uid, req.CourseId, req.ProgressPercent)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update progress: %v", err)
	}

	return &userpb.UpdateProgressResponse{Success: true, Status: newStatus}, nil
}

func (s *UserServer) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	p, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	usedCount, _ := s.repo.CountUserCourses(ctx, uid)

	// Получаем списки курсов (как и раньше)
	courses, _ := s.repo.GetUserCourses(ctx, uid)
	var active, completed []*userpb.CoursePreview
	for _, c := range courses {
		pb := &userpb.CoursePreview{Id: c.CourseID, Title: c.Title, ProgressPercent: c.ProgressPercent, CoverUrl: c.CoverURL}
		if c.Status == "completed" {
			completed = append(completed, pb)
		} else {
			active = append(active, pb)
		}
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastActivity := time.Date(p.LastStreakAt.Year(), p.LastStreakAt.Month(), p.LastStreakAt.Day(), 0, 0, 0, 0, p.LastStreakAt.Location())

	daysDiff := int(today.Sub(lastActivity).Hours() / 24)

	displayStreak := int32(p.Streak)
	isActiveToday := false

	if daysDiff == 0 {
		isActiveToday = true
	} else if daysDiff > 1 {
		// Если пользователь зашел, но еще не прошел урок, а последний раз был давно -> показываем 0
		displayStreak = 0
	}

	var unlockedIDs []int32
	// Базовые аватарки (1-5) доступны всем
	for i := 1; i <= 5; i++ {
		unlockedIDs = append(unlockedIDs, int32(i))
	}
	for _, ua := range p.UnlockedAvatars {
		unlockedIDs = append(unlockedIDs, int32(ua.AvatarID))
	}

	rank, err := s.repo.GetUserRank(ctx, uid)
	if err != nil {
		fmt.Printf("Error getting user rank: %v\n", err)
	}

	return &userpb.GetProfileResponse{
		Id:       p.ID.String(),
		Email:    p.Email,
		Username: p.Username,
		AvatarId: int32(p.AvatarID),

		// Новые поля
		SubscriptionStatus: p.SubscriptionStatus,
		CourseLimit:        int32(p.CourseLimit),
		CoursesUsed:        int32(usedCount),
		DeviceLimit:        int32(p.DeviceLimit),
		ExpiresAt:          p.SubscriptionEndsAt.Unix(),
		TgAccess:           p.HasTgAccess,

		ActiveCourses:    active,
		CompletedCourses: completed,

		Streak:              displayStreak,
		IsStreakActiveToday: isActiveToday,
		Balance:             int32(p.Balance),
		UnlockedAvatarIds:   unlockedIDs,
		Rank:                int32(rank),
	}, nil
}

func (s *UserServer) CompleteLesson(ctx context.Context, req *userpb.CompleteLessonRequest) (*userpb.CompleteLessonResponse, error) {
	uid, _ := uuid.Parse(req.UserId)

	// 1. Сохраняем урок как пройденный.
	// Функция AddCompletedLesson возвращает (bool created, error).
	// created == true только если этот урок еще не был пройден.
	created, err := s.repo.AddCompletedLesson(ctx, &domain.CompletedLesson{
		UserID:   uid,
		CourseID: req.CourseId,
		LessonID: req.LessonId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to save lesson")
	}

	// 2. Начисляем награды ТОЛЬКО за новые уроки (защита от фарма)
	if created {
		// Обновляем стрик

		streakUpdated, err := s.repo.CheckAndIncrementStreak(ctx, uid)
		if err != nil {
			fmt.Printf("Error updating streak: %v\n", err)
		}

		// Начисляем +10 снежинок ТОЛЬКО если стрик был обновлен (первый урок за день)
		if streakUpdated {
			if newBalance, err := s.repo.ChangeBalance(ctx, uid, 10); err != nil {
				fmt.Printf("Error changing balance: %v\n", err)
			} else {
				fmt.Printf("User %s earned daily bonus. New balance: %d\n", uid, newBalance)
			}
		}
	}

	// 3. Считаем процент прохождения
	completedCount, _ := s.repo.CountCompletedLessons(ctx, uid, req.CourseId)
	var percent int32
	if req.TotalLessons > 0 {
		percent = int32((float64(completedCount) / float64(req.TotalLessons)) * 100)
	}
	if percent > 100 {
		percent = 100
	}

	// 4. Проверка на завершение курса (Награда 50 снежинок)
	finalStatus := "active"

	if percent >= 100 {
		// Получаем ТЕКУЩИЙ статус из БД, чтобы понять, завершаем мы его впервые или нет
		currentStatus, _ := s.repo.GetUserCourseStatus(ctx, uid, req.CourseId)

		// Если курс еще не был помечен как "completed", значит это первое завершение
		if currentStatus != "completed" {
			// Выдаем большую награду
			s.repo.ChangeBalance(ctx, uid, 50)
			// Увеличиваем счетчик пройденных курсов для лидерборда
			s.repo.IncrementCompletedCount(ctx, uid)
		}
		finalStatus = "completed"
	}

	// 5. Обновляем статус и процент в БД
	_, err = s.repo.UpdateProgress(ctx, uid, req.CourseId, percent)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update progress")
	}

	return &userpb.CompleteLessonResponse{
		Success:    true,
		NewPercent: percent,
		Status:     finalStatus,
	}, nil
}

func (s *UserServer) GetCompletedLessons(ctx context.Context, req *userpb.GetCompletedLessonsRequest) (*userpb.GetCompletedLessonsResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	ids, err := s.repo.GetCompletedLessonIDs(ctx, uid, req.CourseId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get lessons")
	}
	return &userpb.GetCompletedLessonsResponse{LessonIds: ids}, nil
}

func (s *UserServer) SetSubscription(ctx context.Context, req *userpb.SetSubscriptionRequest) (*userpb.SetSubscriptionResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	updates := map[string]interface{}{
		"subscription_status":  req.PlanName,
		"course_limit":         int(req.CourseLimit),
		"device_limit":         int(req.DeviceLimit),
		"has_tg_access":        req.TgAccess,
		"subscription_ends_at": time.Unix(req.ExpiresAt, 0),
	}

	if err := s.repo.UpdateSubscription(ctx, uid, updates); err != nil {
		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	return &userpb.SetSubscriptionResponse{Success: true}, nil
}

func (s *UserServer) StartCourse(ctx context.Context, req *userpb.StartCourseRequest) (*userpb.StartCourseResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	profile, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}

	// Логика доступа
	if profile.SubscriptionStatus != "admin" {

		// А. Проверка срока действия
		if profile.SubscriptionStatus != "Обычный" && time.Now().After(profile.SubscriptionEndsAt) {
			// Можно здесь автоматически откатывать на "Обычный", но пока вернем ошибку
			return nil, status.Error(codes.PermissionDenied, "Ваша подписка истекла")
		}

		// Б. Если курс уже добавлен - ОК
		has, _ := s.repo.UserHasCourse(ctx, uid, req.CourseId)
		if has {
			return &userpb.StartCourseResponse{Success: true}, nil
		}

		// В. Проверка лимитов (Слоты)
		// Если CourseLimit == -1, это безлимит
		if profile.CourseLimit != -1 {
			used, _ := s.repo.CountUserCourses(ctx, uid)
			if int(used) >= profile.CourseLimit {
				return nil, status.Error(codes.ResourceExhausted, "Лимит курсов по вашему тарифу исчерпан")
			}
		}
	}

	// Все проверки пройдены, добавляем курс
	uc := &domain.UserCourse{
		UserID:   uid,
		CourseID: req.CourseId,
		Title:    req.Title,
		CoverURL: req.CoverUrl,
		Status:   "active",
	}
	_ = s.repo.StartCourse(ctx, uc)

	return &userpb.StartCourseResponse{Success: true}, nil
}

func (s *UserServer) AddCourseLimit(ctx context.Context, req *userpb.AddCourseLimitRequest) (*userpb.AddCourseLimitResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	err := s.repo.AddCourseSlots(ctx, uid, int(req.Count))
	return &userpb.AddCourseLimitResponse{Success: err == nil}, err
}

func (s *UserServer) ChangeBalance(ctx context.Context, req *userpb.ChangeBalanceRequest) (*userpb.ChangeBalanceResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	newBal, err := s.repo.ChangeBalance(ctx, uid, int(req.Amount))
	return &userpb.ChangeBalanceResponse{Success: err == nil, NewBalance: int32(newBal)}, err
}

func (s *UserServer) UnlockAvatar(ctx context.Context, req *userpb.UnlockAvatarRequest) (*userpb.UnlockAvatarResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	exists, err := s.repo.AddUnlockedAvatar(ctx, uid, int(req.AvatarId))
	return &userpb.UnlockAvatarResponse{Success: err == nil, AlreadyOwned: exists}, err
}

func (s *UserServer) GetLeaderboard(ctx context.Context, req *userpb.GetLeaderboardRequest) (*userpb.GetLeaderboardResponse, error) {
	users, err := s.repo.GetLeaderboard(ctx, int(req.Limit))
	if err != nil {
		return nil, err
	}

	var entries []*userpb.LeaderboardEntry
	for _, u := range users {
		entries = append(entries, &userpb.LeaderboardEntry{
			UserId:         u.ID.String(),
			Username:       u.Username,
			AvatarId:       int32(u.AvatarID),
			Streak:         int32(u.Streak),
			CompletedCount: int32(u.CompletedCount),
			Balance:        int32(u.Balance),
		})
	}
	return &userpb.GetLeaderboardResponse{Entries: entries}, nil
}
