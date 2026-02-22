package processor

import "encoding/json"

type HistoryMessage struct {
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
	Text   string `json:"text"`
	Seq    string `json:"seq"`
	Ts     int64  `json:"ts_ms"`
}

func (m HistoryMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type DLQMessage struct {
	HistoryMessage
	Error string `json:"error"`
	Retry int    `json:"retry_count"`
}
