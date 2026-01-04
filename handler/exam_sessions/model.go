package exam_sessions

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
)

type StartExamRequest struct {
	SetId             *decimal.Decimal  `json:"setId,omitempty"`
	ClassId           *decimal.Decimal  `json:"classId,omitempty"`
	DailyPlanId       *decimal.Decimal  `json:"dailyPlanId,omitempty"`
	QuestionCount     []decimal.Decimal `json:"questionCount"`
	ExamTotalQuestion decimal.Decimal   `json:"examTotalQuestion"`
	TimeLimit         *time.Time        `json:"timeLimit,omitempty"`
	Username          string
}

func (r StartExamRequest) Validate() error {
	if r.SetId == nil && r.ClassId == nil && r.DailyPlanId == nil {
		return errors.New("one of setId, classId or dailyPlanId must be provided")
	}
	if r.ExamTotalQuestion.IsZero() || r.ExamTotalQuestion.IsNegative() {
		return errors.New("examTotalQuestion must be provided")
	}
	return nil
}

type FlashCardDetails struct {
	Id        decimal.Decimal `json:"id"`
	Front     string          `json:"front"`
	Back      string          `json:"back"`
	Choices   []string        `json:"choices"`
	Status    string          `json:"status"`
	CreateAt  time.Time       `json:"createAt"`
	OwnerName string          `json:"ownerName"`
	Seq       decimal.Decimal `json:"seq"`
}

type StartExamResponse struct {
	Id          decimal.Decimal    `json:"id"`
	Questions   []FlashCardDetails `json:"questions"`
	IsSubmitted string             `json:"isSubmitted"`
	ExpireAt    *time.Time         `json:"expireAt"`
}
type ExamSessionDto struct {
	QuestionIDs []int64
	IsSubmitted string
	Score       *int
	ExpiresAt   *time.Time
	OwnerID     int
	OwnerName   string
	Answers     *json.RawMessage
}

type InquiryExamResponse struct {
	Id             decimal.Decimal    `json:"id"`
	Questions      []FlashCardDetails `json:"questions"`
	IsSubmitted    string             `json:"isSubmitted"`
	Score          *int               `json:"score"`
	OwnerID        int                `json:"ownerId"`
	OwnerName      string             `json:"ownerName"`
	TotalQuestions int                `json:"totalQuestions"`
	ExpireAt       *time.Time         `json:"expireAt"`
	Answers        *json.RawMessage   `json:"answers"`
}

type AnswersDetail struct {
	CurrentId  decimal.Decimal `json:"currentId"`
	AnswerList []struct {
		CardId  decimal.Decimal `json:"cardId"`
		Seq     decimal.Decimal `json:"seq"`
		Answer  string          `json:"answer"`
		Correct bool
	} `json:"answerList"`
}

type ExamSessionUpdateRequest struct {
	Id          decimal.Decimal `json:"id"`
	Answers     *AnswersDetail  `json:"answers,omitempty"`
	IsSubmitted *string         `json:"isSubmitted,omitempty"`
	Username    string
	Score       *int `json:"score,omitempty"`
}

func (r ExamSessionUpdateRequest) Validate() error {
	if r.Id.IsZero() {
		return errors.New("id is required")
	}
	if r.IsSubmitted != nil && *r.IsSubmitted != utils.FlagY && *r.IsSubmitted != utils.FlagN {
		return fmt.Errorf("IsSubmitted must be one of [%v, %v]", utils.FlagY, utils.FlagN)
	}
	return nil
}
func (r ExamSessionUpdateRequest) ToJson() string {
	if r.Answers != nil {
		jsonBytes, _ := json.Marshal(r.Answers)
		return string(jsonBytes)
	}
	return "{}"
}
