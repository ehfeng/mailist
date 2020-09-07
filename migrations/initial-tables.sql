create table lists (
    name text primary key,
    created timestamp default now()
);

create table subscribers (
    created timestamp default now(),
    list text not null,
    email text not null,
    unique(list, email)
);
