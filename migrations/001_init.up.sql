CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    username TEXT NOT NULL DEFAULT '',
    first_name TEXT NOT NULL DEFAULT '',
    timezone TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE expenses (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    category TEXT NOT NULL
        CHECK (
            category IN (
                'food',
                'transport',
                'housing',
                'entertainment',
                'health',
                'shopping',
                'other'
            )
        ),
    spent_at DATE NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expenses_user_spent_at
    ON expenses(user_id, spent_at DESC);

CREATE INDEX idx_expenses_user_category
    ON expenses(user_id, category);
