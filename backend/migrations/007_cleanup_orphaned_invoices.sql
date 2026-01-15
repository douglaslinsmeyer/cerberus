-- Clean up existing orphaned invoices (artifact_id = NULL) created before duplicate prevention was added
-- This migration handles the backlog of duplicate invoices from artifact replacements

-- Find orphaned invoices (artifact_id is NULL) and match them with newer invoices
-- that have the same business identifiers (program, vendor, invoice_number)
WITH orphaned_invoices AS (
  SELECT
    i.invoice_id,
    i.program_id,
    i.vendor_name,
    i.invoice_number,
    i.submitted_at
  FROM invoices i
  WHERE i.artifact_id IS NULL
    AND i.deleted_at IS NULL
    AND i.submitted_at >= NOW() - INTERVAL '30 days' -- Only recent orphans (last 30 days)
),
likely_duplicates AS (
  SELECT DISTINCT ON (oi.invoice_id)
    oi.invoice_id AS old_invoice_id,
    i.invoice_id AS new_invoice_id,
    i.submitted_at,
    i.artifact_id
  FROM orphaned_invoices oi
  INNER JOIN invoices i ON
    i.program_id = oi.program_id
    AND i.vendor_name = oi.vendor_name
    AND (
      -- Match on invoice_number if both are present
      (i.invoice_number = oi.invoice_number) OR
      -- Or both are NULL (no invoice number on either)
      (i.invoice_number IS NULL AND oi.invoice_number IS NULL)
    )
    AND i.artifact_id IS NOT NULL  -- New invoice has artifact link
    AND i.deleted_at IS NULL       -- New invoice is active
    AND i.submitted_at > oi.submitted_at  -- New invoice created after old one
  ORDER BY oi.invoice_id, i.submitted_at ASC  -- Take earliest matching new invoice
)
-- Update orphaned invoices: mark as deleted and link to replacement
UPDATE invoices
SET
  deleted_at = NOW(),
  replaced_by_invoice_id = ld.new_invoice_id,
  rejected_reason = COALESCE(rejected_reason || '; ', '') ||
    'Auto-archived: Replaced by newer invoice from re-uploaded artifact (cleanup migration)'
FROM likely_duplicates ld
WHERE invoices.invoice_id = ld.old_invoice_id;

-- Log the results
DO $$
DECLARE
  cleanup_count INT;
BEGIN
  SELECT COUNT(*) INTO cleanup_count
  FROM invoices
  WHERE deleted_at >= NOW() - INTERVAL '1 minute'
    AND replaced_by_invoice_id IS NOT NULL
    AND rejected_reason LIKE '%cleanup migration%';

  RAISE NOTICE 'Cleanup migration completed: % orphaned invoices archived', cleanup_count;
END $$;
