//go:build !linux

package fdpass

import "fmt"

func Listen(ctrlPath string) (<-chan int, int, error) {
	return nil, 0, fmt.Errorf("fdpass not supported on this platform")
}
