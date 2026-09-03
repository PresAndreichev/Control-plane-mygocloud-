ALTER TABLE applications DROP COLUMN IF EXISTS docker_image;
ALTER TABLE deployments DROP COLUMN IF EXISTS container_id;