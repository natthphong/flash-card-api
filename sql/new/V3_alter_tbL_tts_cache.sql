alter table public.tbl_tts_cache
add audio_key varchar(50);

drop table public.tbl_flashcard_auto_source;

drop table public.tbl_flashcard_audio;


alter table public.tbl_chat_messages
    add recognized_texts TEXT[];

