CREATE TABLE stats_cache (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    driver_id BIGINT NOT NULL,
    stat_type VARCHAR(50) NOT NULL,
    stat_date DATE NOT NULL,
    total_earned DECIMAL(10,2) DEFAULT 0,
    total_trips INT DEFAULT 0,
    avg_fare DECIMAL(10,2) DEFAULT 0,
    total_distance DECIMAL(10,2) DEFAULT 0,
    earnings_per_hour DECIMAL(10,2) DEFAULT 0,
    active_hours DECIMAL(10,2) DEFAULT 0,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE KEY unique_stat (driver_id, stat_type, stat_date),
    INDEX idx_driver_type_date (driver_id, stat_type, stat_date),
    INDEX idx_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
