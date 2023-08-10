-- +migrate Up
CREATE TABLE access_data (
    id SERIAL PRIMARY KEY,
    time TIMESTAMP NOT NULL,
    ip VARCHAR(50) NOT NULL
);