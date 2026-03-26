ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS employee_count BIGINT NOT NULL DEFAULT 0
CHECK (employee_count >= 0);

UPDATE game.businesses b
SET employee_count = counts.employee_count
FROM (
    SELECT business_id, COUNT(*)::bigint AS employee_count
    FROM game.business_employees
    GROUP BY business_id
) counts
WHERE b.id = counts.business_id;

UPDATE game.businesses
SET employee_count = 0
WHERE employee_count IS NULL;
