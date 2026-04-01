package bot

import (
	"fmt"
	"os"

	"github.com/nexfortisme/bart/internal/classifier"
)

const (
	LOG_CLASSIFICATIONS = true
)

func MessageIntendedForBartClassifier(message string, stores map[string]*classifier.MemoryStore) classifier.MessageIntent {
	result, err := classifier.NewClassifier[classifier.MessageIntent](
		classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")),
		stores["message_intent"],
	).WithThreshold(0.7).WithFallbackIntent(classifier.MessageIntentAmbiguous).Classify(message)
	if err != nil {
		return classifier.MessageIntentAmbiguous
	}

	if LOG_CLASSIFICATIONS {
		logClassifications(message, result)
	}

	return result.Intent
}

func ToolIntentClassifier(message string, stores map[string]*classifier.MemoryStore) classifier.ToolIntent {
	result, err := classifier.NewClassifier[classifier.ToolIntent](
		classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")),
		stores["tool_intent"],
	).WithThreshold(0.5).WithFallbackIntent(classifier.ToolIntentNull).Classify(message)
	if err != nil {
		return classifier.ToolIntentNull
	}

	if LOG_CLASSIFICATIONS {
		logClassifications(message, result)
	}

	return result.Intent
}

func logClassifications[T classifier.IntentType](message string, result classifier.ClassifierResult[T]) {
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

	fmt.Println()
}
