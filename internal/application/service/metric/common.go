package metric

import (
	"regexp"
	"strings"
	"unicode"
)

func sum(m map[string]int) int {
	s := 0
	for _, v := range m {
		s += v
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func splitSentences(text string) []string {
	re := regexp.MustCompile(`([.。!?！？])`)

	split := re.Split(text, -1)

	var sentences []string
	current := strings.Builder{}

	for i, s := range split {
		if i%2 == 0 {
			current.WriteString(s)
		} else {
			if current.Len() > 0 {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		sentences = append(sentences, remaining)
	}

	return sentences
}

func splitIntoWords(sentences []string) []string {
	var tokens []string
	for _, text := range sentences {
		var current strings.Builder
		for _, r := range text {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				current.WriteRune(r)
			} else {
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				if unicode.IsPunct(r) || unicode.IsSymbol(r) {
					tokens = append(tokens, string(r))
				}
			}
		}
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
		}
	}
	return tokens
}

func ToSet[T comparable](li []T) map[T]struct{} {
	res := make(map[T]struct{}, len(li))
	for _, v := range li {
		res[v] = struct{}{}
	}
	return res
}

func SliceMap[T any, Y any](li []T, fn func(T) Y) []Y {
	res := make([]Y, len(li))
	for i, v := range li {
		res[i] = fn(v)
	}
	return res
}

func Hit[T comparable](li []T, set map[T]struct{}) int {
	count := 0
	for _, v := range li {
		if _, exist := set[v]; exist {
			count++
		}
	}
	return count
}

func Fold[T any, Y any](slice []T, initial Y, f func(Y, T) Y) Y {
	accumulator := initial
	for _, item := range slice {
		accumulator = f(accumulator, item)
	}
	return accumulator
}
