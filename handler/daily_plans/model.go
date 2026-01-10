package daily_plans

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
)

type DailyPlanSettingRequest struct {
	UserIdToken  string
	DailyActive  *string          `json:"dailyActive,omitempty"`
	DailyTarget  *decimal.Decimal `json:"dailyTarget,omitempty"`
	DailySetId   *decimal.Decimal `json:"dailySetId,omitempty"`
	DefaultSetId *decimal.Decimal `json:"defaultSetId,omitempty"`
}

func (r *DailyPlanSettingRequest) Validate() error {
	if r.DailyTarget.IsZero() {
		return fmt.Errorf("dailyTarget cannot be zero")
	}
	if !r.DailyTarget.IsPositive() {
		return fmt.Errorf("dailyTarget cannot be negative")
	}

	if r.DailyActive != nil && *r.DailyActive != utils.FlagY && *r.DailyActive != utils.FlagN {
		return fmt.Errorf("dailyActive must be one of [%v, %v]", utils.FlagY, utils.FlagN)
	}
	if r.DailyActive != nil && *r.DailyActive == utils.FlagY && (r.DailySetId == nil && r.DefaultSetId == nil) {
		return fmt.Errorf("dailyActive must be one dailySetId or defaultSetId")
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
