package flashcard_sets

import (
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type FlashCardsCreateRequest struct {
	SetId  decimal.Decimal `json:"setId"`
	UserId string
	Cards  []InsertFlashCards `json:"flashCards"`
}

func (r FlashCardsCreateRequest) Validate() error {
	if r.SetId.IsZero() {
		return errors.New("setId is required")
	}
	if len(r.Cards) == 0 {
		return errors.New("cards is required")
	}
	return nil
}

type FlashCardsUpdateRequest struct {
	Id      decimal.Decimal `json:"id"`
	Front   *string         `json:"front,omitempty"`
	Back    *string         `json:"back,omitempty"`
	Choices *[]string       `json:"choices,omitempty"`
	Status  *string         `json:"status,omitempty"`
	UserId  string          // from middleware
}

func (r FlashCardsUpdateRequest) Validate() error {
	if r.Id.IsZero() {
		return errors.New("id is required")
	}
	if r.Status != nil {
		if *r.Status != CardStatusStudying && *r.Status != CardStatusLearned &&
			*r.Status != CardStatusWrongAnswerInTest &&
			*r.Status != CardStatusCorrectAnswerInTest &&
			*r.Status != CardStatusCorrectAnswerInLearn &&
			*r.Status != CardStatusWrongAnswerInLearn {
			return errors.New("status must be 'studying' or 'learned'")
		}
	}
	return nil
}

type FlashCardsDeleteRequest struct {
	Id     decimal.Decimal `json:"id"`
	UserId string          // from middleware
}

func (r FlashCardsDeleteRequest) Validate() error {
	if r.Id.IsZero() {
		return errors.New("id is required")
	}
	return nil
}
