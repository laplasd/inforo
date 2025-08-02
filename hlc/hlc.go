package hlc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type HLC struct {
	Physical uint64
	Logical  uint64
	nodeID   string
	mu       sync.Mutex
}

func NewHLC() *HLC {
	return &HLC{
		nodeID: GenerateNodeID(),
	}
}

func (h *HLC) Now() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := uint64(time.Now().UnixNano())

	if now > h.Physical {
		h.Physical = now
		h.Logical = 0
	} else {
		h.Logical++
	}

	return time.Unix(0, int64(h.Physical))
}

func (h *HLC) Update(external time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()

	extPhysical := uint64(external.UnixNano())

	switch {
	case extPhysical > h.Physical:
		h.Physical = extPhysical
		h.Logical = 0
	case extPhysical == h.Physical:
		h.Logical++
	default:
		// Ничего не делаем, локальное время новее
	}
}

var counter uint64

func GenerateNodeID() string {
	// Часть 1: Уникальный идентификатор хоста
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	// Часть 2: Случайные байты
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes) // Игнорируем ошибку, если крипто-генератор недоступен

	// Часть 3: Атомарный счетчик
	seq := atomic.AddUint64(&counter, 1)

	return hex.EncodeToString([]byte(hostname)) + "-" +
		hex.EncodeToString(randomBytes) + "-" +
		fmt.Sprintf("%04d", seq)
}
