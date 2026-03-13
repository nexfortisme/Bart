package classifier

import "fmt"

const (
	IntentPositive  = "positive"
	IntentNegative  = "negative"
	IntentPassive   = "passive"
	IntentAmbiguous = "ambiguous"
)

func NewClassifier(embedder Embedder, store *MemoryStore) *Classifier {
	return &Classifier{
		embedder:   embedder,
		store:      store,
		numResults: 5,
		threshold:  0.5,
	}
}

func (c *Classifier) WithNumResults(numResults int) *Classifier {
	c.numResults = numResults
	return c
}

func (c *Classifier) WithThreshold(threshold float32) *Classifier {
	c.threshold = threshold
	return c
}

func (c *Classifier) Classify(text string) (ClassifierResult, error) {
	if c.store.Len() == 0 {
		return ClassifierResult{}, fmt.Errorf("store is empty")
	}

	vector, err := c.embedder.Embed(text)
	if err != nil {
		return ClassifierResult{}, fmt.Errorf("embedding failed: %w", err)
	}

	matches := c.store.Query(vector, c.numResults)

	// Weighted vote — each neighbor contributes its similarity score
	votes := map[string]float32{}
	for _, match := range matches {
		if match.Similarity >= c.threshold {
			votes[match.Entry.Intent] += match.Similarity
		}
	}

	// Find the winning intent
	var winner string
	var topScore float32
	for intent, score := range votes {
		if score > topScore {
			topScore = score
			winner = intent
		}
	}

	// Nothing cleared the threshold
	if winner == "" {
		winner = IntentAmbiguous
	}

	return ClassifierResult{
		Intent:     winner,
		Confidence: topScore,
		TopMatches: matches,
	}, nil
}
