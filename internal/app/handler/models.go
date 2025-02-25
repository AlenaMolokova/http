package handler

import "encoding/json"

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
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
