CREATE TYPE storage_user_role AS ENUM ('admin', 'assistant', 'lecturer', 'unconfirmed');

CREATE TABLE IF NOT EXISTS storage_user(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  name varchar(50) NOT NULL,
  role storage_user_role NOT NULL DEFAULT 'unconfirmed',
  password varchar(128) NOT NULL,
  active boolean DEFAULT TRUE,
  UNIQUE(name)
);

CREATE TRIGGER mdt_storage_user
  BEFORE UPDATE ON storage_user
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime (updated_at);
