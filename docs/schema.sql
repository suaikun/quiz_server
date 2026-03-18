-- Quiz Server database schema
-- Compatible with MySQL 8.x

CREATE DATABASE IF NOT EXISTS quiz_db
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE quiz_db;

-- User account table
CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  password VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_username (username)
) ENGINE=InnoDB;

-- Question bank table
CREATE TABLE IF NOT EXISTS questions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  text VARCHAR(1024) NOT NULL,
  opt_a VARCHAR(255) NOT NULL,
  opt_b VARCHAR(255) NOT NULL,
  opt_c VARCHAR(255) NOT NULL,
  opt_d VARCHAR(255) NOT NULL,
  answer CHAR(1) NOT NULL COMMENT 'A/B/C/D',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CHECK (answer IN ('A', 'B', 'C', 'D'))
) ENGINE=InnoDB;

-- Best score per user
CREATE TABLE IF NOT EXISTS user_scores (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  score INT NOT NULL DEFAULT 0,
  time_taken INT NOT NULL DEFAULT 0 COMMENT 'seconds',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_scores_username (username),
  KEY idx_user_scores_score_time (score, time_taken),
  CONSTRAINT fk_user_scores_username
    FOREIGN KEY (username) REFERENCES users(username)
    ON DELETE CASCADE
    ON UPDATE CASCADE
) ENGINE=InnoDB;

-- Sample seed data (optional)
INSERT INTO questions (text, opt_a, opt_b, opt_c, opt_d, answer) VALUES
('Go language is developed by which company?', 'Microsoft', 'Google', 'Meta', 'Amazon', 'B'),
('Which protocol is commonly used for REST APIs?', 'HTTP', 'FTP', 'SMTP', 'SSH', 'A'),
('Redis sorted set is called?', 'List', 'Hash', 'Set', 'ZSet', 'D'),
('JWT consists of how many parts?', '2', '3', '4', '5', 'B'),
('Which SQL clause is used for sorting?', 'FILTER', 'ORDER BY', 'GROUP', 'HAVING', 'B');

