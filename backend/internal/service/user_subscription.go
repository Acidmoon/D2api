package service

import (
	"math"
	"time"
)

const subscriptionDayDuration = 24 * time.Hour

type UserSubscription struct {
	ID      int64
	UserID  int64
	GroupID int64
	// SourceGroupID records the legacy APC subscription group used to create
	// this wallet. Runtime billing must not use it for authorization.
	SourceGroupID *int64
	PlanName      string

	StartsAt  time.Time
	ExpiresAt time.Time
	Status    string

	DailyWindowStart   *time.Time
	WeeklyWindowStart  *time.Time
	MonthlyWindowStart *time.Time

	DailyLimitUSD   *float64
	WeeklyLimitUSD  *float64
	MonthlyLimitUSD *float64

	DailyUsageUSD   float64
	WeeklyUsageUSD  float64
	MonthlyUsageUSD float64

	AssignedBy *int64
	AssignedAt time.Time
	Notes      string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	User           *User
	Group          *Group
	AssignedByUser *User
}

// SubscriptionUsageAllocation describes how much of a request can be charged to
// subscription quota before the remainder falls back to the user's balance.
type SubscriptionUsageAllocation struct {
	SubscriptionCost float64
	BalanceCost      float64
}

func (s *UserSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && time.Now().Before(s.ExpiresAt)
}

func (s *UserSubscription) IsSubscriptionWallet() bool {
	return s != nil
}

func (s *UserSubscription) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *UserSubscription) DaysRemaining() int {
	return s.daysRemainingAt(time.Now())
}

func (s *UserSubscription) daysRemainingAt(now time.Time) int {
	remaining := s.ExpiresAt.Sub(now)
	if remaining <= 0 {
		return 0
	}

	days := int(remaining / subscriptionDayDuration)
	if remaining%subscriptionDayDuration != 0 {
		days++
	}
	return days
}

func (s *UserSubscription) IsWindowActivated() bool {
	return s.DailyWindowStart != nil || s.WeeklyWindowStart != nil || s.MonthlyWindowStart != nil
}

func (s *UserSubscription) HasOneTimeDailyQuota() bool {
	if s == nil || s.StartsAt.IsZero() || s.ExpiresAt.IsZero() {
		return false
	}
	return !s.ExpiresAt.After(s.StartsAt.AddDate(0, 0, 1))
}

func (s *UserSubscription) NeedsDailyReset() bool {
	return s.NeedsDailyResetAt(time.Now())
}

func (s *UserSubscription) NeedsDailyResetAt(now time.Time) bool {
	if s.DailyWindowStart == nil {
		return false
	}
	if s.HasOneTimeDailyQuota() {
		return false
	}
	return !now.Before(s.DailyWindowStart.Add(24 * time.Hour))
}

func (s *UserSubscription) NeedsWeeklyReset() bool {
	return s.NeedsWeeklyResetAt(time.Now())
}

func (s *UserSubscription) NeedsWeeklyResetAt(now time.Time) bool {
	if s.WeeklyWindowStart == nil {
		return false
	}
	return !now.Before(s.WeeklyWindowStart.Add(7 * 24 * time.Hour))
}

func (s *UserSubscription) NeedsMonthlyReset() bool {
	return s.NeedsMonthlyResetAt(time.Now())
}

func (s *UserSubscription) NeedsMonthlyResetAt(now time.Time) bool {
	if s.MonthlyWindowStart == nil {
		return false
	}
	return !now.Before(s.MonthlyWindowStart.Add(30 * 24 * time.Hour))
}

func (s *UserSubscription) canAutomaticallyResetDailyAt(now time.Time) bool {
	_, ok := s.automaticWindowStartAt(s.DailyWindowStart, 24*time.Hour, now)
	return !s.HasOneTimeDailyQuota() && ok
}

func (s *UserSubscription) canAutomaticallyResetWeeklyAt(now time.Time) bool {
	_, ok := s.automaticWindowStartAt(s.WeeklyWindowStart, 7*24*time.Hour, now)
	return ok
}

func (s *UserSubscription) canAutomaticallyResetMonthlyAt(now time.Time) bool {
	_, ok := s.automaticWindowStartAt(s.MonthlyWindowStart, 30*24*time.Hour, now)
	return ok
}

func (s *UserSubscription) automaticWindowStartAt(previous *time.Time, period time.Duration, now time.Time) (time.Time, bool) {
	if previous == nil {
		return time.Time{}, false
	}

	anchor := *previous
	// Older subscriptions initialized their first windows at midnight on their
	// start date. Only that initial value is unambiguous; later midnight anchors
	// may be manual resets and must remain authoritative.
	legacyAnchor := startOfDay(s.StartsAt)
	if legacyAnchor.Before(s.StartsAt) && anchor.Equal(legacyAnchor) {
		anchor = s.StartsAt
	}
	next := anchor.Add(period)
	if now.Before(next) || !next.Before(s.ExpiresAt) {
		return time.Time{}, false
	}

	periods := now.Sub(anchor) / period
	lastPeriodBeforeExpiry := (s.ExpiresAt.Sub(anchor) - 1) / period
	if periods > lastPeriodBeforeExpiry {
		periods = lastPeriodBeforeExpiry
	}
	return anchor.Add(periods * period), true
}

func (s *UserSubscription) DailyResetTime() *time.Time {
	if s.DailyWindowStart == nil {
		return nil
	}
	if s.HasOneTimeDailyQuota() {
		t := s.ExpiresAt
		return &t
	}
	t := s.DailyWindowStart.Add(24 * time.Hour)
	return &t
}

func (s *UserSubscription) WeeklyResetTime() *time.Time {
	if s.WeeklyWindowStart == nil {
		return nil
	}
	t := s.WeeklyWindowStart.Add(7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) MonthlyResetTime() *time.Time {
	if s.MonthlyWindowStart == nil {
		return nil
	}
	t := s.MonthlyWindowStart.Add(30 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) HasDailyLimit() bool {
	return s != nil && s.DailyLimitUSD != nil && *s.DailyLimitUSD > 0
}

func (s *UserSubscription) HasWeeklyLimit() bool {
	return s != nil && s.WeeklyLimitUSD != nil && *s.WeeklyLimitUSD > 0
}

func (s *UserSubscription) HasMonthlyLimit() bool {
	return s != nil && s.MonthlyLimitUSD != nil && *s.MonthlyLimitUSD > 0
}

func (s *UserSubscription) CheckDailyLimit(additionalCost float64) bool {
	if !s.HasDailyLimit() {
		return true
	}
	return s.DailyUsageUSD+additionalCost <= *s.DailyLimitUSD
}

func (s *UserSubscription) CheckWeeklyLimit(additionalCost float64) bool {
	if !s.HasWeeklyLimit() {
		return true
	}
	return s.WeeklyUsageUSD+additionalCost <= *s.WeeklyLimitUSD
}

func (s *UserSubscription) CheckMonthlyLimit(additionalCost float64) bool {
	if !s.HasMonthlyLimit() {
		return true
	}
	return s.MonthlyUsageUSD+additionalCost <= *s.MonthlyLimitUSD
}

func (s *UserSubscription) CheckAllLimits(additionalCost float64) (daily, weekly, monthly bool) {
	daily = s.CheckDailyLimit(additionalCost)
	weekly = s.CheckWeeklyLimit(additionalCost)
	monthly = s.CheckMonthlyLimit(additionalCost)
	return
}

func (s *UserSubscription) RemainingQuotaUSD() float64 {
	if s == nil {
		return 0
	}

	remaining := -1.0
	applyLimit := func(limit *float64, used float64) {
		if limit == nil || *limit <= 0 {
			return
		}
		left := *limit - used
		if left < 0 {
			left = 0
		}
		if remaining < 0 || left < remaining {
			remaining = left
		}
	}

	applyLimit(s.DailyLimitUSD, s.DailyUsageUSD)
	applyLimit(s.WeeklyLimitUSD, s.WeeklyUsageUSD)
	applyLimit(s.MonthlyLimitUSD, s.MonthlyUsageUSD)
	if remaining < 0 {
		return math.MaxFloat64
	}
	return remaining
}

func (s *UserSubscription) AllocateUsageCost(costUSD float64) SubscriptionUsageAllocation {
	if costUSD <= 0 || s == nil {
		return SubscriptionUsageAllocation{}
	}
	remaining := s.RemainingQuotaUSD()
	if remaining <= 0 {
		return SubscriptionUsageAllocation{BalanceCost: costUSD}
	}
	if remaining >= costUSD {
		return SubscriptionUsageAllocation{SubscriptionCost: costUSD}
	}
	return SubscriptionUsageAllocation{
		SubscriptionCost: remaining,
		BalanceCost:      costUSD - remaining,
	}
}
