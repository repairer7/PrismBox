package subscription

import (
	"bytes"
	"fmt"

	"github.com/prismbox/prismbox/internal/config"
)

// ParseBase64 parses base64-encoded subscription content into Outbound items.
// If the content is already plain text with proxy links, it parses lines directly.
func ParseBase64(data []byte) ([]config.Outbound, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty subscription data")
	}

	// Remove UTF-8 BOM if present
	trimmed = bytes.TrimPrefix(trimmed, []byte("\xef\xbb\xbf"))

	var content string

	// Attempt base64 decode
	decoded, err := decodeBase64Flexible(string(trimmed))
	if err == nil && len(decoded) > 0 {
		content = string(decoded)
	} else {
		// Might already be plain text links
		content = string(trimmed)
	}

	outbounds, _ := ParseMultipleLinks(content)
	if len(outbounds) == 0 {
		return nil, fmt.Errorf("no valid proxy nodes found in base64 subscription")
	}

	return outbounds, nil
}
