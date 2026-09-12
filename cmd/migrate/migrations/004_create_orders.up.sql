CREATE TYPE order_status AS ENUM ('pending', 'confirmed', 'cancelled');

CREATE TABLE orders(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    placed_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    total_price numeric(15, 2) NOT NULL CHECK (total_price >= 0),
    status order_status NOT NULL DEFAULT 'pending',
    idempotency_key VARCHAR(255) NOT NULL UNIQUE
);
