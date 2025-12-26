create table functions (
  id uuid primary key,
  source_path text not null,
  entrypoint text not null,
  create_time timestamp default now(),
  update_time timestamp default not(),
);