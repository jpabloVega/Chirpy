package main

import (
	"slices"
	"strings"
)

func removeProfanity(text string) string {
	bad_words := []string{"kerfuffle", "sharbert", "fornax"}
	orgText := strings.Split(text, " ")
	sepText := strings.Split(strings.ToLower(text), " ")
	for i, word := range sepText {
		if slices.Contains(bad_words, word) {
			orgText[i] = "****"
		}
	}
	return strings.Join(orgText, " ")
}
