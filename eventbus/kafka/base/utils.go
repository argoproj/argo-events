package base

import (
	"fmt"
)

func EventKey(source string, subject string) string {
	return fmt.Sprintf("%s.%s", source, subject)
}
