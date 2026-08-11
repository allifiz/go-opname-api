-- +goose Up
ALTER TABLE purchase_orders
    ADD CONSTRAINT uq_purchase_order_procurement_request UNIQUE (procurement_request_id);

-- +goose Down
ALTER TABLE purchase_orders
    DROP CONSTRAINT IF EXISTS uq_purchase_order_procurement_request;
