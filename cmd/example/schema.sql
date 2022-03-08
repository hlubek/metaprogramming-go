CREATE TABLE IF NOT EXISTS products
(
    product_id         uuid PRIMARY KEY,
    article_number     text,
    name               text,
    description        text,
    color              text,
    size               text,
    stock_availability int,
    price_cents        int,
    on_sale            bool
);
