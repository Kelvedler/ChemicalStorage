CREATE TABLE IF NOT EXISTS storage(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  name varchar(100) NOT NULL,
  cells smallint NOT NULL
);

CREATE TRIGGER mdt_storage
  BEFORE UPDATE ON storage
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime (updated_at);

CREATE TABLE IF NOT EXISTS storage_cell(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  storage uuid NOT NULL REFERENCES storage (id) ON DELETE CASCADE,
  number smallint NOT NULL,
  UNIQUE(storage, number)
);

CREATE TRIGGER mdt_storage_cell
  BEFORE UPDATE ON storage_cell
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime (updated_at);

ALTER TABLE reagent_instance ALTER expires_at TYPE date;

ALTER TABLE reagent_instance ADD storage_cell uuid REFERENCES storage_cell (id) ON DELETE SET NULL;

CREATE FUNCTION cell_number_limit() RETURNS trigger AS $cell_number_limit$
  DECLARE
    cells_max smallint := (SELECT cells FROM storage WHERE id = NEW.storage);
  BEGIN
    IF NEW.number < 1 OR NEW.number > cells_max THEN
      RAISE EXCEPTION USING
        ERRCODE = 'A0001',
        MESSAGE = 'cell number out of limits',
        CONSTRAINT = 'storage_cell_number_limit',
        TABLE = 'storage_cell',
        COLUMN = 'number';
    END IF;
    RETURN NEW;
  END;
$cell_number_limit$ LANGUAGE plpgsql;

CREATE TRIGGER cell_number_limit BEFORE INSERT OR UPDATE ON storage_cell
  FOR EACH ROW EXECUTE FUNCTION cell_number_limit();

CREATE FUNCTION cell_constraint() RETURNS trigger AS $cell_constraint$
  DECLARE
    cell_max smallint := (SELECT number FROM storage_cell WHERE storage = NEW.id ORDER BY number DESC LIMIT 1);
  BEGIN
    if NEW.cells < cell_max THEN
      RAISE EXCEPTION USING
        ERRCODE = 'A0002',
        MESSAGE = 'cell with higher number exist',
        CONSTRAINT = 'storage_cells_constraint',
        TABLE = 'storage',
        COLUMN = 'cells';
    END IF;
    RETURN NEW;
  END;
$cell_constraint$ LANGUAGE plpgsql;

CREATE TRIGGER cell_constraint BEFORE UPDATE ON storage
  FOR EACH ROW EXECUTE FUNCTION cell_constraint();

