package classifier

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

func NewStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Add(id, text, intent string, vector []float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, StoreEntry{
		ID:     id,
		Text:   text,
		Intent: intent,
		Vector: vector,
	})
}

func (s *MemoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// TODO: Replace with SQLite
func (s *MemoryStore) Save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	persisted := make([]StoreEntry, len(s.entries))
	for i, e := range s.entries {
		persisted[i] = StoreEntry{
			ID:     e.ID,
			Text:   e.Text,
			Intent: e.Intent,
			Vector: e.Vector,
		}
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal store: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// TODO: Replace with SQLite
func (s *MemoryStore) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read store file: %w", err)
	}

	var persisted []StoreEntry
	if err := json.Unmarshal(data, &persisted); err != nil {
		return fmt.Errorf("unmarshal store: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make([]StoreEntry, len(persisted))
	for i, p := range persisted {
		s.entries[i] = StoreEntry{
			ID:     p.ID,
			Text:   p.Text,
			Intent: p.Intent,
			Vector: p.Vector,
		}
	}

	return nil
}

func (s *MemoryStore) Query(vector []float32, numResults int) []QueryResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]QueryResult, 0, len(s.entries))
	for _, entry := range s.entries {
		sim := cosineSimilarity(vector, entry.Vector)
		results = append(results, QueryResult{Entry: entry, Similarity: sim})
	}

	// Sort descending by similarity
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if numResults > len(results) {
		numResults = len(results)
	}
	return results[:numResults]
}

// -- Math that is beyond me (Thanks Claude) -- 
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
