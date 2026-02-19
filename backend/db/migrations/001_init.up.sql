-- 001_init.up.sql

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS rooms (
    id VARCHAR(36) PRIMARY KEY,
    created_by VARCHAR(36) NOT NULL,
    dm_user_id VARCHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'lobby',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id),
    FOREIGN KEY (dm_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS room_members (
    room_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    role ENUM('player','dm') NOT NULL DEFAULT 'player',
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_id, user_id),
    FOREIGN KEY (room_id) REFERENCES rooms(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS room_sequences (
    room_id VARCHAR(36) PRIMARY KEY,
    next_seq BIGINT NOT NULL DEFAULT 1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS commands_dedup (
    room_id VARCHAR(36) NOT NULL,
    actor_user_id VARCHAR(36) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    command_type VARCHAR(64) NOT NULL,
    command_id VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    result_json TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_id, actor_user_id, idempotency_key, command_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS events (
    room_id VARCHAR(36) NOT NULL,
    seq BIGINT NOT NULL,
    event_id VARCHAR(36) NOT NULL UNIQUE,
    event_type VARCHAR(64) NOT NULL,
    actor_user_id VARCHAR(36) NOT NULL,
    causation_command_id VARCHAR(255),
    payload_json TEXT,
    server_ts TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_id, seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_events_room_seq ON events(room_id, seq);

CREATE TABLE IF NOT EXISTS snapshots (
    room_id VARCHAR(36) NOT NULL,
    last_seq BIGINT NOT NULL,
    state_json MEDIUMTEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_id, last_seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS agent_runs (
    id VARCHAR(36) PRIMARY KEY,
    room_id VARCHAR(36) NOT NULL,
    seq_from BIGINT NOT NULL,
    seq_to BIGINT NOT NULL,
    agent_name VARCHAR(64) NOT NULL,
    viewer_user_id VARCHAR(36),
    input_digest VARCHAR(64),
    output_digest VARCHAR(64),
    status VARCHAR(32) NOT NULL,
    latency_ms BIGINT,
    error_text TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
