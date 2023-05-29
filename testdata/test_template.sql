CREATE DATABASE IF NOT EXISTS {{ must_env("ENV") }}_mysqlbatch;
USE {{ must_env("ENV") }}_mysqlbatch;
DROP TABLE IF EXISTS {{ var("relation","hoge") }};
CREATE TABLE {{ var("relation","hoge") }} (
    id INTEGER auto_increment,
    name VARCHAR(191),
    age INTEGER,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `name` (`name`)
);
INSERT IGNORE INTO {{ var("relation","hoge") }}(name) VALUES(CONCAT(SUBSTRING(MD5(RAND()), 1, 40),'@example.com'));
INSERT IGNORE INTO {{ var("relation","hoge") }}(name) SELECT (CONCAT(SUBSTRING(MD5(RAND()), 1, 40),'@example.com')) FROM {{ var("relation","hoge") }};
{%- for i in range(3) %}
INSERT IGNORE INTO {{ var("relation","hoge") }}(name, age) SELECT (CONCAT(SUBSTRING(MD5(RAND()), 1, 40),'@example.com')),RAND() FROM {{ var("relation","hoge") }};
{%- endfor %}
SELECT * FROM {{ var("relation","hoge") }} WHERE age is NOT NULL LIMIT {{ must_var("limit") }};

