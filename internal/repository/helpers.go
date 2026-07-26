package repository

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// uniqueChildName returns a name that doesn't collide with an existing child
// under the same parent, appending " (2)", " (3)", … and falling back to a short
// random suffix after 10k attempts. When splitExt is true the numeric suffix is
// inserted before a file extension ("photo.png" → "photo (2).png"); folders pass
// false. count reports how many existing children already carry a given name.
func uniqueChildName(name string, splitExt bool, count func(name string) (int64, error)) (string, error) {
	n, err := count(name)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return name, nil
	}

	base, ext := name, ""
	if splitExt {
		if idx := strings.LastIndex(name, "."); idx > 0 {
			base, ext = name[:idx], name[idx:]
		}
	}

	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		n, err := count(candidate)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return candidate, nil
		}
	}
	return base + "-" + uuid.New().String()[:8] + ext, nil
}
