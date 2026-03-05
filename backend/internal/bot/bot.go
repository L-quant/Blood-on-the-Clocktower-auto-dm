// Package bot 单个 Bot 玩家逻辑，性格驱动的自动决策
//
// [IN]  internal/types（Event 与 CommandEnvelope 类型）
// [OUT] api（Bot 管理端点）
// [POS] Bot 个体行为实现，根据性格参数自动参与游戏

package bot

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// Personality defines a bot's decision-making style.
type Personality string

const (
	PersonalityAggressive Personality = "aggressive" // Nominates often, votes yes frequently
	PersonalityCautious   Personality = "cautious"   // Rarely nominates, careful voter
	PersonalityRandom     Personality = "random"     // 50/50 on most decisions
	PersonalitySmart      Personality = "smart"      // Uses role info to make better decisions
)

// BotConfig configures a bot player.
type BotConfig struct {
	UserID      string
	Name        string
	Personality Personality
	Logger      *slog.Logger
}

// Bot represents a bot player in a game.
type Bot struct {
	mu          sync.RWMutex
	userID      string
	name        string
	personality Personality
	logger      *slog.Logger
	dispatcher  CommandDispatcher
	roomID      string

	// Game knowledge
	role      string
	trueRole  string
	team      string
	alive     bool
	demonID   string
	teammates []string
	bluffs    []string
	phase     string
	dayCount  int
	hasVoted  bool

	// Current nomination context (stored on nomination.created, used on defense.ended)
	lastNominee   string
	lastVoteOrder []string // sequential user_id order from vote_order seats
}

// CommandDispatcher sends commands to the game engine.
type CommandDispatcher interface {
	DispatchAsync(cmd types.CommandEnvelope) error
}

// NewBot creates a new bot player.
func NewBot(cfg BotConfig) *Bot {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Personality == "" {
		cfg.Personality = PersonalityRandom
	}
	return &Bot{
		userID:      cfg.UserID,
		name:        cfg.Name,
		personality: cfg.Personality,
		logger:      cfg.Logger,
		alive:       true,
	}
}

// UserID returns the bot's user ID.
func (b *Bot) UserID() string { return b.userID }

// Name returns the bot's display name.
func (b *Bot) Name() string { return b.name }

// SetDispatcher sets the command dispatcher for the bot.
func (b *Bot) SetDispatcher(d CommandDispatcher, roomID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dispatcher = d
	b.roomID = roomID
}

// OnEvent processes a game event and may take action in response.
func (b *Bot) OnEvent(ctx context.Context, ev types.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var payload map[string]string
	_ = json.Unmarshal(ev.Payload, &payload)
	if payload == nil {
		payload = map[string]string{}
	}

	switch ev.EventType {
	case "role.assigned":
		if payload["user_id"] == b.userID {
			b.role = payload["role"]
			b.trueRole = payload["true_role"]
			b.team = payload["team"]
			b.logger.Info("bot assigned role", "bot", b.name, "role", b.role, "team", b.team)
		}

	case "bluffs.assigned":
		if b.trueRole == "imp" {
			var bluffs []string
			_ = json.Unmarshal([]byte(payload["bluffs"]), &bluffs)
			b.bluffs = bluffs
		}

	case "phase.day":
		b.phase = "day"
		b.dayCount++
		b.hasVoted = false
		// Maybe chat after a delay
		go b.maybeChatAfterDelay(ctx)

	case "phase.nomination":
		b.phase = "nomination"
		// Maybe nominate after a delay
		go b.maybeNominateAfterDelay(ctx)

	case "nomination.created":
		// Store nominee for later voting (defense phase must end first)
		b.lastNominee = payload["nominee"]

	case "defense.ended":
		// Now voting phase starts — try to vote after delay
		nominee := b.lastNominee
		go b.maybeVoteAfterDelay(ctx, nominee)

	case "vote.cast":
		// Track our own vote result from server confirmation
		if payload["voter_user_id"] == b.userID || ev.ActorUserID == b.userID {
			b.hasVoted = true
		}

	case "phase.night", "phase.first_night":
		b.phase = "night"
		// Bots now wait for night.action.prompt to act sequentially

	case "night.action.prompt":
		if payload["user_id"] == b.userID {
			go b.handleNightActionPrompt(ctx, payload)
		}

	case "player.died":
		if payload["user_id"] == b.userID {
			b.alive = false
			b.logger.Info("bot died", "bot", b.name, "cause", payload["cause"])
		}

	case "game.ended":
		b.logger.Info("game ended", "bot", b.name, "winner", payload["winner"])
	}
}

func (b *Bot) handleNightActionPrompt(ctx context.Context, payload map[string]string) {
	delay := randomDuration(1000, 3000)
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return
	}

	b.mu.RLock()
	dispatcher := b.dispatcher
	roomID := b.roomID
	b.mu.RUnlock()

	if dispatcher == nil {
		return
	}

	actionType := payload["action_type"]
	targetsJSON := "[]"
	// For select actions, bot picks random targets (empty targets = skip)
	if actionType == "select_one" || actionType == "select_two" {
		targetsJSON = "[]" // Bot skips target selection; engine handles no_action fallback
	}

	cmdPayload, _ := json.Marshal(map[string]string{
		"targets": targetsJSON,
	})
	err := dispatcher.DispatchAsync(types.CommandEnvelope{
		CommandID:   fmt.Sprintf("bot-%s-ability-%d", b.userID, time.Now().UnixMilli()),
		RoomID:      roomID,
		Type:        "ability.use",
		ActorUserID: b.userID,
		Payload:     cmdPayload,
	})
	if err != nil {
		b.logger.Warn("bot night action failed", "bot", b.name, "error", err)
	}
}

func (b *Bot) maybeChatAfterDelay(ctx context.Context) {
	delay := randomDuration(2000, 5000)
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return
	}

	b.mu.RLock()
	alive := b.alive
	dispatcher := b.dispatcher
	roomID := b.roomID
	b.mu.RUnlock()

	if !alive || dispatcher == nil {
		return
	}

	msg := b.generateChat()
	if msg == "" {
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"message": msg,
		"from":    b.name,
	})
	_ = dispatcher.DispatchAsync(types.CommandEnvelope{
		CommandID:   fmt.Sprintf("bot-%s-%d", b.userID, time.Now().UnixMilli()),
		RoomID:      roomID,
		Type:        "public_chat",
		ActorUserID: b.userID,
		Payload:     payload,
	})
}

func (b *Bot) maybeNominateAfterDelay(ctx context.Context) {
	delay := randomDuration(3000, 8000)
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return
	}

	b.mu.RLock()
	alive := b.alive
	dispatcher := b.dispatcher
	personality := b.personality
	b.mu.RUnlock()

	if !alive || dispatcher == nil {
		return
	}

	// Decide whether to nominate based on personality
	shouldNominate := false
	switch personality {
	case PersonalityAggressive:
		shouldNominate = randomChance(70)
	case PersonalityCautious:
		shouldNominate = randomChance(20)
	case PersonalityRandom:
		shouldNominate = randomChance(40)
	case PersonalitySmart:
		shouldNominate = randomChance(50)
	}

	if !shouldNominate {
		return
	}

	// Bot doesn't know other players' IDs directly - nomination needs a target
	// This will be handled by the bot manager which has game state access
	b.logger.Debug("bot wants to nominate", "bot", b.name)
}

func (b *Bot) maybeVoteAfterDelay(ctx context.Context, nominee string) {
	// In sequential voting, bots may need to wait for their turn.
	// Retry up to 60 seconds (defense already ended at this point).
	maxRetries := 20
	for attempt := 0; attempt < maxRetries; attempt++ {
		delay := randomDuration(1000, 3000)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}

		b.mu.RLock()
		alive := b.alive
		dispatcher := b.dispatcher
		roomID := b.roomID
		team := b.team
		personality := b.personality
		hasVoted := b.hasVoted
		b.mu.RUnlock()

		if !alive || dispatcher == nil || hasVoted {
			return
		}

		// Decide vote based on personality and team
		voteYes := b.decideVote(personality, team, nominee)

		voteStr := "no"
		if voteYes {
			voteStr = "yes"
		}

		payload, _ := json.Marshal(map[string]string{
			"vote": voteStr,
		})
		err := dispatcher.DispatchAsync(types.CommandEnvelope{
			CommandID:   fmt.Sprintf("bot-%s-vote-%d", b.userID, time.Now().UnixMilli()),
			RoomID:      roomID,
			Type:        "vote",
			ActorUserID: b.userID,
			Payload:     payload,
		})

		if err == nil {
			// Vote accepted — hasVoted will be set by vote.cast event handler
			return
		}
		// Vote rejected (probably not our turn yet in sequential voting) — retry
		b.logger.Debug("bot vote rejected, will retry",
			"bot", b.name, "attempt", attempt+1, "error", err)
	}
}

func (b *Bot) decideVote(personality Personality, team, nominee string) bool {
	switch personality {
	case PersonalityAggressive:
		return randomChance(65)
	case PersonalityCautious:
		return randomChance(30)
	case PersonalityRandom:
		return randomChance(50)
	case PersonalitySmart:
		if team == "evil" && nominee == b.demonID {
			return false
		} else if team == "evil" {
			return randomChance(60)
		}
		return randomChance(45)
	default:
		return randomChance(50)
	}
}

func (b *Bot) generateChat() string {
	b.mu.RLock()
	personality := b.personality
	team := b.team
	dayCount := b.dayCount
	b.mu.RUnlock()

	if dayCount <= 1 {
		// First day: introductions
		msgs := []string{
			fmt.Sprintf("大家好，我是%s。", b.name),
			fmt.Sprintf("你们好！我是%s，请多关照。", b.name),
			fmt.Sprintf("我是%s，我们来找出恶魔吧！", b.name),
		}
		return msgs[randomInt(len(msgs))]
	}

	switch personality {
	case PersonalityAggressive:
		if team == "evil" {
			msgs := []string{
				"我觉得有人在说谎！",
				"我们得赶快投票处决可疑的人。",
				"信息对不上，一定有人是邪恶的！",
			}
			return msgs[randomInt(len(msgs))]
		}
		msgs := []string{
			"我们需要更果断地行动！",
			"赶快提名投票吧！",
			"不能再犹豫了！",
		}
		return msgs[randomInt(len(msgs))]
	case PersonalityCautious:
		msgs := []string{
			"我们还需要更多信息再做决定。",
			"别急，先分析一下局势。",
			"大家冷静一下，仔细想想。",
		}
		return msgs[randomInt(len(msgs))]
	default:
		msgs := []string{
			"嗯...让我想想。",
			"有什么新的线索吗？",
			"大家怎么看？",
		}
		return msgs[randomInt(len(msgs))]
	}
}

// randomChance returns true with the given probability (0-100).
func randomChance(percent int) bool {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return false
	}
	return n.Int64() < int64(percent)
}

// randomInt returns a random int in [0, n).
func randomInt(n int) int {
	if n <= 0 {
		return 0
	}
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(nBig.Int64())
}

// randomDuration returns a random duration between min and max milliseconds.
func randomDuration(minMs, maxMs int) time.Duration {
	rangeMs := maxMs - minMs
	if rangeMs <= 0 {
		return time.Duration(minMs) * time.Millisecond
	}
	extra := randomInt(rangeMs)
	return time.Duration(minMs+extra) * time.Millisecond
}
