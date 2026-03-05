// Package room 配置结构体：RoomActor 和 RoomManager 的初始化参数
//
// [OUT] room.go（构造函数参数）
// [POS] 减少 NewRoomActor/NewRoomManager 参数数量 (≤4)
package room

import (
	"context"

	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/observability"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/store"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// BotEventNotifier allows the room to notify bots about events
// without directly importing the bot package.
type BotEventNotifier interface {
	OnEvent(ctx context.Context, roomID string, ev types.Event)
}

// RoomDeps holds shared dependencies for creating RoomActors.
type RoomDeps struct {
	Store            *store.Store
	Logger           *zap.Logger
	Metrics          *observability.Metrics
	SnapshotInterval int64
	AutoDM           *agent.AutoDM
	Composer         game.Composer
	BotNotifier      BotEventNotifier
}
