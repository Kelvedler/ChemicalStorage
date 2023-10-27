CREATE EXTENSION moddatetime;

CREATE TABLE IF NOT EXISTS reagent(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  name varchar(300) NOT NULL,
  formula varchar(50) NOT NULL,
  UNIQUE (formula)
);

CREATE TRIGGER mdt_reagent
  BEFORE UPDATE ON reagent
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime (updated_at);

CREATE TABLE IF NOT EXISTS reagent_instance(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  reagent uuid NOT NULL REFERENCES reagent (id) ON DELETE CASCADE,
  used boolean NOT NULL DEFAULT false,
  used_at timestamptz,
  expires_at timestamptz
);

CREATE TRIGGER mdt_reagent_instance
  BEFORE UPDATE ON reagent_instance
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime (updated_at);

