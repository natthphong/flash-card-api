CREATE TABLE tbl_exam_answers (
                                  id            BIGSERIAL PRIMARY KEY,
                                  session_id    BIGINT NOT NULL REFERENCES tbl_exam_sessions(id) ON DELETE CASCADE,
                                  question_id   BIGINT NOT NULL REFERENCES tbl_exam_questions(id) ON DELETE CASCADE,
                                  user_id_token VARCHAR(36) NOT NULL,

    -- for MCQ
                                  selected_choice TEXT,

    -- for typing
                                  typed_text   TEXT,

    -- for speaking
                                  audio_url    TEXT,
                                  recognized_text TEXT,
                                  pronunciation_score SMALLINT,

                                  is_correct   VARCHAR(1),
                                  score_awarded SMALLINT DEFAULT 0,
                                  detail       JSONB,

                                  answered_at  TIMESTAMP DEFAULT now(),

                                  UNIQUE(session_id, question_id)
);

CREATE INDEX idx_exam_answers_session
    ON tbl_exam_answers(session_id);
