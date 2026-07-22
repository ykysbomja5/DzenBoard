package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"nasha_igrushka/database"
)

type SyncStats struct {
	TotalCategories   int    `json:"total_categories"`
	TotalProducts     int    `json:"total_products"`
	TotalOrders       int    `json:"total_orders"`
	UnexportedOrders  int    `json:"unexported_orders"`
	LatestOrderNumber string `json:"latest_order_number"`
}

// AdminDashboardHandler renders the admin panel page
func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:           "Панель управления - «Наша Игрушка»",
		MetaDescription: "Просмотр заказов, статистика синхронизации с 1С и управление каталогом товаров.",
		Categories:      cats,
		ActiveTab:       "admin",
	}

	renderTemplate(w, "admin", data)
}

// AdminStatsHandler returns JSON stats for the admin panel
func AdminStatsHandler(w http.ResponseWriter, r *http.Request) {
	var stats SyncStats

	err := database.DB.QueryRow("SELECT COUNT(*) FROM categories;").Scan(&stats.TotalCategories)
	if err != nil {
		log.Printf("Stats error: %v", err)
	}

	err = database.DB.QueryRow("SELECT COUNT(*) FROM products;").Scan(&stats.TotalProducts)
	if err != nil {
		log.Printf("Stats error: %v", err)
	}

	err = database.DB.QueryRow("SELECT COUNT(*) FROM orders;").Scan(&stats.TotalOrders)
	if err != nil {
		log.Printf("Stats error: %v", err)
	}

	err = database.DB.QueryRow("SELECT COUNT(*) FROM orders WHERE c1_exported = FALSE;").Scan(&stats.UnexportedOrders)
	if err != nil {
		log.Printf("Stats error: %v", err)
	}

	err = database.DB.QueryRow("SELECT order_number FROM orders ORDER BY created_at DESC LIMIT 1;").Scan(&stats.LatestOrderNumber)
	if err != nil {
		stats.LatestOrderNumber = "Нет заказов"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// AdminOrdersHandler lists all orders in JSON format
func AdminOrdersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT id, order_number, customer_name, customer_email, customer_phone, 
		       delivery_method, payment_method, comment, total_price, status, c1_exported, created_at 
		FROM orders 
		ORDER BY created_at DESC;`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type OrderItemJSON struct {
		Name     string  `json:"name"`
		Quantity int     `json:"qty"`
		Price    float64 `json:"price"`
	}

	type OrderJSON struct {
		ID             string          `json:"id"`
		OrderNumber    string          `json:"order_number"`
		CustomerName   string          `json:"customer_name"`
		CustomerEmail  string          `json:"customer_email"`
		CustomerPhone  string          `json:"customer_phone"`
		DeliveryMethod string          `json:"delivery_method"`
		PaymentMethod  string          `json:"payment_method"`
		Comment        string          `json:"comment"`
		TotalPrice     float64         `json:"total_price"`
		Status         string          `json:"status"`
		C1Exported     bool            `json:"c1_exported"`
		CreatedAt      string          `json:"created_at"`
		Items          []OrderItemJSON `json:"items"`
	}

	var orders []OrderJSON
	for rows.Next() {
		var o OrderJSON
		var createdAtTime interface{} // postgres timestamp
		err := rows.Scan(&o.ID, &o.OrderNumber, &o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
			&o.DeliveryMethod, &o.PaymentMethod, &o.Comment, &o.TotalPrice, &o.Status, &o.C1Exported, &createdAtTime)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		if t, ok := createdAtTime.(time.Time); ok {
			o.CreatedAt = t.Format("02.01.2006 15:04")
		} else {
			o.CreatedAt = "Неизвестно"
		}

		// Fetch items for this order
		itemRows, err := database.DB.Query("SELECT name, quantity, price FROM order_items WHERE order_id = $1;", o.ID)
		if err == nil {
			for itemRows.Next() {
				var item OrderItemJSON
				if err := itemRows.Scan(&item.Name, &item.Quantity, &item.Price); err == nil {
					o.Items = append(o.Items, item)
				}
			}
			itemRows.Close()
		}

		orders = append(orders, o)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// AdminUpdateOrderStatusHandler updates order status
func AdminUpdateOrderStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	_, err := database.DB.Exec("UPDATE orders SET status = $1 WHERE id = $2;", req.Status, req.OrderID)
	if err != nil {
		http.Error(w, "Database error: failed to update status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}
