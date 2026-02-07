package exam_sessions

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
)

const (
	ACTIVE    = "ACTIVE"
	SUBMITTED = "SUBMITTED"
	EXPIRED   = "EXPIRED"
	CANCELLED = "CANCELLED"
)

type StartExamRequest struct {
	SetId *decimal.Decimal `json:"sourceSetId,omitempty"`
	Mode  string           `json:"mode"`
	//ClassId           *decimal.Decimal  `json:"classId,omitempty"`
	DailyPlanId      *decimal.Decimal `json:"planId,omitempty"`
	QuestionCount    decimal.Decimal  `json:"totalQuestions"`
	TimeLimitSeconds *int64           `json:"timeLimitSec,omitempty"`
	TimeLimit        *time.Time
	UserId           string
}

func (r StartExamRequest) Validate() error {
	if r.SetId == nil && r.DailyPlanId == nil {
		return errors.New("one of setId, classId or dailyPlanId must be provided")
	}
	if r.QuestionCount.IsZero() || r.QuestionCount.IsNegative() {
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
	ID             int64      `json:"id"`
	Status         string     `json:"status"` // ACTIVE|SUBMITTED|EXPIRED|CANCELLED
	Mode           string     `json:"mode"`
	TotalQuestions int        `json:"totalQuestions"`
	ScoreTotal     int        `json:"scoreTotal"`
	ScoreMax       int        `json:"scoreMax"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	SubmittedAt    *time.Time `json:"submittedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`

	Questions []ExamQuestionDto `json:"questions"`
}

type ExamQuestionDto struct {
	QuestionID       int64    `json:"questionId"`
	Seq              int      `json:"seq"`
	CardID           int64    `json:"cardId"`
	QuestionType     string   `json:"questionType"`
	FrontSnapshot    string   `json:"frontSnapshot"`
	BackSnapshot     string   `json:"backSnapshot"`
	ChoicesSnapshot  []string `json:"choicesSnapshot,omitempty"`
	PromptTtsCacheId *int64   `json:"promptTtsCacheId,omitempty"`
	ScoreMax         int      `json:"scoreMax"`

	Answer *ExamAnswerDto `json:"answer,omitempty"`
}

type ExamAnswerDto struct {
	AnswerID           int64           `json:"answerId"`
	SelectedChoice     *string         `json:"selectedChoice,omitempty"`
	TypedText          *string         `json:"typedText,omitempty"`
	AudioURL           *string         `json:"audioUrl,omitempty"`
	RecognizedText     *string         `json:"recognizedText,omitempty"`
	PronunciationScore *int            `json:"pronunciationScore,omitempty"`
	IsCorrect          *string         `json:"isCorrect,omitempty"`    // Y|N
	ScoreAwarded       *int            `json:"scoreAwarded,omitempty"` // 0..scoreMax
	AnsweredAt         *time.Time      `json:"answeredAt,omitempty"`
	Detail             json.RawMessage `json:"detail,omitempty"` // jsonb
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
	SessionId   decimal.Decimal `json:"sessionId"`
	SeqId       decimal.Decimal `json:"seqId"`
	AnswerType  string          `json:"answerType"`
	Choice      string          `json:"choice"`
	UserIdToken string
}

func (r ExamSessionUpdateRequest) Validate() error {
	if utils.GetIndexFromString(r.Choice) == 0 {
		return errors.New("choice must start with a letter")
	}
	if r.SessionId.IsZero() {
		return errors.New("sessionId must be provided")
	}
	if r.SeqId.IsZero() {
		return errors.New("seqId must be provided")
	}
	return nil
}

type ExamSessionListRequest struct {
	Page     decimal.Decimal `json:"page,required"`
	Size     decimal.Decimal `json:"size,required"`
	SearchBy string          `json:"searchBy"`
}

type ExamSessionListResponse struct {
	Content       []ExamSessionListResponseDetails `json:"content"`
	TotalPage     decimal.Decimal                  `json:"totalPage"`
	TotalElements decimal.Decimal                  `json:"totalElements"`
}

type ExamSessionListResponseDetails struct {
	ID             string     `json:"id" db:"id"`
	Mode           string     `json:"mode" db:"mode"`
	SourceSetID    *string    `json:"sourceSetId" db:"source_set_id"`
	UserIDToken    string     `json:"userIdToken" db:"user_id_token"`
	PlanID         *string    `json:"planId" db:"plan_id"`
	TotalQuestions int        `json:"totalQuestions" db:"total_questions"`
	TimeLimitSec   *int       `json:"timeLimitSec" db:"time_limit_sec"`
	Status         string     `json:"status" db:"status"`
	ScoreTotal     float64    `json:"scoreTotal" db:"score_total"`
	ScoreMax       int        `json:"scoreMax" db:"score_max"`
	StartedAt      time.Time  `json:"startedAt" db:"started_at"`
	ExpiresAt      *time.Time `json:"expiresAt" db:"expires_at"`
	SubmittedAt    *time.Time `json:"submittedAt" db:"submitted_at"`
	CreatedAt      time.Time  `json:"createdAt" db:"create_at"`
}

type ExamSubmitItem struct {
	CardID       int64
	Source       string
	IsCorrect    string
	Grade        int16
	AnswerDetail json.RawMessage
}
