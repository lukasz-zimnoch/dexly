CREATE TYPE position_type AS ENUM ('LONG', 'SHORT');
CREATE TYPE position_status AS ENUM ('OPEN', 'CLOSED');
CREATE TYPE order_side AS ENUM ('BUY', 'SELL');

CREATE TABLE account (
    id UUID PRIMARY KEY,
    email VARCHAR NOT NULL,
    exchange VARCHAR NOT NULL,
    exchange_api_key VARCHAR NOT NULL,
    exchange_secret_key VARCHAR NOT NULL,
    risk_factor NUMERIC NOT NULL,
    open_position_limit INTEGER NOT NULL
);

CREATE TABLE workload (
    id UUID PRIMARY KEY,
    account_id UUID REFERENCES account NOT NULL,
    base_asset VARCHAR NOT NULL,
    quote_asset VARCHAR NOT NULL
);

CREATE TABLE position (
    id UUID PRIMARY KEY,
    workload_id UUID REFERENCES workload NOT NULL,
    type position_type NOT NULL,
    status position_status NOT NULL,
    entry_price NUMERIC NOT NULL,
    size NUMERIC NOT NULL,
    take_profit_price NUMERIC NOT NULL,
    stop_loss_price NUMERIC NOT NULL,
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