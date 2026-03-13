package classifier

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// These structs are to handle the fact that the dataset is nested and has multiple turns to the conversation
type dataset struct {
	Examples []datasetExample `json:"examples"`
}

type datasetExample struct {
	Type string `json:"type"`
	Messages []datasetMessage `json:"messages"`
}

type datasetMessage struct {
	Turn int `json:"turn"`
	Text string `json:"text"`
}

var (
	pathToExamples = "resources/embeddings/test_embeddings_chatbot_interaction_dataset_v4_discord_noisy.json"
)

func SeedEmbeddingsDataset() {
	examples, err := loadExamples(pathToExamples)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("loaded %d examples from %s\n\n", len(examples), pathToExamples)

	lmStudioEmbedder := NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL"))
	store := NewStore()

	for i, ex := range examples {
		fmt.Printf("[%d/%d] embedding [%s]: %q\n", i+1, len(examples), ex.Intent, ex.Text)
		vector, err := lmStudioEmbedder.Embed(ex.Text)
		if err != nil {
			fmt.Printf("  warning: skipping — %v\n", err)
			continue
		}
		store.Add(fmt.Sprintf("%d", i), ex.Text, ex.Intent, vector)
	}

	fmt.Printf("\nstored %d embeddings\n", store.Len())

	// -- Persist to disk --
	if err := store.Save(storePath); err != nil {
		fmt.Printf("error: could not save store: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("saved store to %s\n", storePath)
}

func loadExamples(path string) ([]Example, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	// Detect format: dataset is a JSON object with an "examples" key
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		if _, ok := raw["examples"]; ok {
			var ds dataset
			if err := json.Unmarshal(data, &ds); err != nil {
				return nil, fmt.Errorf("could not parse dataset %s: %w", path, err)
			}
			return convertDataset(ds), nil
		}
	}

	// Fall back to flat array format
	var examples []Example
	if err := json.Unmarshal(data, &examples); err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", path, err)
	}
	return examples, nil
}

func convertDataset(ds dataset) []Example {
	examples := make([]Example, 0, len(ds.Examples))
	for _, ex := range ds.Examples {
		var parts []string
		for _, msg := range ex.Messages {
			parts = append(parts, msg.Text)
		}
		examples = append(examples, Example{
			Text:   strings.Join(parts, "\n"),
			Intent: ex.Type,
		})
	}
	return examples
}