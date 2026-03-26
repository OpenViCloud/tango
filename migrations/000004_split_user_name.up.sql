ALTER TABLE users
ADD COLUMN IF NOT EXISTS first_name TEXT,
ADD COLUMN IF NOT EXISTS last_name TEXT;

UPDATE users
SET
    first_name = COALESCE(NULLIF(first_name, ''), NULLIF(name, ''), 'Unknown'),
    last_name = COALESCE(last_name, '');

ALTER TABLE users
ALTER COLUMN first_name SET NOT NULL,
ALTER COLUMN last_name SET NOT NULL;

ALTER TABLE users
DROP COLUMN IF EXISTS name;
