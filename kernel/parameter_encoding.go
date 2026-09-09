package kernel

import (
	"fmt"
	"strings"
)

// URI expansion retains only reserved characters that cannot introduce query fields or fragments.
func parameterEscape(value string, allowReserved bool) string {
	var out strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		safe := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("-._~", rune(c))
		if allowReserved && c == '%' && i+2 < len(value) && isHex(value[i+1]) && isHex(value[i+2]) {
			out.WriteString(value[i : i+3])
			i += 2
			continue
		}
		if safe || (allowReserved && strings.ContainsRune(":/?@!$'()*;,", rune(c))) {
			out.WriteByte(c)
		} else {
			out.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return out.String()
}
func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
