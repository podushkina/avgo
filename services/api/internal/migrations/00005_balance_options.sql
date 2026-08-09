-- +goose Up

UPDATE training_options o SET text = v.new_text
FROM (VALUES
    ('Вернуть разницу только после подтверждения от банка, что платёж окончательный',
     'Сначала убедиться в банке, что платёж не отзовут'),
    ('Оставить лишнее себе',
     'Оставить лишние деньги себе и ничего не возвращать'),
    ('Вернуть 15 000 на указанную карту',
     'Вернуть 15 000 на карту, которую он прислал'),

    ('Отказаться и предложить оформить через Авито Доставку',
     'Отказаться и предложить Авито Доставку'),
    ('Только номер карты',
     'Только номер карты, без срока и кода'),
    ('Номер карты и CVC, без срока',
     'Номер карты и код с обратной стороны'),

    ('Отказаться от предоплаты и предложить осмотр или Авито Доставку',
     'Отказаться от предоплаты, предложить осмотр'),
    ('Внести половину как компромисс',
     'Внести половину суммы как компромисс'),
    ('Попросить его прислать видео товара и потом внести предоплату',
     'Попросить видео товара и внести предоплату'),

    ('Не сканировать и предложить оплату через сделку на площадке',
     'Не сканировать, платить через сделку площадки'),
    ('Отсканировать — QR надёжнее ссылки',
     'Отсканировать: QR-код надёжнее обычной ссылки'),
    ('Отсканировать и проверить получателя перед подтверждением',
     'Отсканировать и проверить получателя'),

    ('Дождаться реального зачисления на счёт',
     'Дождаться, пока деньги придут на счёт'),
    ('Закрыть страницу и платить только внутри приложения',
     'Закрыть страницу, платить в приложении'),
    ('Не платить и обратиться в поддержку Авито',
     'Не платить, написать в поддержку Авито'),
    ('Отказаться и платить только через сделку на площадке',
     'Отказаться, платить через сделку площадки')
) AS v(old_text, new_text)
WHERE o.text = v.old_text;

UPDATE training_options o SET position = s.new_position
FROM (
    SELECT id,
           (row_number() OVER (
               PARTITION BY step_id
               ORDER BY md5(id::text || 'antiscam-shuffle-v1')
           ) - 1)::int AS new_position
    FROM training_options
) AS s
WHERE o.id = s.id;

-- +goose StatementBegin
DO $$
DECLARE
    first_correct int;
    total_steps   int;
    max_gap       int;
BEGIN
    SELECT count(*) FILTER (WHERE position = 0), count(*)
    INTO first_correct, total_steps
    FROM training_options WHERE is_correct;

    IF first_correct = 0 THEN
        RAISE EXCEPTION
            'перемешивание не удалось: правильный ответ ни разу не на первой позиции из % шагов',
            total_steps;
    END IF;

    IF first_correct = total_steps THEN
        RAISE EXCEPTION
            'перемешивание не удалось: правильный ответ всегда на первой позиции';
    END IF;

    SELECT max(gap) INTO max_gap FROM (
        SELECT max(length(o.text)) FILTER (WHERE o.is_correct)
             - avg(length(o.text)) FILTER (WHERE NOT o.is_correct) AS gap
        FROM training_options o GROUP BY o.step_id
    ) t;

    IF max_gap > 20 THEN
        RAISE EXCEPTION
            'верный вариант всё ещё заметно длиннее прочих: разрыв % символов', max_gap;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
