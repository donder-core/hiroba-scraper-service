CREATE TABLE IF NOT EXISTS score_detail (
    song_no             INT          NOT NULL,
    level               INT          NOT NULL,
    taiko_no            INT          NOT NULL,
    crown_src           VARCHAR(255) NOT NULL DEFAULT '',
    best_score_icon_src VARCHAR(255) NOT NULL DEFAULT '',
    ranking             VARCHAR(50)  NOT NULL DEFAULT '',
    high_score          INT          NOT NULL DEFAULT 0,
    good                INT          NOT NULL DEFAULT 0,
    combo               INT          NOT NULL DEFAULT 0,
    ok                  INT          NOT NULL DEFAULT 0,
    drumroll            INT          NOT NULL DEFAULT 0,
    bad                 INT          NOT NULL DEFAULT 0,
    PRIMARY KEY (song_no, level, taiko_no)
);