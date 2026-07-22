package models

import "time"

type Category struct {
	ID        string `json:"id"`        // UUID from 1C or custom
	ParentID  string `json:"parent_id"` // Empty if root
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

type Product struct {
	ID          string             `json:"id"` // Generated UUID or 1C ID
	C1ID        string             `json:"c1_id"`
	Name        string             `json:"name"`
	SKU         string             `json:"sku"`
	Slug        string             `json:"slug"`
	Description string             `json:"description"`
	Price       float64            `json:"price"`
	Quantity    int                `json:"quantity"`
	Image       string             `json:"image"` // Main image path
	CategoryID  string             `json:"category_id"`
	IsActive    bool               `json:"is_active"`
	IsFeatured  bool               `json:"is_featured"`
	CreatedAt   time.Time          `json:"created_at"`
	Images      []string           `json:"images"`     // Additional images
	Attributes  []ProductAttribute `json:"attributes"` // Attributes/Requisites
}

type ProductAttribute struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Value     string `json:"value"`
}

type Order struct {
	ID             string      `json:"id"`
	OrderNumber    string      `json:"order_number"`
	CustomerName   string      `json:"customer_name"`
	CustomerEmail  string      `json:"customer_email"`
	CustomerPhone  string      `json:"customer_phone"`
	DeliveryMethod string      `json:"delivery_method"`
	PaymentMethod  string      `json:"payment_method"`
	Comment        string      `json:"comment"`
	TotalPrice     float64     `json:"total_price"`
	Status         string      `json:"status"` // e.g. "Pending", "Processing", "Completed", "Canceled"
	C1Exported     bool        `json:"c1_exported"`
	CreatedAt      time.Time   `json:"created_at"`
	Items          []OrderItem `json:"items"`
}

type OrderItem struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"order_id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type BlogPost struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Summary   string    `json:"summary"`
	Content   string    `json:"content"`
	Image     string    `json:"image"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Review struct {
	ID         string    `json:"id"`
	ProductID  string    `json:"product_id"`
	Author     string    `json:"author"`
	Rating     int       `json:"rating"`
	Text       string    `json:"text"`
	IsApproved bool      `json:"is_approved"`
	CreatedAt  time.Time `json:"created_at"`
}
