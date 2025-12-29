CREATE TABLE IF NOT EXISTS exceptions (
    id SERIAL PRIMARY KEY,
    type VARCHAR(50),
    event_id VARCHAR(100),
    message TEXT,
    severity VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_exceptions_type ON exceptions(type);
CREATE INDEX IF NOT EXISTS idx_exceptions_event_id ON exceptions(event_id);
CREATE INDEX IF NOT EXISTS idx_exceptions_created_at ON exceptions(created_at);
