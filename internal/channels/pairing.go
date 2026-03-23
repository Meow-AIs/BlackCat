package channels

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// PairingManager handles code-based user authentication for channels.
type PairingManager struct {
	mu    sync.Mutex
	codes map[string]*pairingCode // code → user info
}

type pairingCode struct {
	Code      string
	Platform  Platform
	UserID    string
	ExpiresAt int64
}

// NewPairingManager creates a pairing manager.
func NewPairingManager() *PairingManager {
	return &PairingManager{codes: make(map[string]*pairingCode)}
}

// GenerateCode creates a 6-digit pairing code for a platform user.
func (pm *PairingManager) GenerateCode(platform Platform, userID string) string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	code := generateSecureCode()
	pm.codes[code] = &pairingCode{
		Code:      code,
		Platform:  platform,
		UserID:    userID,
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
	}
	return code
}

// ValidateCode checks if a pairing code is valid and returns the user info.
func (pm *PairingManager) ValidateCode(code string) (Platform, string, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pc, ok := pm.codes[code]
	if !ok {
		return "", "", false
	}

	if time.Now().Unix() > pc.ExpiresAt {
		delete(pm.codes, code)
		return "", "", false
	}

	// One-time use
	delete(pm.codes, code)
	return pc.Platform, pc.UserID, true
}

// CleanExpired removes expired pairing codes.
func (pm *PairingManager) CleanExpired() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now().Unix()
	count := 0
	for code, pc := range pm.codes {
		if now > pc.ExpiresAt {
			delete(pm.codes, code)
			count++
		}
	}
	return count
}

func generateSecureCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	num := (int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])) % 1000000
	if num < 0 {
		num = -num
	}
	return fmt.Sprintf("%06d", num)
}
