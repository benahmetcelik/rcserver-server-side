package security

import (
	"strings"
)

var blockedFragments = []string{
	">/dev/sd", "mkfs", "dd if=", ":/dev/", ":(){", "chmod -R 777 /",
	"rm -rf /", "rm -rf /*", ":(){:|:&};:",
}

func CommandAllowed(cmd string) bool {
	c := strings.TrimSpace(strings.ToLower(cmd))
	for _, b := range blockedFragments {
		if strings.Contains(c, b) {
			return false
		}
	}
	return true
}

func ArgAllowed(args []string) bool {
	joined := strings.ToLower(strings.Join(args, " "))
	for _, b := range blockedFragments {
		if strings.Contains(joined, b) {
			return false
		}
	}
	return true
}
