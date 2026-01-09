CREATE TABLE tbl_user_config (
         user_id_token   VARCHAR(36) PRIMARY KEY,
         status          VARCHAR(50) DEFAULT 'ACTIVE', -- ACTIVE|SUSPENDED
         daily_active    VARCHAR(1) DEFAULT 'Y',
         daily_target    INT DEFAULT 20,

         locale          VARCHAR(20) DEFAULT 'en-US',
         create_at       TIMESTAMP DEFAULT now(),
         update_at       TIMESTAMP
);


CREATE TABLE tbl_tts_cache (
           id              BIGSERIAL PRIMARY KEY,
           cache_key       VARCHAR(128) UNIQUE NOT NULL,
           text            TEXT NOT NULL,
           voice           VARCHAR(50) NOT NULL,
           speed           NUMERIC(3,2) NOT NULL,
           format          VARCHAR(10) NOT NULL DEFAULT 'mp3',
           locale          VARCHAR(20) NOT NULL DEFAULT 'en-US',
           audio_url       TEXT NOT NULL,
           duration_ms     INT,
           created_at      TIMESTAMP DEFAULT now(),
           last_accessed_at TIMESTAMP,
           hit_count       INT DEFAULT 0
);



CREATE TABLE tbl_tts_cache (
                               id              BIGSERIAL PRIMARY KEY,
                               cache_key       VARCHAR(128) UNIQUE NOT NULL,
                               text            TEXT NOT NULL,
                               voice           VARCHAR(50) NOT NULL,
                               speed           NUMERIC(3,2) NOT NULL,
                               format          VARCHAR(10) NOT NULL DEFAULT 'mp3',
                               locale          VARCHAR(20) NOT NULL DEFAULT 'en-US',
                               audio_url       TEXT NOT NULL,
                               duration_ms     INT,
                               created_at      TIMESTAMP DEFAULT now(),
                               last_accessed_at TIMESTAMP,
                               hit_count       INT DEFAULT 0
);

CREATE TABLE tbl_flashcard_audio (
                                     flashcard_id   BIGINT NOT NULL REFERENCES tbl_flashcards(id) ON DELETE CASCADE,
                                     audio_type     VARCHAR(20) NOT NULL, -- front_normal|front_slow|back
                                     tts_cache_id   BIGINT NOT NULL REFERENCES tbl_tts_cache(id),
                                     created_at     TIMESTAMP DEFAULT now(),
                                     PRIMARY KEY (flashcard_id, audio_type)
);


CREATE TABLE tbl_user_flashcard_srs (
                                        user_id_token   VARCHAR(36) NOT NULL,
                                        card_id         BIGINT NOT NULL REFERENCES tbl_flashcards(id) ON DELETE CASCADE,
                                        box             SMALLINT DEFAULT 1, -- 1..5
                                        next_review_at  TIMESTAMP,
                                        last_review_at  TIMESTAMP,
                                        streak          INT DEFAULT 0,
                                        total_reviews   INT DEFAULT 0,
                                        last_grade      SMALLINT, -- 0..5
                                        created_at      TIMESTAMP DEFAULT now(),
                                        updated_at      TIMESTAMP,
                                        PRIMARY KEY (user_id_token, card_id)
);

CREATE INDEX idx_srs_next_review
    ON tbl_user_flashcard_srs(user_id_token, next_review_at);




CREATE TABLE tbl_review_log (
                                id            BIGSERIAL PRIMARY KEY,
                                user_id_token VARCHAR(36) NOT NULL,
                                card_id       BIGINT NOT NULL REFERENCES tbl_flashcards(id) ON DELETE CASCADE,
                                source        VARCHAR(20) NOT NULL, -- 'DAILY'|'EXAM'|'PRACTICE'
                                grade         SMALLINT NOT NULL, -- 0..5
                                is_correct    VARCHAR(1) NOT NULL,
                                answer_detail JSONB, -- {choice, expected, userInput...}
                                created_at    TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_review_log_user_time
    ON tbl_review_log(user_id_token, created_at);



CREATE TABLE tbl_learning_event (
                                    id            BIGSERIAL PRIMARY KEY,
                                    user_id_token VARCHAR(36) NOT NULL,
                                    session_id    VARCHAR(50),
                                    event_type    VARCHAR(50) NOT NULL,
                                    payload       JSONB NOT NULL,
                                    created_at    TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_learning_event_user_time
    ON tbl_learning_event(user_id_token, created_at);




CREATE TABLE tbl_flashcard_auto_source (
       id            BIGSERIAL PRIMARY KEY,
       user_id_token VARCHAR(36) NOT NULL,
       source_key    VARCHAR(128) NOT NULL,
       source_type   VARCHAR(50) NOT NULL,
       flashcard_id  BIGINT NOT NULL REFERENCES tbl_flashcards(id) ON DELETE CASCADE,
       created_at    TIMESTAMP DEFAULT now(),
       UNIQUE (user_id_token, source_key)
);



CREATE TABLE tbl_pronunciation_attempt (
       id             BIGSERIAL PRIMARY KEY,
       user_id_token  VARCHAR(36) NOT NULL,
       card_id        BIGINT REFERENCES tbl_flashcards(id) ON DELETE SET NULL,
       expected_text  TEXT NOT NULL,
       recognized_text TEXT NOT NULL,
       wer            NUMERIC(6,4),
       score_overall  SMALLINT,
       score_detail   JSONB,
       audio_url      TEXT,
       created_at     TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_pron_attempt_user_time
    ON tbl_pronunciation_attempt(user_id_token, created_at);



drop table tbl_exam_sessions;


CREATE TABLE tbl_exam_sessions (
   id            BIGSERIAL PRIMARY KEY,
   user_id_token VARCHAR(36) NOT NULL,

   mode          VARCHAR(30) NOT NULL, -- 'MCQ'|'TYPING'|'LISTENING'|'SPEAKING'|'MIXED'
   source_set_id INT NULL REFERENCES tbl_flashcard_sets(id),
   plan_id       INT NULL REFERENCES tbl_daily_plans(id),

   total_questions INT NOT NULL,
   time_limit_sec  INT, -- nullable = no limit
   status        VARCHAR(20) DEFAULT 'ACTIVE', -- ACTIVE|SUBMITTED|EXPIRED|CANCELLED

   started_at    TIMESTAMP DEFAULT now(),
   expires_at    TIMESTAMP,
   submitted_at  TIMESTAMP,

   score_total   SMALLINT DEFAULT 0,
   score_max     SMALLINT DEFAULT 0,

   create_at     TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_exam_sessions_user_status
    ON tbl_exam_sessions(user_id_token, status);
CREATE INDEX idx_exam_sessions_expires
    ON tbl_exam_sessions(expires_at);


CREATE TABLE tbl_exam_questions (
    id            BIGSERIAL PRIMARY KEY,
    session_id    BIGINT NOT NULL REFERENCES tbl_exam_sessions(id) ON DELETE CASCADE,
    seq           INT NOT NULL, -- 1..N

    card_id       BIGINT NOT NULL REFERENCES tbl_flashcards(id),
    question_type VARCHAR(20) NOT NULL, -- MCQ|TYPING|LISTENING|SPEAKING

-- snapshot for consistency
    front_snapshot TEXT,
    back_snapshot  TEXT,
    choices_snapshot TEXT[], -- for MCQ

-- listening/speaking support
    prompt_tts_cache_id BIGINT REFERENCES tbl_tts_cache(id),

    score_max     SMALLINT NOT NULL DEFAULT 1,

    UNIQUE(session_id, seq)
);

CREATE INDEX idx_exam_questions_session
    ON tbl_exam_questions(session_id);


ALTER TABLE tbl_daily_plans
    ADD COLUMN user_id_token VARCHAR(36);
ALTER TABLE tbl_daily_plans
    ADD COLUMN user_id_token VARCHAR(36);




ALTER TABLE tbl_daily_plans
DROP COLUMN user_id;

ALTER TABLE tbl_daily_plans
    ALTER COLUMN user_id_token SET NOT NULL;

DROP INDEX IF EXISTS idx_tbl_daily_plans_plan_date_user_id;
CREATE INDEX idx_tbl_daily_plans_plan_date_user
    ON tbl_daily_plans (user_id_token, plan_date);

ALTER TABLE tbl_daily_plans
DROP CONSTRAINT IF EXISTS tbl_daily_plans_user_id_plan_date_key;

ALTER TABLE tbl_daily_plans
    ADD CONSTRAINT uq_daily_plan UNIQUE (user_id_token, plan_date);

CREATE TABLE tbl_chat_sessions (
                                   id            VARCHAR(50) PRIMARY KEY, -- uuid
                                   user_id_token VARCHAR(36) NOT NULL,
                                   topic         TEXT,
                                   created_at    TIMESTAMP DEFAULT now()
);

CREATE TABLE tbl_chat_messages (
                                   id            BIGSERIAL PRIMARY KEY,
                                   session_id    VARCHAR(50) REFERENCES tbl_chat_sessions(id) ON DELETE CASCADE,
                                   role          VARCHAR(10) NOT NULL, -- user|ai
                                   text          TEXT NOT NULL,
                                   tts_cache_id  BIGINT REFERENCES tbl_tts_cache(id),
                                   created_at    TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_chat_msg_session_time ON tbl_chat_messages(session_id, created_at);



create table public.tbl_flashcard_sets_tracker
(
    set_id        integer,
    user_id_token varchar(36),
    tracker_type varchar(36),
    card_id       bigint,
    primary key (user_id_token,set_id,tracker_type)
);

