-- LANdapter factory reset: remove application data while preserving schema.
-- Keep common migration metadata tables intact so schema state is not lost.
-- This intentionally truncates every other ordinary table in the public schema.
DO $factory_reset$
DECLARE
    table_record record;
BEGIN
    FOR table_record IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename NOT IN (
              'schema_migrations',
              'goose_db_version',
              'flyway_schema_history',
              'atlas_schema_revisions'
          )
    LOOP
        EXECUTE format(
            'TRUNCATE TABLE %I.%I RESTART IDENTITY CASCADE',
            'public',
            table_record.tablename
        );
    END LOOP;
END;
$factory_reset$;
