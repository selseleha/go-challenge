CREATE TABLE IF NOT EXISTS memberships (
    user_id   TEXT        NOT NULL,
    segment   TEXT        NOT NULL,
    tagged_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, segment)
);

-- Estimate/warm fallbacks filter by segment + recency (millions of rows need this).
CREATE INDEX IF NOT EXISTS memberships_segment_tagged_at_idx
    ON memberships (segment, tagged_at DESC);

CREATE TABLE IF NOT EXISTS outbox (
    id           BIGSERIAL PRIMARY KEY,
    user_id      TEXT        NOT NULL,
    segment      TEXT        NOT NULL,
    tagged_at    TIMESTAMPTZ NOT NULL,
    attempts     INT         NOT NULL DEFAULT 0,
    last_error   TEXT,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS outbox_claim_idx
    ON outbox (available_at)
    WHERE processed_at IS NULL;
