package flashcard_sets

import (
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
)

type InsertFlashCards struct {
	Front   string   `json:"front"`
	Back    string   `json:"back"`
	Choices []string `json:"choices"`
}

type FlashCardSetsCreateRequest struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	IsPublic    string              `json:"isPublic"`
	FlashCards  *[]InsertFlashCards `json:"flashCards,omitempty"`
	OwnerId     int
	Username    string
}

func (r *FlashCardSetsCreateRequest) Validate() error {
	if r.Title == "" {
		return errors.New("title is required")
	}
	if r.Description == "" {
		return errors.New("description is required")
	}
	if r.IsPublic == "" {
		return errors.New("isPublic is required")
	}
	if r.IsPublic != utils.FlagY && r.IsPublic != utils.FlagN {
		return errors.New("isPublic is only Y or N")
	}
	return nil
}

type ResetFlashCardStatusRequest struct {
	SetID  decimal.Decimal `json:"setId"`
	Status string          `json:"status"`
}

func (b *ResetFlashCardStatusRequest) Validate() error {
	if b.SetID.IsZero() {
		return errors.New("setId is required and must be > 0")
	}
	switch b.Status {
	case CardStatusStudying, CardStatusLearned:
		return nil
	default:
		return errors.New(`status must be either "studying" or "learned"`)
	}
}

type DuplicateFlashCardsSetRequest struct {
	OldSetID decimal.Decimal `json:"setId"`
	OwnerID  int
	Username string
}

func (r DuplicateFlashCardsSetRequest) Validate() error {
	if r.OldSetID.IsZero() {
		return errors.New("oldSetId is required")
	}
	return nil
}

type FlashCardSetsTrackerUpsert struct {
	SetID   decimal.Decimal `json:"setId"`
	CardID  decimal.Decimal `json:"cardId"`
	OwnerID int
}

func (r FlashCardSetsTrackerUpsert) Validate() error {
	if r.SetID.IsZero() {
		return errors.New("setId is required")
	}
	if r.CardID.IsZero() {
		return errors.New("cardId is required")
	}
	return nil
}

type FlashCardSetsUpdateRequest struct {
	Id          decimal.Decimal `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	IsPublic    string          `json:"isPublic"`
	Username    string
}

// Validate ensures we have the minimum required data to run an UPDATE.
func (r FlashCardSetsUpdateRequest) Validate() error {
	if r.Id.IsZero() {
		return errors.New("id is required")
	}
	return nil
}

type FlashCardSetsDeleteRequest struct {
	Id       decimal.Decimal `json:"id"`
	Username string
}

func (r FlashCardSetsDeleteRequest) Validate() error {
	if r.Id.IsZero() {
		return errors.New("id is required")
	}
	return nil
}

type FlashCardSetsListRequest struct {
	Page     decimal.Decimal `json:"page"`
	Size     decimal.Decimal `json:"size"`
	IsMine   string          `json:"isMine"`
	IsPublic string          `json:"isPublic"`
	SearchBy string          `json:"searchBy"`
}

func (r FlashCardSetsListRequest) Validate() error {
	if r.Page.IsZero() || r.Page.IsNegative() {
		return errors.New("page is required")
	}
	if r.Size.IsZero() || r.Size.IsNegative() {
		return errors.New("size is required")
	}
	if r.Size.LessThan(decimal.NewFromInt(10)) {
		return errors.New("size is less than 10")
	}
	return nil
}

type FlashCardSetsListResponseDetails struct {
	SetId       decimal.Decimal `json:"setId"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	IsPublic    string          `json:"isPublic"`
	OwnerId     int             `json:"ownerId"`
	OwnerName   string          `json:"ownerName"`
	Term        decimal.Decimal `json:"term"`
}
type FlashCardSetsListResponse struct {
	Content       []FlashCardSetsListResponseDetails `json:"content"`
	TotalPage     decimal.Decimal                    `json:"totalPage"`
	TotalElements decimal.Decimal                    `json:"totalElements"`
}

type FlashCardSetsInquiryResponse struct {
	Id        decimal.Decimal `json:"id"`
	Front     string          `json:"front"`
	Back      string          `json:"back"`
	Choices   []string        `json:"choices"`
	Status    string          `json:"status"`
	CreateAt  time.Time       `json:"createAt"`
	OwnerName string          `json:"ownerName"`
	IsCurrent bool            `json:"isCurrent"`
	Seq       decimal.Decimal `json:"seq"`
}

// TODO can import with update setId a
type FlashCardsSetsCsvImportRequest struct {
	SetId       decimal.Decimal `json:"setId"`
	CommandRec  string          `json:"commandRec"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	IsPublic    string          `json:"isPublic"`
	File        string          `json:"file"` // base64‑encoded CSV
	OwnerId     int             `json:"-"`
	Username    string          `json:"-"`
}

func (r FlashCardsSetsCsvImportRequest) Validate() error {
	if r.Title == "" {
		return errors.New("title is required")
	}
	if r.Description == "" {
		return errors.New("description is required")
	}
	if r.IsPublic != "Y" && r.IsPublic != "N" {
		return errors.New("isPublic must be Y or N")
	}
	if r.File == "" {
		return errors.New("file (base64 CSV) is required")
	}
	if r.CommandRec != "" && len([]rune(r.CommandRec)) != 1 {
		return errors.New("commandRec must be exactly one character")
	}
	return nil
}
