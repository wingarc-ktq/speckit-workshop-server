ALTER TABLE files DROP CONSTRAINT IF EXISTS files_owner_user_id_name_key;
ALTER TABLE files DROP CONSTRAINT IF EXISTS files_size_check;
ALTER TABLE files ADD CONSTRAINT files_size_check CHECK (size >= 0);
