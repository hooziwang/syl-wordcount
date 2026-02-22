package output

import (
	"encoding/json"
	"fmt"
	"io"
)

func Write(w io.Writer, format string, events []map[string]any) error {
	switch format {
	case "ndjson":
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		for _, e := range events {
			if err := enc.Encode(e); err != nil {
				return err
			}
		}
		return nil
	case "json":
		obj := map[string]any{"events": events}
		b, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	default:
		return fmt.Errorf("不支持的输出格式：%s", format)
	}
}
