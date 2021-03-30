CREATE TYPE position_type AS ENUM ('LONG', 'SHORT');
CREATE TYPE position_status AS ENUM ('OPEN', 'CLOSE');
CREATE TYPE order_side AS ENUM ('BUY', 'SELL');

CREATE TABLE position (
    id UUID PRIMARY KEY,
    type position_type NOT NULL,
    status position_status NOT NULL,
    entry_price NUMERIC NOT NULL,
    size NUMERIC NOT NULL,
    take_profit_price NUMERIC NOT NULL,
    stop_loss_price NUMERIC NOT NULL,
    pair VARCHAR(16) NOT NULL,
    exchange VARCHAR(64) NOT NULL,
    time TIMESTAMP NOT NULL
);

CREATE TABLE position_order (
    id UUID PRIMARY KEY,
    position_id UUID REFERENCES position NOT NULL,
    side order_side NOT NULL,
    price NUMERIC NOT NULL,
    size NUMERIC NOT NULL,
    time TIMESTAMP NOT NULL,
    executed BOOLEAN NOT NULL,
    UNIQUE(position_id, side)
);