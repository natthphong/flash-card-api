package daily_plans

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
)

type DailyPlanSettingRequest struct {
	UserId      decimal.Decimal
	DailyActive string          `json:"dailyActive"`
	DailyTarget decimal.Decimal `json:"dailyTarget"`
}

func (r *DailyPlanSettingRequest) Validate() error {
	if r.DailyTarget.IsZero() {
		return fmt.Errorf("dailyTarget cannot be zero")
	}
	if !r.DailyTarget.IsPositive() {
		return fmt.Errorf("dailyTarget cannot be negative")
	}
	if r.DailyActive != utils.FlagY && r.DailyActive != utils.FlagN {
		return fmt.Errorf("dailyActive must be one of [%v, %v]", utils.FlagY, utils.FlagN)
	}
	return nil
}

type DailyFlashCardSetsInquiryResponse struct {
	DailyPlanId decimal.Decimal `json:"dailyPlanId"`
	Id          decimal.Decimal `json:"id"`
	Front       string          `json:"front"`
	Back        string          `json:"back"`
	Choices     []string        `json:"choices"`
	Status      string          `json:"status"`
	CreateAt    time.Time       `json:"createAt"`
	OwnerName   string          `json:"ownerName"`
	Seq         decimal.Decimal `json:"seq"`
}
