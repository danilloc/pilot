CREATE TABLE daily_goals (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    driver_id BIGINT NOT NULL,
    goal_date DATE NOT NULL,
    goal_amount DECIMAL(10,2) NOT NULL,
    actual_amount DECIMAL(10,2) DEFAULT 0,
    status VARCHAR(50) DEFAULT 'PENDING',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    UNIQUE KEY unique_driver_date (driver_id, goal_date),
    INDEX idx_driver_id (driver_id),
    INDEX idx_goal_date (goal_date),
    INDEX idx_status (status),
    CONSTRAINT check_goal_positive CHECK (goal_amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
