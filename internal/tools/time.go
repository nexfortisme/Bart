package tools

import (
	"fmt"
	"time"
)

func GetTime(_ map[string]any) (string, error) {
    return fmt.Sprintf(`{"time": "%s"}`, time.Now().Format(time.RFC3339)), nil
}