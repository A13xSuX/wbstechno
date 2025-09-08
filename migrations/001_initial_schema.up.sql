-- Миграция для создания первоначальной схемы базы данных

-- Таблица заказов
CREATE TABLE IF NOT EXISTS orders (
    order_uid          VARCHAR(255) PRIMARY KEY,
    track_number       VARCHAR(255) NOT NULL,
    entry              VARCHAR(255) NOT NULL,
    locale             VARCHAR(10) NOT NULL,
    internal_signature VARCHAR(255),
    customer_id        VARCHAR(255) NOT NULL,
    delivery_service   VARCHAR(255) NOT NULL,
    shardkey           VARCHAR(255) NOT NULL,
    sm_id              INTEGER NOT NULL,
    date_created       TIMESTAMP NOT NULL,
    oof_shard          VARCHAR(255) NOT NULL,
    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица доставки
CREATE TABLE IF NOT EXISTS delivery (
    id         SERIAL PRIMARY KEY,
    order_uid  VARCHAR(255) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    phone      VARCHAR(255) NOT NULL,
    zip        VARCHAR(255) NOT NULL,
    city       VARCHAR(255) NOT NULL,
    address    VARCHAR(255) NOT NULL,
    region     VARCHAR(255) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (order_uid)
);

-- Таблица платежей
CREATE TABLE IF NOT EXISTS payment (
    id            SERIAL PRIMARY KEY,
    order_uid     VARCHAR(255) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    transaction   VARCHAR(255) NOT NULL,
    request_id    VARCHAR(255),
    currency      VARCHAR(3) NOT NULL,
    provider      VARCHAR(255) NOT NULL,
    amount        INTEGER NOT NULL,
    payment_dt    BIGINT NOT NULL,
    bank          VARCHAR(255) NOT NULL,
    delivery_cost INTEGER DEFAULT 0,
    goods_total   INTEGER DEFAULT 0,
    custom_fee    INTEGER DEFAULT 0,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (order_uid)
);

-- Таблица товаров
CREATE TABLE IF NOT EXISTS items (
    id           SERIAL PRIMARY KEY,
    order_uid    VARCHAR(255) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id      INTEGER NOT NULL,
    track_number VARCHAR(255) NOT NULL,
    price        INTEGER NOT NULL,
    rid          VARCHAR(255) NOT NULL,
    name         VARCHAR(255) NOT NULL,
    sale         INTEGER DEFAULT 0,
    size         VARCHAR(255),
    total_price  INTEGER DEFAULT 0,
    nm_id        INTEGER,
    brand        VARCHAR(255) NOT NULL,
    status       INTEGER DEFAULT 0,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для улучшения производительности
CREATE INDEX IF NOT EXISTS idx_orders_order_uid ON orders(order_uid);
CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created);
CREATE INDEX IF NOT EXISTS idx_delivery_order_uid ON delivery(order_uid);
CREATE INDEX IF NOT EXISTS idx_payment_order_uid ON payment(order_uid);
CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);
CREATE INDEX IF NOT EXISTS idx_items_chrt_id ON items(chrt_id);