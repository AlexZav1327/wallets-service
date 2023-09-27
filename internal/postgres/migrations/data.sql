-- +migrate Up
CREATE TABLE wallet (
    wallet_id UUID NOT NULL PRIMARY KEY,
    owner VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    balance NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE idempotency (
    transaction_key UUID NOT NULL PRIMARY KEY
);
