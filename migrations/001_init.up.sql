CREATE TABLE
    customers (
        customer_id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL
    );

CREATE TABLE
    wallets (
        customer_id TEXT PRIMARY KEY REFERENCES customers (customer_id) ON DELETE CASCADE,
        balance BIGINT NOT NULL DEFAULT 0,
        updated_at TIMESTAMPTZ NOT NULL
    );

CREATE TABLE
    sms_messages (
        message_id UUID PRIMARY KEY,
        customer_id TEXT NOT NULL REFERENCES customers (customer_id) ON DELETE CASCADE,
        to_number TEXT NOT NULL,
        text TEXT NOT NULL,
        priority TEXT NOT NULL,
        status TEXT NOT NULL,
        cost BIGINT NOT NULL,
        client_ref TEXT,
        send_attempts INT NOT NULL DEFAULT 0,
        sla_breached BOOLEAN NOT NULL DEFAULT FALSE,
        provider_message_id TEXT,
        created_at TIMESTAMPTZ NOT NULL,
        queued_at TIMESTAMPTZ,
        operator_sent_at TIMESTAMPTZ,
        failed_at TIMESTAMPTZ,
        fail_reason TEXT,
        UNIQUE (customer_id, client_ref)
    );

CREATE INDEX sms_messages_customer_created_idx ON sms_messages (customer_id, created_at DESC);

CREATE INDEX sms_messages_customer_status_idx ON sms_messages (customer_id, status);