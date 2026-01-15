-- Add replacement tracking to invoices table
-- This enables tracking when an invoice is replaced due to artifact re-upload

-- Add field to track invoice replacement chain
ALTER TABLE invoices
ADD COLUMN replaced_by_invoice_id UUID REFERENCES invoices(invoice_id) ON DELETE SET NULL;

-- Index for efficient queries on replacement relationships
CREATE INDEX idx_invoices_replaced_by ON invoices(replaced_by_invoice_id);

-- Index for duplicate detection query (program + vendor + invoice_number)
-- Partial index only on non-deleted invoices for efficiency
CREATE INDEX idx_invoices_lookup ON invoices(program_id, vendor_name, invoice_number)
WHERE deleted_at IS NULL;

-- Document the purpose of this field
COMMENT ON COLUMN invoices.replaced_by_invoice_id IS
'Links to the invoice that replaced this one when artifact was re-uploaded. Used for audit trail and tracking invoice replacement chains.';
