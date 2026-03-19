package classifier

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type seedData struct {
	MessageIntent []seedEntry      `json:"message_intent"`
	ToolIntent    []seedEntry      `json:"tool_intent"`
	EdgeCases     []edgeCaseEntry  `json:"edge_cases"`
}

type seedEntry struct {
	Text  string `json:"text"`
	Label string `json:"label"`
}

type edgeCaseEntry struct {
	Text          string `json:"text"`
	MessageIntent string `json:"message_intent"`
	ToolIntent    *string `json:"tool_intent"`
}

var (
	pathToSeedData = "resources/embeddings/bart_classifier_seed_data_v1-1.json"
)

func SeedEmbeddingsDataset() {
	seedSection(pathToSeedData, IntentTypeMessage, MessageIntentStorePath)
	seedSection(pathToSeedData, IntentTypeTool, ToolIntentStorePath)
}

func seedSection(dataPath string, intentType IntentType, storePath string) {
	examples, err := loadSection(dataPath, intentType)
	if err != nil {
		fmt.Printf("error loading %s: %v\n", intentType, err)
		os.Exit(1)
	}

	fmt.Printf("loaded %d examples from [%s] in %s\n\n", len(examples), intentType, dataPath)

	embedder := NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL"))
	store := NewStore()

	for i, ex := range examples {
		fmt.Printf("[%d/%d] embedding [%s]: %q\n", i+1, len(examples), ex.Intent, ex.Text)
		vector, err := embedder.Embed(ex.Text)
		if err != nil {
			fmt.Printf("  warning: skipping — %v\n", err)
			continue
		}
		store.Add(fmt.Sprintf("%d", i), ex.Text, ex.Intent, vector)
	}

	fmt.Printf("\nstored %d embeddings\n", store.Len())

	if err := store.Save(storePath); err != nil {
		fmt.Printf("error: could not save store: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("saved store to %s\n\n", storePath)
}

func loadSection(path string, intentType IntentType) ([]Example, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	var sd seedData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", path, err)
	}

	var entries []seedEntry
	var includeEdgeCaseFn func(e edgeCaseEntry) (label string, ok bool)

	switch intentType {
	case IntentTypeMessage:
		entries = sd.MessageIntent
		includeEdgeCaseFn = func(e edgeCaseEntry) (string, bool) {
			label := strings.TrimSpace(e.MessageIntent)
			return label, label != ""
		}
	case IntentTypeTool:
		entries = sd.ToolIntent
		includeEdgeCaseFn = func(e edgeCaseEntry) (string, bool) {
			if e.ToolIntent == nil {
				return "", false
			}

			label := strings.TrimSpace(*e.ToolIntent)
			if label == "" || strings.EqualFold(label, "null") {
				return "", false
			}

			return label, true
		}
	default:
		return nil, fmt.Errorf("unknown intent type: %s", intentType)
	}

	examples := make([]Example, 0, len(entries)+len(sd.EdgeCases))
	for _, e := range entries {
		examples = append(examples, Example{Text: e.Text, Intent: e.Label})
	}
	for _, e := range sd.EdgeCases {
		label, ok := includeEdgeCaseFn(e)
		if !ok {
			continue
		}
		examples = append(examples, Example{Text: e.Text, Intent: label})
	}
	return examples, nil
}
