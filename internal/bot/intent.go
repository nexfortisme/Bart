package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/nexfortisme/bart/internal/classifier"
)

// Super Basic Slop Job From Codex. Should probably be improved with RAG or something like that.
func MessageIntendedForBotScored(message string) bool {
	normalized := strings.TrimSpace(message)
	if normalized == "" {
		return false
	}

	lower := strings.ToLower(normalized)
	score := 0

	positivePrefixes := []string{
		"bart,",
		"@bart",
		"<@",
	}
	for _, prefix := range positivePrefixes {
		if strings.HasPrefix(lower, prefix) {
			score += 3
		}
	}

	positivePhrases := []string{
		"can you",
		"could you",
		"would you",
		"will you",
		"help me",
		"summarize",
		"explain",
		"tell me",
		"show me",
		"what do you think",
	}
	for _, phrase := range positivePhrases {
		if strings.Contains(lower, phrase) {
			score += 2
		}
	}

	if strings.HasPrefix(lower, "bart ") {
		score += 1
	}

	negativePrefixes := []string{
		"can bart ",
		"does bart ",
		"is bart ",
		"will bart ",
		"would bart ",
		"could bart ",
		"should bart ",
	}
	for _, prefix := range negativePrefixes {
		if strings.HasPrefix(lower, prefix) {
			score -= 3
		}
	}

	negativePhrases := []string{
		"you should ask bart",
		"let me ask bart",
		"maybe bart can help",
		"bart can do that",
		"bart handles that",
		"bart is good at",
		"have you tried asking bart",
		"let's see what bart says",
		"i heard bart can",
		"bart doesn't",
		"bart isnt",
		"bart isn't",
		"bart might not",
		"i'm not sure bart can",
		"im not sure bart can",
		"the bart chatbot",
		"bart uses",
		"our company uses bart",
		"we integrated bart",
		"set bart up",
	}
	for _, phrase := range negativePhrases {
		if strings.Contains(lower, phrase) {
			score -= 3
		}
	}

	return score >= 2
}

func MessageIntendedForBartClassifier(message string, store *classifier.MemoryStore) string {
	result, err := classifier.NewClassifier(classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")), store).Classify(message)
	if err != nil {
		return ""
	}

	fmt.Printf("\nmessage:    %q\n", message)
	fmt.Printf("intent:     %s\n", result.Intent)
	fmt.Printf("confidence: %.4f\n", result.Confidence)
	fmt.Println("\ntop matches:")

	for i, match := range result.TopMatches {
		fmt.Printf("  %d. [%.4f] [%s] %q\n",
			i+1,
			match.Similarity,
			match.Entry.Intent,
			match.Entry.Text,
		)
	}

	return result.Intent
}
