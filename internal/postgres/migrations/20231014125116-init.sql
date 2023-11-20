-- +migrate Up
CREATE TABLE wallet (
    wallet_id UUID NOT NULL PRIMARY KEY,
    email VARCHAR NOT NULL,
    owner VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    balance NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT date_trunc('second', NOW()),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT date_trunc('second', NOW()),
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    inactive_mailed BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (email)
);

CREATE TABLE idempotency (
    transaction_key UUID NOT NULL PRIMARY KEY
);

CREATE TABLE history (
    wallet_id UUID NOT NULL,
    email VARCHAR NOT NULL,
    owner VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    balance NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    operation_type VARCHAR NOT NULL
);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION log_history()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO history (wallet_id, email, owner, balance, currency, created_at, operation_type)
        VALUES (NEW.wallet_id, NEW.email, NEW.owner, NEW.balance, NEW.currency, date_trunc('second', NOW()), 'CREATE');
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO history (wallet_id, email, owner, balance, currency, created_at, operation_type)
        VALUES (NEW.wallet_id, NEW.email, NEW.owner, NEW.balance, NEW.currency, date_trunc('second', NOW()), 'UPDATE');
END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION log_deleted_wallet()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO history (wallet_id, email, owner, balance, currency, created_at, operation_type)
    VALUES (OLD.wallet_id, OLD.email, OLD.owner, OLD.balance, OLD.currency, date_trunc('second', NOW()), 'DELETE');
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION log_mailed_wallet()
    RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO history (wallet_id, email, owner, balance, currency, created_at, operation_type)
    VALUES (OLD.wallet_id, OLD.email, OLD.owner, OLD.balance, OLD.currency, date_trunc('second', NOW()), 'MAIL');
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

CREATE TRIGGER history_trigger
    AFTER INSERT OR UPDATE ON wallet
        FOR EACH ROW
        WHEN ( NEW.deleted=FALSE AND NEW.inactive_mailed=FALSE)
        EXECUTE FUNCTION log_history();

CREATE TRIGGER delete_trigger
    AFTER UPDATE ON wallet
        FOR EACH ROW
        WHEN ( NEW.deleted=TRUE)
        EXECUTE FUNCTION log_deleted_wallet();

CREATE TRIGGER mail_trigger
    AFTER UPDATE ON wallet
        FOR EACH ROW
        WHEN ( OLD.inactive_mailed=FALSE AND NEW.inactive_mailed=TRUE)
        EXECUTE FUNCTION log_mailed_wallet();