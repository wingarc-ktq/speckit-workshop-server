ALTER TABLE files DROP CONSTRAINT IF EXISTS files_size_check;
ALTER TABLE files ADD CONSTRAINT files_size_check CHECK (size > 0);
ALTER TABLE files ADD CONSTRAINT files_owner_user_id_name_key UNIQUE (owner_user_id, name);
