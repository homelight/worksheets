drop table if exists worksheets;
create table worksheets (
  id             uuid,
  version        int,
  name           varchar,

  unique(id)
);

drop table if exists worksheet_values;
create table worksheet_values (
  id             serial,
  worksheet_id   uuid,
  index          int,
  from_version   int,
  to_version     int,
  value          varchar,

  unique(id)
);
