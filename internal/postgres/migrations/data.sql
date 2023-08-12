-- +migrate Up
CREATE TABLE access_data (
    id SERIAL PRIMARY KEY,
    user_ip VARCHAR(25) NOT NULL,
    access_time VARCHAR(25) NOT NULL
);