package bot

import (
	"log"
	"os"

	"github.com/nexfortisme/bart/internal/classifier"
)

const (
	PRINT_INTENT_RESULTS = true
	MIN_CONFIDENCE       = 0.7
)

func MessageIntendedForBartClassifier(
	message string,
	stores map[string]*classifier.MemoryStore,
	logger *log.Logger,
) classifier.MessageIntent {
	result, err := classifier.NewClassifier(classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")), stores["message_intent"]).WithThreshold(0.7).Classify(message)
	if err != nil {
		logger.Printf("Error classifying message intent: %v\n", err)
		return classifier.MessageIntentAmbiguous
	}

	if PRINT_INTENT_RESULTS {
		printIntentResults(message, result, logger)
	}

	if result.Confidence < MIN_CONFIDENCE {
		logger.Printf("Message intent confidence below threshold: %.4f\n", result.Confidence)
		return classifier.MessageIntentAmbient
	}

	return classifier.MessageIntent(result.Intent)
}

func ToolIntentClassifier(
	message string,
	stores map[string]*classifier.MemoryStore,
	logger *log.Logger,
) classifier.ToolIntent {
	result, err := classifier.NewClassifier(classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")), stores["tool_intent"]).WithThreshold(0.5).Classify(message)
	if err != nil {
		logger.Printf("Error classifying tool intent: %v\n", err)
		return classifier.ToolIntentNull // TODO - Update to be ambiguous
	}

	if PRINT_INTENT_RESULTS {
		printIntentResults(message, result, logger)
	}

	if result.Confidence < MIN_CONFIDENCE {
		logger.Printf("Tool intent confidence below threshold: %.4f\n", result.Confidence)
		return classifier.ToolIntentNull // TODO - Update to be ambiguous
	}

	return classifier.ToolIntent(result.Intent)
}

func printIntentResults(
	message string,
	result classifier.ClassifierResult,
	logger *log.Logger,
) {
	logger.Printf("message:    %q\n", message)
	logger.Printf("intent:     %s\n", result.Intent)
	logger.Printf("confidence: %.4f\n", result.Confidence)
	logger.Printf("top matches:\n")

	for i, match := range result.TopMatches {
		logger.Printf("  %d. [%.4f] [%s] %q\n",
			i+1,
			match.Similarity,
			match.Entry.Intent,
			match.Entry.Text,
		)
	}
}
