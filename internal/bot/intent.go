package bot

import (
	"fmt"
	"os"

	"github.com/nexfortisme/bart/internal/classifier"
)

const (
	PRINT_INTENT_RESULTS = true
	MIN_CONFIDENCE       = 0.7
)

func MessageIntendedForBartClassifier(message string, stores map[string]*classifier.MemoryStore) classifier.MessageIntent {
	result, err := classifier.NewClassifier(classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")), stores["message_intent"]).WithThreshold(0.7).Classify(message)
	if err != nil {
		fmt.Printf("Error classifying message intent: %v\n", err)
		return classifier.MessageIntentAmbiguous
	}

	if PRINT_INTENT_RESULTS {
		printIntentResults(message, result)
	}

	if result.Confidence < MIN_CONFIDENCE {
		return classifier.MessageIntentAmbient
	}

	return classifier.MessageIntent(result.Intent)
}

func ToolIntentClassifier(message string, stores map[string]*classifier.MemoryStore) classifier.ToolIntent {
	result, err := classifier.NewClassifier(classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")), stores["tool_intent"]).WithThreshold(0.5).Classify(message)
	if err != nil {
		return classifier.ToolIntentNull // TODO - Update to be ambiguous
	}

	if PRINT_INTENT_RESULTS {
		printIntentResults(message, result)
	}

	if result.Confidence < MIN_CONFIDENCE {
		return classifier.ToolIntentNull // TODO - Update to be ambiguous
	}

	return classifier.ToolIntent(result.Intent)
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
