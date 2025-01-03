package duels

import (
	"backend/lib/services"
	"context"
	"fmt"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
)

func calculateEloChanges(result *DuelResult) (p1_delta, p2_delta int) {
	base_points := 32
	if result.Outcome.Winner == P1 {
		return base_points, -base_points
	}
	return -base_points, base_points
}

func FriendlyDuelResultProcessor(ctx context.Context, result *DuelResult, cache *services.Cache, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	queries := basepool.New(db.Pool)

	defer cancel()
	// Start transaction
	tx, err := db.Pool.Begin(query_ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(query_ctx)

	var duel_outcome basepool.DuelOutcome
	if result.Outcome.Winner == P1 {
		duel_outcome = basepool.DuelOutcomeP1WON
	} else if result.Outcome.Winner == P2 {
		duel_outcome = basepool.DuelOutcomeP2WON
	} else {
		duel_outcome = basepool.DuelOutcomeDraw
	}

	sessionID, err := services.StringToUUID(result.SessionID)
	if err != nil {
		return fmt.Errorf("failed to conevrt session id: %w", err)
	}

	qtx := queries.WithTx(tx)
	err = qtx.InsertDuelResult(query_ctx, basepool.InsertDuelResultParams{
		SessionID:       sessionID,
		P1ID:            result.SessionData.P1.PID,
		P2ID:            result.SessionData.P2.PID,
		DuelOutcome:     duel_outcome,
		DuelType:        result.SessionData.DuelType,
		WinningMethod:   result.Outcome.Method,
		P1EloDelta:      int32(0),
		P2EloDelta:      int32(0),
		Duration:        int32(result.Outcome.Duration),
		P1EgoCount:      int32(result.P1Summary.EgoCount),
		P1Energy:        int32(result.P1Summary.Energy),
		P1CorruptedData: int32(result.P1Summary.CorruptedData),
		P1EmotionalData: int32(result.P1Summary.EmotionalData),
		P1QuantumData:   int32(result.P1Summary.QuantumData),
		P1LogicalData:   int32(result.P1Summary.LogicalData),
		P2EgoCount:      int32(result.P2Summary.EgoCount),
		P2Energy:        int32(result.P2Summary.Energy),
		P2CorruptedData: int32(result.P2Summary.CorruptedData),
		P2EmotionalData: int32(result.P2Summary.EmotionalData),
		P2QuantumData:   int32(result.P2Summary.QuantumData),
		P2LogicalData:   int32(result.P2Summary.LogicalData),
	})
	if err != nil {
		return fmt.Errorf("failed to store the result of the duel: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(query_ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func RankedDuelResultProcessor(ctx context.Context, result *DuelResult, cache *services.Cache, db *services.Database) error {
	// err = qtx.UpdatePlayerElo(query_ctx, basepool.UpdatePlayerEloParams{
	// 	EloDelta: int32(p1_elo_delta),
	// 	UserID:   result.SessionData.P1.PID,
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to update p1 elo: %w", err)
	// }

	// err = qtx.UpdatePlayerElo(query_ctx, basepool.UpdatePlayerEloParams{
	// 	EloDelta: int32(p2_elo_delta),
	// 	UserID:   result.SessionData.P2.PID,
	// })
	return nil
}

func TournamentDuelResultProcessor(ctx context.Context, result *DuelResult, cache *services.Cache, db *services.Database) error {
	return nil
}
