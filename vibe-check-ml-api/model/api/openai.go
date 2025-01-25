package model

type OpenAICompletionsRequest struct {
	Model             string  `json:"model"`
	Prompt            string  `json:"prompt"`
	Max_tokens        int     `json:"max_tokens"`
	Temperature       float32 `json:"temperature"`
	Top_p             float32 `json:"top_p"`
	Frequency_penalty float32 `json:"frequency_penalty"` // -2.0 - 2.0
	Presence_penalty  float32 `json:"presence_penalty"`  // -2.0 - 2.0
}

type OpenAICompletionsResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		Logprobs     string `json:"logprobs"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type OpenAIChatCompletionsRequest struct {
	Model      string  `json:"model"`
	Top_p      float64 `json:"top_p"`
	Max_tokens int     `json:"max_tokens"`
	Messages   []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type OpenAIChatCompletionsResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
