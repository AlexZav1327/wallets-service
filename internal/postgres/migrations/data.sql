-- +migrate Up
CREATE TABLE access_data (
    id SERIAL PRIMARY KEY,
    user_ip VARCHAR NOT NULL,
    access_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);