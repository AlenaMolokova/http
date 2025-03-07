package models

import "encoding/json"

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func (r ShortenResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Result string `json:"result"`
	}{
		Result: r.Result,
	})
}

func (r *ShortenRequest) UnmarshalJSON(data []byte) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	r.URL = req.URL
	return nil
}
