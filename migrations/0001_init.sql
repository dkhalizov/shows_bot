create table shows_bot.shows
(
    id             text                    not null
        primary key,
    name           text                    not null,
    overview       text,
    poster_url     text,
    status         text,
    first_air_date timestamp,
    provider       text                    not null,
    provider_id    text                    not null,
    created_at     timestamp default now() not null,
    imdb_id        text,
    unique (provider, provider_id)
);

alter table shows_bot.shows
    owner to postgres;

create table shows_bot.episodes
(
    id             text                    not null
        primary key,
    show_id        text                    not null
        references shows_bot.shows
            on delete cascade,
    name           text                    not null,
    season_number  integer                 not null,
    episode_number integer                 not null,
    air_date       timestamp,
    overview       text,
    provider       text                    not null,
    provider_id    text                    not null,
    created_at     timestamp default now() not null,
    unique (provider, provider_id)
);

alter table shows_bot.episodes
    owner to postgres;

create index idx_shows_imdb_id
    on shows_bot.shows (imdb_id)
    where (imdb_id IS NOT NULL);

create table shows_bot.users
(
    id         bigint                  not null
        primary key,
    username   text,
    first_name text,
    last_name  text,
    created_at timestamp default now() not null
);

alter table shows_bot.users
    owner to postgres;

create table shows_bot.notifications
(
    id          serial
        primary key,
    user_id     bigint                  not null
        references shows_bot.users
            on delete cascade,
    episode_id  text                    not null
        references shows_bot.episodes
            on delete cascade,
    notified_at timestamp,
    created_at  timestamp default now() not null,
    unique (user_id, episode_id)
);

alter table shows_bot.notifications
    owner to postgres;

create table shows_bot.user_shows
(
    user_id    bigint                  not null
        references shows_bot.users
            on delete cascade,
    show_id    text                    not null
        references shows_bot.shows
            on delete cascade,
    created_at timestamp default now() not null,
    primary key (user_id, show_id)
);

alter table shows_bot.user_shows
    owner to postgres;

