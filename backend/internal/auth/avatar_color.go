package auth

import (
	"crypto/rand"
	"hash/fnv"
)

// Fixed palette (hex). Avoids purple/indigo; readable with white text.
var AvatarPalette = []string{
	"#0d9488", // teal-600
	"#059669", // emerald-600
	"#0284c7", // sky-600
	"#0891b2", // cyan-600
	"#d97706", // amber-600
	"#ea580c", // orange-600
	"#e11d48", // rose-600
	"#3f3f46", // zinc-700
}

func RandomAvatarColor() string {
	if len(AvatarPalette) == 0 {
		return "#3f3f46"
	}
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return AvatarPalette[0]
	}
	return AvatarPalette[int(b[0])%len(AvatarPalette)]
}

// AvatarColorForID returns a stable palette color from a user id (for legacy rows).
func AvatarColorForID(id string) string {
	if len(AvatarPalette) == 0 {
		return "#3f3f46"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return AvatarPalette[int(h.Sum32())%len(AvatarPalette)]
}

func ResolveAvatarColor(id, stored string) string {
	if stored != "" {
		return stored
	}
	return AvatarColorForID(id)
}
