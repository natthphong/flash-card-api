package voice

import (
	"regexp"
	"strings"
)

var (
	rePunct = regexp.MustCompile(`[^\p{L}\p{N}\s']+`) // keep letters, numbers, spaces, apostrophe
	reSpace = regexp.MustCompile(`\s+`)
)

func normalizeText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = rePunct.ReplaceAllString(s, " ")
	s = reSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func tokenizeWords(s string) []string {
	s = normalizeText(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, " ")
}

type WerReport struct {
	SourceWords []string `json:"sourceWords"`
	SttWords    []string `json:"sttWords"`

	N int `json:"n"` // source word count
	S int `json:"substitutions"`
	I int `json:"insertions"`
	D int `json:"deletions"`

	WER   float64 `json:"wer"`
	Score int     `json:"score"`

	Mismatches []Mismatch `json:"mismatches"` // optional but useful for UI
}

type Mismatch struct {
	Type       string `json:"type"` // "sub" | "ins" | "del"
	SourceWord string `json:"sourceWord,omitempty"`
	SttWord    string `json:"sttWord,omitempty"`
	Pos        int    `json:"pos"` // position in source (best-effort)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func ScoreByWER(sourceText, sttText string) WerReport {
	src := tokenizeWords(sourceText)
	hyp := tokenizeWords(sttText)

	n := len(src)
	// If source is empty, define score as 0 unless hyp also empty
	if n == 0 {
		score := 0
		if len(hyp) == 0 {
			score = 100
		}
		return WerReport{
			SourceWords: src,
			SttWords:    hyp,
			N:           0,
			S:           0,
			I:           len(hyp),
			D:           0,
			WER:         0,
			Score:       score,
		}
	}

	// dp costs
	dp := make([][]int, n+1)
	bt := make([][]byte, n+1) // backtrace: 'M' match/sub, 'I', 'D'
	for i := 0; i <= n; i++ {
		dp[i] = make([]int, len(hyp)+1)
		bt[i] = make([]byte, len(hyp)+1)
	}

	// init
	for i := 1; i <= n; i++ {
		dp[i][0] = i
		bt[i][0] = 'D'
	}
	for j := 1; j <= len(hyp); j++ {
		dp[0][j] = j
		bt[0][j] = 'I'
	}

	// fill
	for i := 1; i <= n; i++ {
		for j := 1; j <= len(hyp); j++ {
			costSub := 0
			if src[i-1] != hyp[j-1] {
				costSub = 1
			}
			a := dp[i-1][j] + 1         // delete
			b := dp[i][j-1] + 1         // insert
			c := dp[i-1][j-1] + costSub // match/sub

			dp[i][j] = c
			bt[i][j] = 'M'
			if a < dp[i][j] {
				dp[i][j] = a
				bt[i][j] = 'D'
			}
			if b < dp[i][j] {
				dp[i][j] = b
				bt[i][j] = 'I'
			}
		}
	}

	// backtrace for S/I/D and mismatch list
	i, j := n, len(hyp)
	S, I, D := 0, 0, 0
	mms := make([]Mismatch, 0)

	for i > 0 || j > 0 {
		switch bt[i][j] {
		case 'M':
			// move diag
			if i > 0 && j > 0 && src[i-1] != hyp[j-1] {
				S++
				mms = append(mms, Mismatch{
					Type:       "sub",
					SourceWord: src[i-1],
					SttWord:    hyp[j-1],
					Pos:        i - 1,
				})
			}
			i--
			j--
		case 'D':
			D++
			mms = append(mms, Mismatch{
				Type:       "del",
				SourceWord: src[i-1],
				Pos:        i - 1,
			})
			i--
		case 'I':
			I++
			mms = append(mms, Mismatch{
				Type:    "ins",
				SttWord: hyp[j-1],
				Pos:     i, // insertion around current source index
			})
			j--
		default:
			// fallback safety
			if i > 0 {
				i--
			} else if j > 0 {
				j--
			}
		}
	}

	wer := float64(S+I+D) / float64(n)
	score := clampInt(int((1.0-wer)*100.0+0.5), 0, 100)

	return WerReport{
		SourceWords: src,
		SttWords:    hyp,
		N:           n,
		S:           S,
		I:           I,
		D:           D,
		WER:         wer,
		Score:       score,
		Mismatches:  reverseMismatch(mms), // reverse to be in reading order
	}
}

func reverseMismatch(a []Mismatch) []Mismatch {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
	return a
}
