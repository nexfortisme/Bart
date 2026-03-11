package tools

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
)

func GetWeather(args map[string]any) (string, error) {
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