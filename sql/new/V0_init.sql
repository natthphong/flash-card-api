

-- flashcard_sets
CREATE TABLE tbl_flashcard_sets
(
    id          serial PRIMARY KEY,
    owner_user_token VARCHAR(36) ,
    title       varchar(255),
    description text,
    is_public   varchar(1) DEFAULT 'N',
    create_at   timestamp  DEFAULT now(),
    create_by   varchar(255),
    update_at   timestamp,
    update_by   varchar(255),
    is_deleted  varchar(1) DEFAULT 'N'
);

-- flashcards
CREATE TABLE tbl_flashcards
(
    id         bigserial PRIMARY KEY,
    set_id     int REFERENCES tbl_flashcard_sets (id) ON DELETE CASCADE,
    front      text,
    back       text,
    choices    text[],                         -- 4 choices
    status     varchar(50) DEFAULT 'studying', -- studying| learned
    create_at  timestamp   DEFAULT now(),
    create_by  varchar(255),
    update_at  timestamp,
    update_by  varchar(255),
    is_deleted varchar(1)  DEFAULT 'N'
);

-- study_classes
CREATE TABLE tbl_study_classes
(
    id          serial PRIMARY KEY,
    owner_user_token VARCHAR(36),
    name        varchar(255),
    is_private  varchar(1) DEFAULT 'Y',
    invite_code varchar(255),
    create_at   timestamp  DEFAULT now(),
    create_by   varchar(255),
    update_at   timestamp,
    update_by   varchar(255),
    is_deleted  varchar(1) DEFAULT 'N'
);

CREATE TABLE tbl_study_class_sets
(
    class_id   int REFERENCES tbl_study_classes (id) ON DELETE CASCADE,
    set_id     int REFERENCES tbl_flashcard_sets (id),
    create_at  timestamp  DEFAULT now(),
    create_by  varchar(255),
    update_at  timestamp,
    update_by  varchar(255),
    is_deleted varchar(1) DEFAULT 'N',
    PRIMARY KEY (class_id, set_id)
);

CREATE TABLE tbl_study_class_members
(
    class_id int REFERENCES tbl_study_classes (id) ON DELETE CASCADE,
    user_id_token VARCHAR(36),
    role     varchar(50) DEFAULT 'student',
    PRIMARY KEY (class_id, user_id_token)
);



-- invites (study or group)
CREATE TABLE tbl_invites
(
    id          serial PRIMARY KEY,
    code        varchar(255),
    target_type varchar(50), -- 'study' | 'group'
    target_id   int,
    expiry      timestamp,
    used_by     int,         -- nullable
    invite_date date,
    create_at   timestamp  DEFAULT now(),
    create_by   varchar(255),
    update_at   timestamp,
    update_by   varchar(255),
    is_deleted  varchar(1) DEFAULT 'N'
);


-- daily plans (snapshot)
CREATE TABLE tbl_daily_plans
(
    id         serial PRIMARY KEY,
    user_id    int ,
    plan_date  date,
    card_ids   bigint[], -- ordered list
    test_at    timestamp  default null,
    create_at  timestamp  DEFAULT now(),
    create_by  varchar(255),
    update_at  timestamp,
    update_by  varchar(255),
    is_deleted varchar(1) DEFAULT 'N',
    UNIQUE (user_id, plan_date)
);

-- exam sessions
CREATE TABLE tbl_exam_sessions
(
    id           serial PRIMARY KEY,
    user_id      int ,
    question_ids bigint[], -- locked order
    answers      jsonb,    -- [{cardId,choice}]
    score        smallint,
    is_submitted varchar(1) DEFAULT 'N',
    expires_at   timestamp,
    create_at    timestamp  DEFAULT now(),
    create_by    varchar(255),
    update_at    timestamp,
    update_by    varchar(255),
    is_deleted   varchar(1) DEFAULT 'N'
);


-- Foreign-key lookup indexes
CREATE INDEX idx_tbl_flashcard_sets_owner_user_token
    ON tbl_flashcard_sets (owner_user_token);

CREATE INDEX idx_tbl_flashcards_set_id
    ON tbl_flashcards (set_id);

CREATE INDEX idx_tbl_study_class_sets_class_id
    ON tbl_study_class_sets (class_id);

CREATE INDEX idx_tbl_study_class_sets_set_id
    ON tbl_study_class_sets (set_id);

CREATE INDEX idx_tbl_study_class_members_user_id_token
    ON tbl_study_class_members (user_id_token);



-- Invites lookups by target
CREATE INDEX idx_tbl_invites_target
    ON tbl_invites (target_type, target_id);



-- Daily plans queries by date
CREATE INDEX idx_tbl_daily_plans_plan_date_user_id
    ON tbl_daily_plans (user_id, plan_date);

-- Exam sessions filters
CREATE INDEX idx_tbl_exam_sessions_user_id
    ON tbl_exam_sessions (user_id);

CREATE INDEX idx_tbl_exam_sessions_expires_at
    ON tbl_exam_sessions (expires_at);

CREATE INDEX idx_tbl_exam_sessions_is_submitted
    ON tbl_exam_sessions (is_submitted);




