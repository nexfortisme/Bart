package bot

import (
	"fmt"
	"os"

	"github.com/nexfortisme/bart/internal/classifier"
)

const (
	PRINT_INTENT_RESULTS = false
)

func MessageIntendedForBartClassifier(message string, stores map[string]*classifier.MemoryStore) string {
	result, err := classifier.NewClassifier(classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")), stores["message_intent"]).WithThreshold(0.7).Classify(message)
	if err != nil {
		return ""
	}

	if PRINT_INTENT_RESULTS {
		printIntentResults(message, result)
	}

	return result.Intent
}

func ToolIntentClassifier(message string, stores map[string]*classifier.MemoryStore) string {
	result, err := classifier.NewClassifier(classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")), stores["tool_intent"]).WithThreshold(0.5).Classify(message)
	if err != nil {
		return ""
	}

	if PRINT_INTENT_RESULTS {
		printIntentResults(message, result)
	}

	return result.Intent
}

func printIntentResults(message string, result classifier.ClassifierResult) {
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
}
