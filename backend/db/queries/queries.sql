-- name: GetUser :one
SELECT id, email, password_hash, created_at FROM users WHERE id = ? LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, created_at FROM users WHERE email = ? LIMIT 1;

-- name: CreateUser :exec
INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?);

-- name: GetRoom :one
SELECT id, created_by, dm_user_id, status, created_at FROM rooms WHERE id = ? LIMIT 1;

-- name: CreateRoom :exec
INSERT INTO rooms (id, created_by, dm_user_id, status, created_at) VALUES (?, ?, ?, ?, ?);

-- name: AddRoomMember :exec
INSERT INTO room_members (room_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE role = VALUES(role);

-- name: GetRoomMembers :many
SELECT room_id, user_id, role, joined_at FROM room_members WHERE room_id = ?;

-- name: IsMember :one
SELECT role FROM room_members WHERE room_id = ? AND user_id = ? LIMIT 1;

-- name: GetNextSeq :one
SELECT next_seq FROM room_sequences WHERE room_id = ? FOR UPDATE;

-- name: UpsertRoomSeq :exec
INSERT INTO room_sequences (room_id, next_seq) VALUES (?, ?) ON DUPLICATE KEY UPDATE next_seq = VALUES(next_seq);

-- name: InsertEvent :exec
INSERT INTO events (room_id, seq, event_id, event_type, actor_user_id, causation_command_id, payload_json, server_ts)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: LoadEventsAfter :many
SELECT room_id, seq, event_id, event_type, actor_user_id, causation_command_id, payload_json, server_ts
FROM events WHERE room_id = ? AND seq > ? ORDER BY seq ASC LIMIT ?;

-- name: GetDedupRecord :one
SELECT room_id, actor_user_id, idempotency_key, command_type, command_id, status, result_json, created_at
FROM commands_dedup WHERE room_id = ? AND actor_user_id = ? AND idempotency_key = ? AND command_type = ? LIMIT 1;

-- name: SaveDedupRecord :exec
INSERT INTO commands_dedup (room_id, actor_user_id, idempotency_key, command_type, command_id, status, result_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE status = VALUES(status), result_json = VALUES(result_json);

-- name: GetLatestSnapshot :one
SELECT room_id, last_seq, state_json, created_at FROM snapshots WHERE room_id = ? ORDER BY last_seq DESC LIMIT 1;

-- name: SaveSnapshot :exec
INSERT INTO snapshots (room_id, last_seq, state_json, created_at) VALUES (?, ?, ?, ?);

-- name: InsertAgentRun :exec
INSERT INTO agent_runs (id, room_id, seq_from, seq_to, agent_name, viewer_user_id, input_digest, output_digest, status, latency_ms, error_text, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
