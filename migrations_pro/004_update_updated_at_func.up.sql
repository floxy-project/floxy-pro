CREATE FUNCTION workflows.update_updated_at_column() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
    DECLARE
        p text;
    BEGIN
        FOR p IN
            SELECT relname
            FROM pg_class c
                     JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname = 'workflows' AND relname LIKE 'workflow_instances_%'
            LOOP
                EXECUTE format('DROP TRIGGER IF EXISTS update_workflow_instances_updated_at ON workflows.%I;', p);
                EXECUTE format('CREATE TRIGGER update_workflow_instances_updated_at BEFORE UPDATE ON workflows.%I FOR EACH ROW EXECUTE FUNCTION workflows.update_updated_at_column();', p);
            END LOOP;
    END $$;
