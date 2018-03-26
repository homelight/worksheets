-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
-- 
-- http://www.apache.org/licenses/LICENSE-2.0
-- 
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

drop table if exists worksheets;
create table worksheets (
  id             uuid,
  version        int,
  name           varchar,

  unique(id)
);

drop table if exists worksheet_edits;
create table worksheet_edits (
  edit_id        uuid,
  created_at     bigint,
  worksheet_id   uuid,
  to_version     int,

  -- Edits can modify any worksheet at most once.
  unique(edit_id, worksheet_id),

  -- Only one edit can lead to a worksheet being updated to a specific version.
  unique(worksheet_id, to_version)
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

drop table if exists worksheet_parents;
create table worksheet_parents (
  child_id           uuid,
  parent_id          uuid,
  parent_field_index int,

  unique(child_id, parent_id, parent_field_index)
);

drop table if exists worksheet_slice_elements;
create table worksheet_slice_elements (
  id             serial,
  slice_id       uuid,
  rank           int,
  from_version   int,
  to_version     int,
  value          varchar,

  unique(id)
);
