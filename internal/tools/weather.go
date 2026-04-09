package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool Interface Implementation
type WeatherTool struct {
	Tool
}

type WeatherInput struct {
	City string `json:"city" jsonschema:"the city to get weather for"`
}

func (t *WeatherTool) ToolInput() any {
	return WeatherInput{}
}

func (t *WeatherTool) ToolHandlerFn() func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
		input, ok := in.(WeatherInput)
		if !ok {
			return nil, nil, fmt.Errorf("invalid input type for get_weather")
		}
		result, err := getWeather(map[string]any{"city": input.City})
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil, nil
	}
}

func (t *WeatherTool) GetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_weather",
		Description: "Get current weather for a city",
        // InputSchema: t.ToolInput(),
	}
}

// Tool Utility Implementation
func getWeather(args map[string]any) (string, error) {
    city, ok := args["city"].(string)
    if !ok || city == "" {
        return "", fmt.Errorf("city argument required")
    }

    // Example: call Open-Meteo (free, no API key needed)
    geoURL := fmt.Sprintf(
        "https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1",
        url.QueryEscape(city),
    )
    geoResp, err := http.Get(geoURL)
    if err != nil {
        return "", err
    }
    defer geoResp.Body.Close()

    var geoData struct {
        Results []struct {
            Latitude  float64 `json:"latitude"`
            Longitude float64 `json:"longitude"`
        } `json:"results"`
    }
    body, _ := io.ReadAll(geoResp.Body)
    json.Unmarshal(body, &geoData)

    if len(geoData.Results) == 0 {
        return "", fmt.Errorf("city not found: %s", city)
    }

    lat := geoData.Results[0].Latitude
    lon := geoData.Results[0].Longitude

    weatherURL := fmt.Sprintf(
        "https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current_weather=true",
        lat, lon,
    )
    wResp, err := http.Get(weatherURL)
    if err != nil {
        return "", err
    }
    defer wResp.Body.Close()

    var weatherData struct {
        CurrentWeather struct {
            Temperature float64 `json:"temperature"`
            Windspeed   float64 `json:"windspeed"`
            Weathercode int     `json:"weathercode"`
        } `json:"current_weather"`
    }
    wBody, _ := io.ReadAll(wResp.Body)
    json.Unmarshal(wBody, &weatherData)

    result := fmt.Sprintf(
        `{"city": "%s", "temperature_c": %.1f, "windspeed_kmh": %.1f}`,
        city,
        weatherData.CurrentWeather.Temperature,
        weatherData.CurrentWeather.Windspeed,
    )
    return result, nil
}