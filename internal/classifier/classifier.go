package classifier

import "fmt"

func NewClassifier[T IntentType](embedder Embedder, store *MemoryStore) *Classifier[T] {
	return &Classifier[T]{
		embedder:       embedder,
		store:          store,
		numResults:     5,
		threshold:      0.5,
		fallbackIntent: *new(T),
	}
}

func (c *Classifier[T]) WithNumResults(numResults int) *Classifier[T] {
	c.numResults = numResults
	return c
}

func (c *Classifier[T]) WithThreshold(threshold float32) *Classifier[T] {
	c.threshold = threshold
	return c
}

func (c *Classifier[T]) WithFallbackIntent(intent T) *Classifier[T] {
	c.fallbackIntent = intent
	return c
}

func (c *Classifier[T]) Classify(text string) (ClassifierResult[T], error) {
	if c.store.Len() == 0 {
		return ClassifierResult[T]{}, fmt.Errorf("store is empty")
	}

	vector, err := c.embedder.Embed(text)
	if err != nil {
		return ClassifierResult[T]{}, fmt.Errorf("embedding failed: %w", err)
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
		return ClassifierResult[T]{
			Intent:     c.fallbackIntent,
			Confidence: 0,
			TopMatches: matches,
		}, nil
	}

	return ClassifierResult[T]{
		Intent:     T(winner),
		Confidence: topScore,
		TopMatches: matches,
	}, nil
}
