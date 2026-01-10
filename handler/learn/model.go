package learn

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

type ReviewSubMitRequest struct {
	CardId       decimal.Decimal `json:"cardId"`
	Source       string          `json:"source"`
	Grade        int             `json:"grade"`
	IsCorrect    string          `json:"isCorrect"`
	AnswerDetail json.RawMessage `json:"answerDetail"`
}
