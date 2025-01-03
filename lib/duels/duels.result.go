package duels

import (
	"backend/lib/services"

	basepool "github.com/ciphrpool/base-pool/gen"
)

type Summary struct {
	EgoCount      int `json:"ego_count"`
	Energy        int `json:"energy"`
	CorruptedData int `json:"corrupted_data"`
	EmotionalData int `json:"emotional_data"`
	QuantumData   int `json:"quantum_data"`
	LogicalData   int `json:"logical_data"`
}

type PID string

const (
	P1      PID = "p1"
	P2      PID = "p2"
	Default PID = "default"
)

type Outcome struct {
	Winner   PID                    `json:"winner"`
	Method   basepool.WinningMethod `json:"method"`
	Duration int64                  `json:"duration"`
}

type DuelResult struct {
	P1Summary   Summary                  `json:"p1_summary"`
	P2Summary   Summary                  `json:"p2_summary"`
	Outcome     Outcome                  `json:"outcome"`
	SessionData services.DuelSessionData `json:"session_data"`
	SessionID   string                   `json:"session_id"`
}
