package classifier

import "sync"

// Not needed to be in .env but is used across the package
var (
	MessageIntentStorePath = "resources/classifier/message_intent_store.json"
	ToolIntentStorePath    = "resources/classifier/tool_intent_store.json"
)

type IntentDataType string

const (
	IntentDataTypeMessage  IntentDataType = "message_intent"
	IntentDataTypeTool     IntentDataType = "tool_intent"
	IntentDataTypeEdgeCase IntentDataType = "edge_cases"
)

type MessageIntent string
type ToolIntent string

type IntentType interface {
	MessageIntent | ToolIntent
}

const (
	MessageIntentDirect    MessageIntent = "directed"
	MessageIntentAmbient   MessageIntent = "ambient"
	MessageIntentAmbiguous MessageIntent = "ambiguous"

	ToolIntentWeather   ToolIntent = "weather"
	ToolIntentTime      ToolIntent = "time"
	ToolIntentWebSearch ToolIntent = "web_search"
	ToolIntentWebFetch  ToolIntent = "web_fetch"
	ToolIntentNull      ToolIntent = "null"
)

// Store Structs
type StoreEntry struct {
	ID     string    `json:"id"`
	Text   string    `json:"text"`
	Intent string    `json:"intent"`
	Vector []float32 `json:"vector"`
}

type QueryResult struct {
	Entry      StoreEntry
	Similarity float32
}

type MemoryStore struct {
	mu      sync.RWMutex
	entries []StoreEntry
}

// Seeding Structs
type Example struct {
	Text   string `json:"text"`
	Intent string `json:"label"`
}

// Classifier Structs
type Classifier[T IntentType] struct {
	embedder       Embedder
	store          *MemoryStore
	numResults     int
	threshold      float32
	fallbackIntent T
}

type ClassifierResult[T IntentType] struct {
	Intent     T
	Confidence float32
	TopMatches []QueryResult
}

// Embedder is the interface any embedding backend must satisfy.
// Swap out LMStudio for OpenAI, Ollama, Cohere, etc. by implementing this.
type Embedder interface {
	Embed(text string) ([]float32, error)
}
