package bgen

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetBytes(p []byte) {
	for i := 0; i < len(p); i++ {
		p[i] = byte(randInt(65, 126))
	}
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
