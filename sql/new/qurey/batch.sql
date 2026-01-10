WITH cfg AS (
    SELECT
        uc.user_id_token,
        uc.daily_target,
        COALESCE(uc.daily_flash_card_set_id, uc.default_flash_card_set_id) AS set_id,
        (now() AT TIME ZONE 'Asia/Bangkok')::date AS plan_date
    FROM tbl_user_config uc
    WHERE uc.daily_active = 'Y'
      AND uc.status = 'ACTIVE'
),

-- 1) due cards จาก SRS: เอามาก่อนตามลำดับเร่งด่วน
     due_ranked AS (
         SELECT
             s.user_id_token,
             s.card_id,
             row_number() OVER (
      PARTITION BY s.user_id_token
      ORDER BY
        s.next_review_at NULLS FIRST,
        s.box ASC,
        s.last_review_at NULLS FIRST,
        s.card_id ASC
    ) AS rn
         FROM tbl_user_flashcard_srs s
                  JOIN cfg ON cfg.user_id_token = s.user_id_token
         WHERE s.next_review_at IS NULL OR s.next_review_at <= now()
     ),
     due_selected AS (
         SELECT
             d.user_id_token,
             d.card_id,
             d.rn AS ord
         FROM due_ranked d
                  JOIN cfg ON cfg.user_id_token = d.user_id_token
         WHERE d.rn <= cfg.daily_target
     ),

     due_count AS (
         SELECT
             user_id_token,
             count(*)::int AS due_cnt
         FROM due_selected
         GROUP BY user_id_token
     ),

-- 2) fill cards: ถ้ายังไม่ครบ target ให้เติมจาก set_id
     fill_ranked AS (
         SELECT
             cfg.user_id_token,
             c.id AS card_id,
             row_number() OVER (
      PARTITION BY cfg.user_id_token
      ORDER BY c.seq ASC, c.id ASC
    ) AS rn
         FROM cfg
                  JOIN tbl_flashcards c
                       ON c.set_id = cfg.set_id
                           AND c.is_deleted = 'N'
                  LEFT JOIN due_selected d
                            ON d.user_id_token = cfg.user_id_token
                                AND d.card_id = c.id
                  LEFT JOIN tbl_user_flashcard_srs srs
                            ON srs.user_id_token = cfg.user_id_token
                                AND srs.card_id = c.id
         WHERE cfg.set_id IS NOT NULL
           AND d.card_id IS NULL          -- ไม่ซ้ำกับ due
           AND srs.card_id IS NULL        -- เอาเฉพาะ new card (ยังไม่เคยมี srs row)
     ),

     fill_selected AS (
         SELECT
             f.user_id_token,
             f.card_id,
             (100000 + f.rn) AS ord
         FROM fill_ranked f
                  JOIN cfg ON cfg.user_id_token = f.user_id_token
                  LEFT JOIN due_count dc ON dc.user_id_token = f.user_id_token
         WHERE f.rn <= GREATEST(cfg.daily_target - COALESCE(dc.due_cnt, 0), 0)
     ),

     all_cards AS (
         SELECT * FROM due_selected
         UNION ALL
         SELECT * FROM fill_selected
     ),

     final_plan AS (
         SELECT
             cfg.user_id_token,
             cfg.plan_date,
             COALESCE(
                     array_agg(a.card_id ORDER BY a.ord),
                     '{}'::bigint[]
             ) AS card_ids
         FROM cfg
                  LEFT JOIN all_cards a
                            ON a.user_id_token = cfg.user_id_token
         GROUP BY cfg.user_id_token, cfg.plan_date
     )

INSERT INTO tbl_daily_plans (user_id_token, plan_date, card_ids, create_at, update_at, is_deleted)
SELECT
    fp.user_id_token,
    fp.plan_date,
    fp.card_ids,
    now(),
    now(),
    'N'
FROM final_plan fp
-- ถ้าอยาก "ไม่ insert แผนที่ว่าง" ให้เปิดบรรทัดนี้:
-- WHERE array_length(fp.card_ids, 1) IS NOT NULL AND array_length(fp.card_ids, 1) > 0
    ON CONFLICT (user_id_token, plan_date)
DO UPDATE SET
    card_ids  = EXCLUDED.card_ids,
           update_at = now(),
           is_deleted = 'N';


