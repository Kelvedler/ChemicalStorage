DROP TRIGGER reagent_instance_constraint ON reagent_instance;

DROP FUNCTION reagent_instance_constraint;

DROP TRIGGER cell_constraint ON storage;

DROP FUNCTION cell_constraint;

DROP TRIGGER cell_number_limit ON storage_cell;

DROP FUNCTION cell_number_limit();

ALTER TABLE reagent_instance DROP storage_cell, ADD used boolean NOT NULL DEFAULT false, ALTER expires_at TYPE timestamptz, DROP deleted_at;

DROP TRIGGER mdt_storage_cell ON storage_cell;

DROP TABLE storage_cell;

DROP TRIGGER mdt_storage ON storage;

DROP TABLE storage;

