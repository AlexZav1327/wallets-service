-- +migrate Up
CREATE TABLE wallet (
    wallet_id UUID NOT NULL PRIMARY KEY,
    owner VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    balance NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT date_trunc('second', NOW()),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT date_trunc('second', NOW())
);

CREATE TABLE idempotency (
    transaction_key UUID NOT NULL PRIMARY KEY
);

CREATE TABLE history (
    wallet_id UUID NOT NULL,
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
        INSERT INTO history (wallet_id, owner, balance, currency, created_at, operation_type)
        VALUES (NEW.wallet_id, NEW.owner, NEW.balance, NEW.currency, date_trunc('second', NOW()), 'CREATE');
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO history (wallet_id, owner, balance, currency, created_at, operation_type)
        VALUES (NEW.wallet_id, NEW.owner, NEW.balance, NEW.currency, date_trunc('second', NOW()), 'UPDATE');
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO history (wallet_id, owner, balance, currency, created_at, operation_type)
        VALUES (OLD.wallet_id, OLD.owner, OLD.balance, OLD.currency, date_trunc('second', NOW()), 'DELETE');
END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

CREATE TRIGGER history_trigger
    AFTER INSERT OR UPDATE OR DELETE ON wallet
        FOR EACH ROW
        EXECUTE FUNCTION log_history();

