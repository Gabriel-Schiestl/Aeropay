CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

CREATE TABLE IF NOT EXISTS accounts (
    id      UUID PRIMARY KEY,
    balance NUMERIC(19, 4) NOT NULL DEFAULT 0 CHECK (balance >= 0)
);

CREATE TABLE IF NOT EXISTS payments (
    id           UUID PRIMARY KEY,
    amount       NUMERIC(19, 4) NOT NULL CHECK (amount > 0),
    currency     CHAR(3) NOT NULL,
    from_account UUID NOT NULL REFERENCES accounts (id),
    to_account   UUID NOT NULL REFERENCES accounts (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ledger (
    id         BIGSERIAL PRIMARY KEY,
    amount     NUMERIC(19, 4) NOT NULL,
    currency   CHAR(3) NOT NULL,
    account    UUID NOT NULL REFERENCES accounts (id),
    payment_id UUID NOT NULL REFERENCES payments (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_from_account ON payments (from_account);
CREATE INDEX IF NOT EXISTS idx_payments_to_account ON payments (to_account);
CREATE INDEX IF NOT EXISTS idx_ledger_payment_id ON ledger (payment_id);
CREATE INDEX IF NOT EXISTS idx_ledger_account ON ledger (account);

DO $$
BEGIN
    CREATE TYPE idempotency_key_status AS ENUM ('processing', 'completed', 'error');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key          UUID PRIMARY KEY,
    request_hash TEXT NOT NULL,
    status       idempotency_key_status NOT NULL DEFAULT 'processing',
    payment_id   UUID REFERENCES payments (id),
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

DO $$
BEGIN
    CREATE TYPE payments_outbox_status AS ENUM ('pending', 'processed', 'error');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS payments_outbox (
    id         BIGSERIAL PRIMARY KEY,
    status     payments_outbox_status NOT NULL DEFAULT 'pending',
    event_type TEXT NOT NULL,
    payload    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE payments_outbox DROP COLUMN IF EXISTS key;
