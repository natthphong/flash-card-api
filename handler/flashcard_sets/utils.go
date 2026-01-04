package flashcard_sets

import (
	"math/rand"
	"time"
)

const (
	CardStatusStudying             = "studying"
	CardStatusLearned              = "learned"
	CardStatusWrongAnswerInLearn   = "wrongAnswerInLearn"
	CardStatusCorrectAnswerInLearn = "correctAnswerInLearn"
	CardStatusWrongAnswerInTest    = "wrongAnswerInTest"
	CardStatusCorrectAnswerInTest  = "correctAnswerInTest"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetChoices(all []*InsertFlashCards, current *InsertFlashCards) []string {
	if len(current.Choices) >= 4 {
		return current.Choices
	}

	var pool []string
	for _, c := range all {
		if c != current {
			pool = append(pool, c.Back)
		}
	}

	rand.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})

	picks := pool
	if len(pool) > 3 {
		picks = pool[:3]
	}

	choices := append([]string{current.Back}, picks...)
	rand.Shuffle(len(choices), func(i, j int) {
		choices[i], choices[j] = choices[j], choices[i]
	})
	return choices
}
