package testutil

import (
	"fmt"
)

func mirrorOf(s string) string {
	return fmt.Sprintf("ghcr.io/stargz-containers/%s-org", s)
}

var (
	AlpineImage = mirrorOf("alpine:3.13")

	CommonImage = AlpineImage
)
