DROP PROCEDURE IF EXISTS workflows.cleanup_all();

CREATE OR REPLACE PROCEDURE workflows.cleanup_all() AS $$
BEGIN
    CALL partman.run_maintenance_proc();
END;
$$ LANGUAGE plpgsql;
