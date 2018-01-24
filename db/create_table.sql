DROP TABLE IF EXISTS audio_info;

CREATE TABLE audio_info
(
    video_id varchar(20) PRIMARY KEY,
    title varchar(100) NOT NULL,
    author varchar(100),
    thumbnail_url varchar(100),
    length integer,
    audio_path varchar(100) NOT NULL,
    keywords text,
    converted_at integer NOT NULL
)
WITH (OIDS=FALSE);
