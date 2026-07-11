//go:build unit

package service

import (
	"testing"
)

// TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier locks in the fix
// that subscription-mode billing honours the group (and any user-specific) rate
// multiplier — i.e. cmd.SubscriptionCost tracks ActualCost (= TotalCost *
// RateMultiplier), not raw TotalCost.
func TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	subID := int64(42)
	limit := 100.0
	subscriptionGroup := &Group{ID: 99, SubscriptionType: SubscriptionTypeStandard}

	tests := []struct {
		name           string
		totalCost      float64
		actualCost     float64
		isSubscription bool
		wantSub        float64
		wantBalance    float64
	}{
		{
			name:           "subscription with 2x multiplier consumes 2x quota",
			totalCost:      1.0,
			actualCost:     2.0,
			isSubscription: true,
			wantSub:        2.0,
			wantBalance:    0,
		},
		{
			name:           "subscription with 0.5x multiplier consumes 0.5x quota",
			totalCost:      1.0,
			actualCost:     0.5,
			isSubscription: true,
			wantSub:        0.5,
			wantBalance:    0,
		},
		{
			name:           "free subscription (multiplier 0) consumes no quota",
			totalCost:      1.0,
			actualCost:     0,
			isSubscription: true,
			wantSub:        0,
			wantBalance:    0,
		},
		{
			name:           "balance billing keeps using ActualCost (regression)",
			totalCost:      1.0,
			actualCost:     2.0,
			isSubscription: false,
			wantSub:        0,
			wantBalance:    2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &postUsageBillingParams{
				Cost:               &CostBreakdown{TotalCost: tt.totalCost, ActualCost: tt.actualCost},
				User:               &User{ID: 1},
				APIKey:             &APIKey{ID: 2, GroupID: &groupID},
				Account:            &Account{ID: 3},
				Subscription:       &UserSubscription{ID: subID, GroupID: subscriptionGroup.ID, Group: subscriptionGroup, DailyLimitUSD: &limit},
				IsSubscriptionBill: tt.isSubscription,
			}

			cmd := buildUsageBillingCommand("req-1", nil, p)
			if cmd == nil {
				t.Fatal("buildUsageBillingCommand returned nil")
			}
			if cmd.SubscriptionCost != tt.wantSub {
				t.Errorf("SubscriptionCost = %v, want %v", cmd.SubscriptionCost, tt.wantSub)
			}
			if cmd.BalanceCost != tt.wantBalance {
				t.Errorf("BalanceCost = %v, want %v", cmd.BalanceCost, tt.wantBalance)
			}
		})
	}
}

func TestBuildUsageBillingCommand_SubscriptionIsUserWalletAcrossRequestGroups(t *testing.T) {
	t.Parallel()

	openAIGroupID := int64(7)
	apcGroupID := int64(42)
	limit := 3.0
	p := &postUsageBillingParams{
		Cost: &CostBreakdown{TotalCost: 2, ActualCost: 2},
		User: &User{ID: 1},
		APIKey: &APIKey{
			ID:      2,
			GroupID: &openAIGroupID,
			Group:   &Group{ID: openAIGroupID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard},
		},
		Account: &Account{ID: 3},
		Subscription: &UserSubscription{
			ID:            4,
			GroupID:       apcGroupID,
			Group:         &Group{ID: apcGroupID, Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard},
			DailyLimitUSD: &limit,
			DailyUsageUSD: 1,
		},
		IsSubscriptionBill: true,
	}

	cmd := buildUsageBillingCommand("req-cross-group", nil, p)
	if cmd == nil {
		t.Fatal("buildUsageBillingCommand returned nil")
	}
	if cmd.SubscriptionID == nil || *cmd.SubscriptionID != p.Subscription.ID {
		t.Fatalf("SubscriptionID = %v, want %d", cmd.SubscriptionID, p.Subscription.ID)
	}
	if cmd.SubscriptionCost != 2 {
		t.Errorf("SubscriptionCost = %v, want 2", cmd.SubscriptionCost)
	}
	if cmd.BalanceCost != 0 {
		t.Errorf("BalanceCost = %v, want 0", cmd.BalanceCost)
	}
}

func TestBuildUsageBillingCommand_SubscriptionRemainderFallsBackToBalance(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	subID := int64(42)
	limit := 5.0
	p := &postUsageBillingParams{
		Cost:               &CostBreakdown{TotalCost: 2, ActualCost: 2},
		User:               &User{ID: 1},
		APIKey:             &APIKey{ID: 2, GroupID: &groupID},
		Account:            &Account{ID: 3},
		Subscription:       &UserSubscription{ID: subID, GroupID: groupID, Group: &Group{ID: groupID, SubscriptionType: SubscriptionTypeStandard}, DailyLimitUSD: &limit, DailyUsageUSD: 4.25},
		IsSubscriptionBill: true,
	}

	cmd := buildUsageBillingCommand("req-split", nil, p)
	if cmd == nil {
		t.Fatal("buildUsageBillingCommand returned nil")
	}
	if cmd.SubscriptionCost != 0.75 {
		t.Errorf("SubscriptionCost = %v, want 0.75", cmd.SubscriptionCost)
	}
	if cmd.BalanceCost != 1.25 {
		t.Errorf("BalanceCost = %v, want 1.25", cmd.BalanceCost)
	}
}
