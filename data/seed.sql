CREATE DATABASE project_legacy WITH ENCODING = 'UTF8' CONNECTION LIMIT = - 1;

CREATE ROLE project_legacy_admin WITH NOSUPERUSER INHERIT NOCREATEROLE LOGIN NOREPLICATION
  NOBYPASSRLS ENCRYPTED PASSWORD '';
