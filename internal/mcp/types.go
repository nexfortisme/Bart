package mcp

import "github.com/nexfortisme/bart/internal/classifier"

type MCPTool string 

const (
    MCPToolGetTime      MCPTool = "get_time"
    MCPToolGetWeather   MCPTool = "get_weather"
    MCPToolSearch       MCPTool = "web_search"
    MCPToolFetchURL     MCPTool = "fetch_url"
    MCPToolFetchURLs    MCPTool = "fetch_urls"
)

var IntentToMCPTool = map[classifier.ToolIntent][]MCPTool{
    classifier.ToolIntentTime:      {MCPToolGetTime},
    classifier.ToolIntentWeather:   {MCPToolGetWeather},
	classifier.ToolIntentWebFetch:  {MCPToolFetchURL, MCPToolFetchURLs},
    classifier.ToolIntentWebSearch: {MCPToolSearch, MCPToolFetchURL, MCPToolFetchURLs},
    classifier.ToolIntentNull:      {},
}