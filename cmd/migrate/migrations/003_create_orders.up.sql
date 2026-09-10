CREATE TYPE order_status AS ENUM ('pending', 'confirmed', 'cancelled');

CREATE TABLE orders(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    user_id UUID NOT NULL,
    placed_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    status order_status NOT NULL DEFAULT 'pending'
);

