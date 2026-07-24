package util

import (
	"encoding/json"
	"strconv"
)

// DecodePayload перегоняет event.Envelope.Payload (any, после JSON-анмаршалинга это map[string]any)
// в конкретную типизированную структуру через промежуточную сериализацию.
func DecodePayload(raw any, target any) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

// Itoa int64 -> string (strconv)
func Itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Ftoa float64 -> string (strconv)
func Ftoa(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
