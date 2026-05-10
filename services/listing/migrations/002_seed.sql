INSERT INTO users (name, email, phone, city)
SELECT
    'Пользователь ' || i,
    'user' || i || '@example.com',
    '+7' || (9000000000 + i)::BIGINT,
    (ARRAY[
        'Москва', 'Санкт-Петербург', 'Казань',
        'Екатеринбург', 'Новосибирск', 'Нижний Новгород',
        'Самара', 'Краснодар', 'Ростов-на-Дону', 'Уфа'
    ])[1 + (i % 10)]
FROM generate_series(1, 1000) AS i;

WITH
    cats  AS (SELECT UNNEST(ARRAY[
        'Электроника', 'Авто', 'Недвижимость', 'Одежда',
        'Спорт', 'Мебель', 'Работа', 'Животные'
    ]) AS v, generate_series(1, 8) AS n),
    cities AS (SELECT UNNEST(ARRAY[
        'Москва', 'Санкт-Петербург', 'Казань',
        'Екатеринбург', 'Новосибирск', 'Самара'
    ]) AS v, generate_series(1, 6) AS n),
    adjectives AS (SELECT UNNEST(ARRAY[
        'Новый', 'Б/у', 'Отличный', 'Срочно', 'Редкий', 'Дешёвый'
    ]) AS v, generate_series(1, 6) AS n),
    items AS (SELECT UNNEST(ARRAY[
        'iPhone', 'Ноутбук', 'Диван', 'Велосипед', 'Куртка',
        'Телевизор', 'Холодильник', 'Стол', 'Кресло', 'Планшет'
    ]) AS v, generate_series(1, 10) AS n)
INSERT INTO listings (user_id, title, description, price, category, location, status, is_promoted, promoted_until)
SELECT
    u.id,
    adj.v || ' ' || itm.v || ' ' || gs.i,
    'Описание товара: ' || adj.v || ' ' || itm.v || '. Состояние хорошее. Торг уместен. Объявление номер ' || gs.i || '.',
    round((random() * 99000 + 1000)::NUMERIC, 2),
    cat.v,
    cit.v,
    CASE
        WHEN gs.i % 20 = 0 THEN 'pending'
        WHEN gs.i % 30 = 0 THEN 'rejected'
        ELSE 'published'
    END::listing_status,
    gs.i % 15 = 0,
    CASE WHEN gs.i % 15 = 0 THEN now() + interval '30 days' ELSE NULL END
FROM generate_series(1, 50000) AS gs(i)
JOIN (SELECT id, row_number() OVER () AS rn FROM users) u
    ON u.rn = 1 + (gs.i % 1000)
JOIN cats  cat ON cat.n = 1 + (gs.i % 8)
JOIN cities cit ON cit.n = 1 + (gs.i % 6)
JOIN adjectives adj ON adj.n = 1 + (gs.i % 6)
JOIN items itm ON itm.n = 1 + (gs.i % 10);
