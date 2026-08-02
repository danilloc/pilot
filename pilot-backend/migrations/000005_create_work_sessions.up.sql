CREATE TABLE work_sessions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    driver_id BIGINT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    total_trips INT DEFAULT 0,
    total_earnings DECIMAL(10,2) DEFAULT 0,
    total_duration_minutes INT,
    daily_goal_id BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    FOREIGN KEY (daily_goal_id) REFERENCES daily_goals(id) ON DELETE SET NULL,
    INDEX idx_driver_id (driver_id),
    INDEX idx_started_at (started_at),
    INDEX idx_driver_started (driver_id, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
