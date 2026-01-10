package voice

type VoiceRequest struct {
	Text string `json:"text,required"`
}

type TtsRequestToHomeProxy struct {
	Prompt string `json:"prompt"`
}

type TtsResponseFromHomeProxy struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Body    TtsResponseBodyFromHomeProxy
}
type TtsResponseBodyFromHomeProxy struct {
	Key string `json:"key"`
	Url string `json:"url"`
}
