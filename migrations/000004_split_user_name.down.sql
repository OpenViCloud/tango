ALTER TABLE users
ADD COLUMN IF NOT EXISTS name TEXT;

UPDATE users
SET name = TRIM(CONCAT(first_name, ' ', last_name));

UPDATE users
SET name = COALESCE(NULLIF(name, ''), first_name, 'Unknown');

ALTER TABLE users
ALTER COLUMN name SET NOT NULL;

ALTER TABLE users
DROP COLUMN IF EXISTS first_name,
DROP COLUMN IF EXISTS last_name;
