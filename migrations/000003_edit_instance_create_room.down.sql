DROP TRIGGER cell_constraint ON storage;

DROP FUNCTION cell_constraint;

DROP TRIGGER cell_number_limit ON storage_cell;

DROP FUNCTION cell_number_limit();

ALTER TABLE reagent_instance DROP storage_cell;

ALTER TABLE reagent_instance ALTER expires_at TYPE timestamptz;

DROP TRIGGER mdt_storage_cell ON storage_cell;

DROP TABLE storage_cell;

DROP TRIGGER mdt_storage ON storage;

DROP TABLE storage;

