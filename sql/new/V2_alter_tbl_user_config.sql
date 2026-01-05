alter table public.tbl_user_config
    add daily_flash_card_set_id integer;

alter table public.tbl_user_config
    add default_flash_card_set_id integer;

alter table public.tbl_flashcards
    add seq integer default 0;

