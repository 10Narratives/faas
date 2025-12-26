create table operations (
  id uuid primary key,
  service_name text not null,
  metadata jsonb not null,
  result jsonb,
  done boolean,
);