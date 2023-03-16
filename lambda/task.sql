CREATE DATABASE IF NOT EXISTS mysqlbatch;
USE mysqlbatch;
DROP TABLE IF EXISTS users;
CREATE TABLE users (
    id INTEGER auto_increment,
    name VARCHAR(191),
    age INTEGER,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `name` (`name`)
);
INSERT IGNORE INTO users(name) VALUES(CONCAT(SUBSTRING(MD5(RAND()), 1, 40),'@example.com'));
INSERT IGNORE INTO users(name) SELECT (CONCAT(SUBSTRING(MD5(RAND()), 1, 40),'@example.com')) FROM users;
INSERT IGNORE INTO users(name, age) SELECT (CONCAT(SUBSTRING(MD5(RAND()), 1, 40),'@example.com')),RAND() FROM users;
INSERT IGNORE INTO users(name, age) SELECT (CONCAT(SUBSTRING(MD5(RAND()), 1, 40),'@example.com')),RAND() FROM users;
INSERT IGNORE INTO users(name, age) SELECT (CONCAT(SUBSTRING(MD5(RAND()), 1, 40),'@example.com')),RAND() FROM users;
SELECT * FROM users WHERE age is NOT NULL LIMIT 5;

