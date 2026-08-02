# Pilot — Database Design

**Feature**: `database-schema`  
**Database**: MySQL 8.0+  
**Status**: 🟡 Ready  

---

## Schema Overview

```
drivers (1)
  ├─ trips (N)
  ├─ daily_goals (N)
  ├─ payment_records (N)
  └─ work_sessions (N)
```

---

## Table 1: drivers

```sql
CREATE TABLE drivers (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    uber_id VARCHAR(100) UNIQUE NOT NULL,
    
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    profile_picture_url TEXT,
    
    rating DECIMAL(3,2) DEFAULT 0.00,
    total_trips INT DEFAULT 0,
    account_status VARCHAR(50) DEFAULT 'ACTIVE',
    
    oauth_token TEXT,
    oauth_refresh_token TEXT,
    token_expires_at DATETIME,
    
    failed_login_attempts INT DEFAULT 0,
    last_login_ip VARCHAR(45),
    last_login_time DATETIME,
    is_banned BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_synced_at DATETIME,
    
    INDEX idx_uber_id (uber_id),
    INDEX idx_email (email),
    INDEX idx_account_status (account_status),
    INDEX idx_created_at (created_at),
    CONSTRAINT check_rating CHECK (rating >= 0 AND rating <= 5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## Table 2: trips

```sql
CREATE TABLE trips (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    driver_id BIGINT NOT NULL,
    uber_trip_id VARCHAR(100) UNIQUE NOT NULL,
    
    started_at DATETIME NOT NULL,
    ended_at DATETIME NOT NULL,
    duration_minutes INT GENERATED ALWAYS AS (TIMESTAMPDIFF(MINUTE, started_at, ended_at)) STORED,
    
    distance_km DECIMAL(10,2) NOT NULL,
    fare_value DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'BRL',
    
    fare_base DECIMAL(10,2) DEFAULT 0,
    fare_distance DECIMAL(10,2) DEFAULT 0,
    fare_time DECIMAL(10,2) DEFAULT 0,
    toll_charge DECIMAL(10,2) DEFAULT 0,
    service_fee DECIMAL(10,2) DEFAULT 0,
    
    city VARCHAR(100),
    start_latitude DECIMAL(10,8),
    start_longitude DECIMAL(11,8),
    end_latitude DECIMAL(10,8),
    end_longitude DECIMAL(11,8),
    
    status VARCHAR(50) DEFAULT 'COMPLETED',
    
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    INDEX idx_driver_id (driver_id),
    INDEX idx_ended_at (ended_at),
    INDEX idx_driver_ended (driver_id, ended_at),
    INDEX idx_status (status),
    FULLTEXT INDEX ft_city (city)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## Table 3: daily_goals

```sql
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
```

---

## Table 4: payment_records

```sql
CREATE TABLE payment_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    driver_id BIGINT NOT NULL,
    
    uber_payment_id VARCHAR(100) UNIQUE,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'BRL',
    
    tolls DECIMAL(10,2) DEFAULT 0,
    taxes DECIMAL(10,2) DEFAULT 0,
    net_amount DECIMAL(10,2) GENERATED ALWAYS AS (amount - tolls - taxes) STORED,
    
    payment_method VARCHAR(50),
    payment_date DATE NOT NULL,
    
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    INDEX idx_driver_id (driver_id),
    INDEX idx_payment_date (payment_date),
    INDEX idx_driver_date (driver_id, payment_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## Table 5: work_sessions

```sql
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
```

---

## Table 6: stats_cache

```sql
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
```

---

## Migrations

### 001_create_drivers.sql

```sql
CREATE TABLE drivers (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    uber_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    profile_picture_url TEXT,
    rating DECIMAL(3,2) DEFAULT 0.00,
    total_trips INT DEFAULT 0,
    account_status VARCHAR(50) DEFAULT 'ACTIVE',
    oauth_token TEXT,
    oauth_refresh_token TEXT,
    token_expires_at DATETIME,
    failed_login_attempts INT DEFAULT 0,
    last_login_ip VARCHAR(45),
    last_login_time DATETIME,
    is_banned BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_synced_at DATETIME,
    INDEX idx_uber_id (uber_id),
    INDEX idx_email (email),
    INDEX idx_account_status (account_status),
    INDEX idx_created_at (created_at),
    CONSTRAINT check_rating CHECK (rating >= 0 AND rating <= 5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 002_create_trips.sql

```sql
CREATE TABLE trips (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    driver_id BIGINT NOT NULL,
    uber_trip_id VARCHAR(100) UNIQUE NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME NOT NULL,
    duration_minutes INT GENERATED ALWAYS AS (TIMESTAMPDIFF(MINUTE, started_at, ended_at)) STORED,
    distance_km DECIMAL(10,2) NOT NULL,
    fare_value DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'BRL',
    fare_base DECIMAL(10,2) DEFAULT 0,
    fare_distance DECIMAL(10,2) DEFAULT 0,
    fare_time DECIMAL(10,2) DEFAULT 0,
    toll_charge DECIMAL(10,2) DEFAULT 0,
    service_fee DECIMAL(10,2) DEFAULT 0,
    city VARCHAR(100),
    start_latitude DECIMAL(10,8),
    start_longitude DECIMAL(11,8),
    end_latitude DECIMAL(10,8),
    end_longitude DECIMAL(11,8),
    status VARCHAR(50) DEFAULT 'COMPLETED',
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    INDEX idx_driver_id (driver_id),
    INDEX idx_ended_at (ended_at),
    INDEX idx_driver_ended (driver_id, ended_at),
    INDEX idx_status (status),
    FULLTEXT INDEX ft_city (city)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 003_create_daily_goals.sql

```sql
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
```

### 004_create_payment_records.sql

```sql
CREATE TABLE payment_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    driver_id BIGINT NOT NULL,
    uber_payment_id VARCHAR(100) UNIQUE,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'BRL',
    tolls DECIMAL(10,2) DEFAULT 0,
    taxes DECIMAL(10,2) DEFAULT 0,
    net_amount DECIMAL(10,2) GENERATED ALWAYS AS (amount - tolls - taxes) STORED,
    payment_method VARCHAR(50),
    payment_date DATE NOT NULL,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    INDEX idx_driver_id (driver_id),
    INDEX idx_payment_date (payment_date),
    INDEX idx_driver_date (driver_id, payment_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 005_create_work_sessions.sql

```sql
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
```

### 006_create_stats_cache.sql

```sql
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
```

---

## Índices Críticos

| Índice | Tabela | Colunas | Uso |
|--------|--------|---------|-----|
| idx_driver_ended | trips | (driver_id, ended_at) | Ganho por período |
| idx_driver_id | trips | driver_id | Listar trips |
| idx_driver_date | payment_records | (driver_id, payment_date) | Pagamentos |
| unique_driver_date | daily_goals | (driver_id, goal_date) | Uma meta/dia |
| idx_status | trips | status | Filtrar completadas |

---

## ⚠️ Nota sobre sargability (corrigido após validação real com 1M linhas)

As 3 queries abaixo foram testadas contra a tabela `trips` com ~1M linhas
(fatia `database`, T14). A versão original deste documento aplicava
`DATE(ended_at)` diretamente no `WHERE`/`JOIN`, o que impede o MySQL de
usar `idx_driver_ended` no critério de data (ele cai para `idx_driver_id`
sozinho) — um predicado "non-sargable". A performance final ficou dentro
do limite (60-100ms) mesmo assim, porque o volume por motorista é pequeno,
mas as queries abaixo já vêm corrigidas para usar o índice composto
completo, comparando por **intervalo de datas** em vez de transformar a
coluna.

## Queries Críticas

### Daily Earnings

```sql
SELECT 
    SUM(fare_value) as total_earned,
    COUNT(*) as total_trips,
    AVG(fare_value) as avg_fare,
    SUM(distance_km) as total_distance,
    SUM(fare_value) / (SUM(duration_minutes) / 60) as earnings_per_hour
FROM trips
WHERE driver_id = ?
  AND ended_at >= CURDATE()
  AND ended_at < CURDATE() + INTERVAL 1 DAY
  AND status = 'COMPLETED';
```

### Weekly Stats

```sql
SELECT 
    DATE(ended_at) as day,
    SUM(fare_value) as earnings,
    COUNT(*) as trips,
    AVG(fare_value) as avg_fare
FROM trips
WHERE driver_id = ?
  AND ended_at >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)
  AND ended_at < CURDATE() + INTERVAL 1 DAY
  AND status = 'COMPLETED'
GROUP BY DATE(ended_at)
ORDER BY day DESC;
```

> `DATE(ended_at)` no `SELECT`/`GROUP BY` continua ok — o problema de
> sargability é só quando a função entra no `WHERE`/`JOIN` filtrando a
> coluna indexada.

### Goal Progress

```sql
SELECT 
    dg.goal_amount,
    COALESCE(SUM(t.fare_value), 0) as actual_amount,
    (COALESCE(SUM(t.fare_value), 0) / dg.goal_amount * 100) as percent
FROM daily_goals dg
LEFT JOIN trips t ON t.driver_id = dg.driver_id
    AND t.ended_at >= dg.goal_date
    AND t.ended_at < dg.goal_date + INTERVAL 1 DAY
    AND t.status = 'COMPLETED'
WHERE dg.driver_id = ? AND dg.goal_date = CURDATE()
GROUP BY dg.id;
```

