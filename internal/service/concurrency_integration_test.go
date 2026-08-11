package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/allifiz/go-opname-api/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is required for integration tests")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func scalarID(t *testing.T, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func execSQL(t *testing.T, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func uniqueName(prefix string) string { return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()) }

func seedUser(t *testing.T, pool *pgxpool.Pool, roleCode string) string {
	t.Helper()
	return scalarID(t, pool, `
		INSERT INTO users (role_id, name, email, password_hash)
		SELECT id, $2, $3, 'test' FROM roles WHERE code = $1
		RETURNING id::text`, roleCode, uniqueName(roleCode), uniqueName(roleCode)+"@test.local")
}

func seedMaterial(t *testing.T, pool *pgxpool.Pool, qty string) (string, string) {
	t.Helper()
	unitID := scalarID(t, pool, `SELECT id::text FROM units WHERE code='KG'`)
	materialID := scalarID(t, pool, `INSERT INTO materials(name, unit_id) VALUES($1,$2) RETURNING id::text`, uniqueName("material"), unitID)
	execSQL(t, pool, `INSERT INTO material_stocks(material_id, qty, unit_id) VALUES($1,$2,$3)`, materialID, qty, unitID)
	return materialID, unitID
}

func seedScheduledMenu(t *testing.T, pool *pgxpool.Pool, materialID, unitID string) string {
	t.Helper()
	periodID := scalarID(t, pool, `INSERT INTO periods(name,start_date,end_date) VALUES($1,CURRENT_DATE,CURRENT_DATE+13) RETURNING id::text`, uniqueName("period"))
	templateID := scalarID(t, pool, `INSERT INTO menu_templates(name) VALUES($1) RETURNING id::text`, uniqueName("menu"))
	scheduledID := scalarID(t, pool, `INSERT INTO scheduled_menus(period_id,menu_template_id,menu_date,planned_portions) VALUES($1,$2,CURRENT_DATE,10) RETURNING id::text`, periodID, templateID)
	componentID := scalarID(t, pool, `INSERT INTO scheduled_menu_components(scheduled_menu_id,name,sort_order) VALUES($1,'main',0) RETURNING id::text`, scheduledID)
	execSQL(t, pool, `INSERT INTO scheduled_menu_component_materials(scheduled_menu_component_id,material_id,qty_per_portion,unit_id) VALUES($1,$2,1,$3)`, componentID, materialID, unitID)
	return scheduledID
}

func seedPurchaseOrderItem(t *testing.T, pool *pgxpool.Pool, scheduledID, materialID, unitID string, ordered string) string {
	t.Helper()
	requestID := scalarID(t, pool, `INSERT INTO procurement_requests(scheduled_menu_id,status) VALUES($1,'DRAFT') RETURNING id::text`, scheduledID)
	requestItemID := scalarID(t, pool, `INSERT INTO procurement_request_items(procurement_request_id,material_id,gross_requirement_qty,existing_stock_qty,reserved_stock_qty,net_procurement_qty,unit_id) VALUES($1,$2,$3,0,0,$3,$4) RETURNING id::text`, requestID, materialID, ordered, unitID)
	poID := scalarID(t, pool, `INSERT INTO purchase_orders(procurement_request_id,scheduled_menu_id,po_number,delivery_date,status) VALUES($1,$2,$3,CURRENT_DATE+1,'VERIFIED') RETURNING id::text`, requestID, scheduledID, uniqueName("PO"))
	return scalarID(t, pool, `INSERT INTO purchase_order_items(purchase_order_id,procurement_request_item_id,material_id,ordered_qty,unit_id,agreed_unit_price,supplier_name) VALUES($1,$2,$3,$4,$5,1000,'supplier') RETURNING id::text`, poID, requestItemID, materialID, ordered, unitID)
}

func TestConcurrentReceivingSerializesCumulativeStatus(t *testing.T) {
	pool := integrationPool(t)
	materialID, unitID := seedMaterial(t, pool, "0")
	scheduledID := seedScheduledMenu(t, pool, materialID, unitID)
	poItemID := seedPurchaseOrderItem(t, pool, scheduledID, materialID, unitID, "10")
	var poID string
	if err := pool.QueryRow(context.Background(), `SELECT purchase_order_id::text FROM purchase_order_items WHERE id=$1`, poItemID).Scan(&poID); err != nil { t.Fatal(err) }
	pengawas := seedUser(t, pool, "PENGAWAS_BAHAN_BAKU")

	svc := NewReceivingService(repository.NewStore(pool))
	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.CreateReceipt(context.Background(), poID, CreateReceiptInput{ReceivedBy: pengawas, Items: []CreateReceiptItemInput{{PurchaseOrderItemID: poItemID, ReceivedQty: "6"}}})
			errCh <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh { if err != nil { t.Fatal(err) } }

	var qty, status string
	if err := pool.QueryRow(context.Background(), `SELECT qty::text FROM material_stocks WHERE material_id=$1`, materialID).Scan(&qty); err != nil { t.Fatal(err) }
	if err := pool.QueryRow(context.Background(), `SELECT status::text FROM purchase_order_items WHERE id=$1`, poItemID).Scan(&status); err != nil { t.Fatal(err) }
	if qty != "12.0000" || status != "OVER_RECEIVED" { t.Fatalf("got stock=%s status=%s", qty, status) }
}

func TestConcurrentUsageApprovalsApplyStockOnce(t *testing.T) {
	pool := integrationPool(t)
	materialID, unitID := seedMaterial(t, pool, "10")
	scheduledID := seedScheduledMenu(t, pool, materialID, unitID)
	pengawas := seedUser(t, pool, "PENGAWAS_BAHAN_BAKU")
	chef := seedUser(t, pool, "CHEF")
	akuntan := seedUser(t, pool, "AKUNTAN")
	usageID := scalarID(t, pool, `INSERT INTO material_usages(scheduled_menu_id,usage_date,submitted_by,status,submitted_at) VALUES($1,CURRENT_DATE,$2,'WAITING_APPROVAL',NOW()) RETURNING id::text`, scheduledID, pengawas)
	execSQL(t, pool, `INSERT INTO material_usage_items(material_usage_id,material_id,planned_qty,actual_qty,unit_id) VALUES($1,$2,10,4,$3)`, usageID, materialID, unitID)

	svc := NewMaterialUsageService(repository.NewStore(pool))
	start := make(chan struct{})
	errCh := make(chan error, 2)
	for _, approver := range []string{chef, akuntan} {
		go func(id string) {
			<-start
			_, err := svc.Decide(context.Background(), usageID, DecideMaterialUsageInput{ApproverID: id, Decision: "APPROVED"})
			errCh <- err
		}(approver)
	}
	close(start)
	for i := 0; i < 2; i++ { if err := <-errCh; err != nil { t.Fatal(err) } }

	var qty, status string
	var movements int
	pool.QueryRow(context.Background(), `SELECT qty::text FROM material_stocks WHERE material_id=$1`, materialID).Scan(&qty)
	pool.QueryRow(context.Background(), `SELECT status::text FROM material_usages WHERE id=$1`, usageID).Scan(&status)
	pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM stock_movements WHERE reference_type='MATERIAL_USAGE' AND material_id=$1`, materialID).Scan(&movements)
	if qty != "6.0000" || status != "APPROVED" || movements != 1 { t.Fatalf("got stock=%s status=%s movements=%d", qty, status, movements) }
}

func TestConcurrentAdjustmentApprovalsApplyOnce(t *testing.T) {
	pool := integrationPool(t)
	materialID, unitID := seedMaterial(t, pool, "5")
	scheduledID := seedScheduledMenu(t, pool, materialID, unitID)
	pengawas := seedUser(t, pool, "PENGAWAS_BAHAN_BAKU")
	chef := seedUser(t, pool, "CHEF")
	akuntan := seedUser(t, pool, "AKUNTAN")
	opnameID := scalarID(t, pool, `INSERT INTO stock_opnames(scheduled_menu_id,opname_date,performed_by,status) VALUES($1,CURRENT_DATE,$2,'WAITING_ADJUSTMENT_APPROVAL') RETURNING id::text`, scheduledID, pengawas)
	opnameItemID := scalarID(t, pool, `INSERT INTO stock_opname_items(stock_opname_id,material_id,system_qty,physical_qty,unit_id) VALUES($1,$2,5,3,$3) RETURNING id::text`, opnameID, materialID, unitID)
	adjustmentID := scalarID(t, pool, `INSERT INTO stock_adjustments(stock_opname_item_id,material_id,adjustment_qty,reason,submitted_by,status,submitted_at) VALUES($1,$2,-2,'count mismatch',$3,'WAITING_APPROVAL',NOW()) RETURNING id::text`, opnameItemID, materialID, pengawas)

	svc := NewStockOpnameService(repository.NewStore(pool))
	start := make(chan struct{})
	errCh := make(chan error, 2)
	for _, approver := range []string{chef, akuntan} {
		go func(id string) {
			<-start
			_, err := svc.DecideAdjustment(context.Background(), adjustmentID, DecideStockAdjustmentInput{ApproverID: id, Decision: "APPROVED"})
			errCh <- err
		}(approver)
	}
	close(start)
	for i := 0; i < 2; i++ { if err := <-errCh; err != nil { t.Fatal(err) } }

	var qty, adjustmentStatus, opnameStatus string
	var movements int
	pool.QueryRow(context.Background(), `SELECT qty::text FROM material_stocks WHERE material_id=$1`, materialID).Scan(&qty)
	pool.QueryRow(context.Background(), `SELECT status::text FROM stock_adjustments WHERE id=$1`, adjustmentID).Scan(&adjustmentStatus)
	pool.QueryRow(context.Background(), `SELECT status::text FROM stock_opnames WHERE id=$1`, opnameID).Scan(&opnameStatus)
	pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM stock_movements WHERE reference_type='STOCK_ADJUSTMENT' AND reference_id=$1`, adjustmentID).Scan(&movements)
	if qty != "3.0000" || adjustmentStatus != "APPROVED" || opnameStatus != "COMPLETED" || movements != 1 { t.Fatalf("got stock=%s adjustment=%s opname=%s movements=%d", qty, adjustmentStatus, opnameStatus, movements) }
}

func TestConcurrentConditionalStockOutNeverNegative(t *testing.T) {
	pool := integrationPool(t)
	materialID, _ := seedMaterial(t, pool, "5")
	start := make(chan struct{})
	results := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			ct, err := pool.Exec(context.Background(), `UPDATE material_stocks SET qty=qty-4 WHERE material_id=$1 AND qty>=4`, materialID)
			results <- err == nil && ct.RowsAffected() == 1
		}()
	}
	close(start)
	success := 0
	for i := 0; i < 2; i++ { if <-results { success++ } }
	var qty string
	pool.QueryRow(context.Background(), `SELECT qty::text FROM material_stocks WHERE material_id=$1`, materialID).Scan(&qty)
	if success != 1 || qty != "1.0000" { t.Fatalf("success=%d stock=%s", success, qty) }
}
