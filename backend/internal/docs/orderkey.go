package docs

import (
	"fmt"
	"strings"
)

// Fractional indexing keys using base62 digits after a fixed prefix.
const orderAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func InitialOrderKey() string {
	return "a0"
}

func OrderKeyAfter(prev string) string {
	if prev == "" {
		return InitialOrderKey()
	}
	return incrementOrderKey(prev)
}

func OrderKeyBefore(next string) string {
	if next == "" {
		return InitialOrderKey()
	}
	// Produce a key lexicographically before next by midpoint with empty left.
	return midpointOrderKey("", next)
}

func OrderKeyBetween(prev, next string) (string, error) {
	if prev != "" && next != "" && prev >= next {
		return "", fmt.Errorf("invalid order bounds: prev=%q next=%q", prev, next)
	}
	if prev == "" && next == "" {
		return InitialOrderKey(), nil
	}
	if next == "" {
		return OrderKeyAfter(prev), nil
	}
	if prev == "" {
		return OrderKeyBefore(next), nil
	}
	mid := midpointOrderKey(prev, next)
	if mid == "" || mid == prev || mid == next {
		return "", fmt.Errorf("cannot allocate order key between %q and %q", prev, next)
	}
	return mid, nil
}

func midpointOrderKey(a, b string) string {
	if b == "" {
		return incrementOrderKey(a)
	}
	i := 0
	var out strings.Builder
	for {
		digitA := 0
		if i < len(a) {
			digitA = indexOfOrder(a[i]) + 1
		}
		digitB := len(orderAlphabet) + 1
		if i < len(b) {
			digitB = indexOfOrder(b[i]) + 1
		}
		if digitA == -1 {
			digitA = 0
		}
		if i < len(b) && indexOfOrder(b[i]) < 0 {
			digitB = len(orderAlphabet) + 1
		}
		if digitB-digitA > 1 {
			mid := digitA + (digitB-digitA)/2
			out.WriteByte(orderAlphabet[mid-1])
			return out.String()
		}
		if digitA > 0 {
			out.WriteByte(orderAlphabet[digitA-1])
		} else {
			out.WriteByte(orderAlphabet[0])
		}
		i++
		if i > 64 {
			return out.String() + "0"
		}
	}
}

func incrementOrderKey(key string) string {
	if key == "" {
		return InitialOrderKey()
	}
	chars := []byte(key)
	for i := len(chars) - 1; i >= 0; i-- {
		idx := indexOfOrder(chars[i])
		if idx < 0 {
			continue
		}
		if idx+1 < len(orderAlphabet) {
			chars[i] = orderAlphabet[idx+1]
			return string(chars)
		}
		chars[i] = orderAlphabet[0]
	}
	return key + "0"
}

func indexOfOrder(b byte) int {
	for i := 0; i < len(orderAlphabet); i++ {
		if orderAlphabet[i] == b {
			return i
		}
	}
	return -1
}
