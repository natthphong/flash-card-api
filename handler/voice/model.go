package voice

import "github.com/shopspring/decimal"

type VoiceRequest struct {
	Text  string          `json:"text,required"`
	Speed decimal.Decimal `json:"speed"`
}

type TtsRequestToHomeProxy struct {
	Prompt string `json:"prompt"`
}

type TtsResponseFromHomeProxy struct {
	Key string `json:"key"`
	Url string `json:"url"`
}
