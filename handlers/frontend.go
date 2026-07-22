package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"nasha_igrushka/database"
	"nasha_igrushka/models"
)

// Common page data wrapper
type PageData struct {
	Title           string
	MetaDescription string
	Categories      []models.Category
	CurrentCategory models.Category
	Products        []models.Product
	Product         models.Product
	BlogPosts       []models.BlogPost
	BlogPost        models.BlogPost
	Reviews         []models.Review
	RelatedProducts []models.Product
	SearchQuery     string
	ActiveTab       string
	CartItemsCount  int
	PriceMin        float64
	PriceMax        float64
	CurrentSort     string
	InStockOnly     bool
}

func renderTemplate(w http.ResponseWriter, tmplName string, data PageData) {
	tmplFiles := []string{
		"templates/layout.html",
		"templates/" + tmplName + ".html",
	}
	
	// Create templates with custom function map
	funcMap := template.FuncMap{
		"seq": func(start, end int) []int {
			var s []int
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	t := template.New("layout.html").Funcs(funcMap)
	t, err := t.ParseFiles(tmplFiles...)
	if err != nil {
		log.Printf("Template compilation error: %v", err)
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = t.ExecuteTemplate(w, "layout", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Execution error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Fetch all active categories for header/sidebar navigation
func fetchCategories() ([]models.Category, error) {
	rows, err := database.DB.Query("SELECT id, parent_id, name, slug, sort_order, is_active FROM categories WHERE is_active = TRUE ORDER BY sort_order ASC, name ASC;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		var parentID sql.NullString
		if err := rows.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &c.SortOrder, &c.IsActive); err == nil {
			if parentID.Valid {
				c.ParentID = parentID.String
			}
			categories = append(categories, c)
		}
	}
	return categories, nil
}

// HomeHandler renders the homepage
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch featured products
	rows, err := database.DB.Query("SELECT id, name, price, image, slug, quantity FROM products WHERE is_active = TRUE AND is_featured = TRUE LIMIT 8;")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Image, &p.Slug, &p.Quantity); err == nil {
			products = append(products, p)
		}
	}

	// Fetch latest blog posts
	blogRows, err := database.DB.Query("SELECT id, title, slug, summary, image, created_at FROM blog_posts WHERE is_active = TRUE ORDER BY created_at DESC LIMIT 3;")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer blogRows.Close()

	var blogs []models.BlogPost
	for blogRows.Next() {
		var b models.BlogPost
		if err := blogRows.Scan(&b.ID, &b.Title, &b.Slug, &b.Summary, &b.Image, &b.CreatedAt); err == nil {
			blogs = append(blogs, b)
		}
	}

	data := PageData{
		Title:           "Интернет-магазин детских игрушек «Наша Игрушка»",
		MetaDescription: "Широкий выбор детских игрушек в наличии: развивающие игры, мягкие игрушки, конструкторы, детский транспорт. Доставка и гарантия.",
		Categories:      cats,
		Products:        products,
		BlogPosts:       blogs,
		ActiveTab:       "home",
	}

	renderTemplate(w, "home", data)
}

// CatalogHandler handles categories and product listings (filters, search, sorting)
func CatalogHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Route parameters
	catSlug := r.URL.Query().Get("category")
	searchQuery := r.URL.Query().Get("q")
	sortOpt := r.URL.Query().Get("sort")
	inStockStr := r.URL.Query().Get("instock")
	priceMinStr := r.URL.Query().Get("price_min")
	priceMaxStr := r.URL.Query().Get("price_max")

	// Fallback to default sort
	if sortOpt == "" {
		sortOpt = "newness"
	}

	var currentCategory models.Category
	if catSlug != "" {
		err = database.DB.QueryRow("SELECT id, parent_id, name, slug FROM categories WHERE slug = $1 AND is_active = TRUE;", catSlug).
			Scan(&currentCategory.ID, &currentCategory.ParentID, &currentCategory.Name, &currentCategory.Slug)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Base query construction
	query := "SELECT id, name, price, image, slug, quantity FROM products WHERE is_active = TRUE"
	var args []interface{}
	argCount := 1

	if currentCategory.ID != "" {
		query += fmt.Sprintf(" AND category_id = $%d", argCount)
		args = append(args, currentCategory.ID)
		argCount++
	}

	if searchQuery != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+searchQuery+"%")
		argCount++
	}

	inStockOnly := false
	if inStockStr == "1" || inStockStr == "true" {
		query += " AND quantity > 0"
		inStockOnly = true
	}

	var pMin, pMax float64
	if priceMinStr != "" {
		if val, err := strconv.ParseFloat(priceMinStr, 64); err == nil {
			query += fmt.Sprintf(" AND price >= $%d", argCount)
			args = append(args, val)
			argCount++
			pMin = val
		}
	}
	if priceMaxStr != "" {
		if val, err := strconv.ParseFloat(priceMaxStr, 64); err == nil {
			query += fmt.Sprintf(" AND price <= $%d", argCount)
			args = append(args, val)
			argCount++
			pMax = val
		}
	}

	// Sorting
	switch sortOpt {
	case "price_asc":
		query += " ORDER BY price ASC"
	case "price_desc":
		query += " ORDER BY price DESC"
	case "name_asc":
		query += " ORDER BY name ASC"
	case "newness":
		query += " ORDER BY created_at DESC"
	default:
		query += " ORDER BY created_at DESC"
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Image, &p.Slug, &p.Quantity); err == nil {
			products = append(products, p)
		}
	}

	title := "Каталог детских игрушек"
	if currentCategory.Name != "" {
		title = currentCategory.Name + " - купить в магазине «Наша Игрушка»"
	} else if searchQuery != "" {
		title = "Результаты поиска: " + searchQuery
	}

	data := PageData{
		Title:           title,
		MetaDescription: "Каталог товаров. Огромный ассортимент игрушек по отличным ценам. Удобные фильтры и сортировка.",
		Categories:      cats,
		CurrentCategory: currentCategory,
		Products:        products,
		SearchQuery:     searchQuery,
		PriceMin:        pMin,
		PriceMax:        pMax,
		CurrentSort:     sortOpt,
		InStockOnly:     inStockOnly,
		ActiveTab:       "catalog",
	}

	renderTemplate(w, "category", data)
}

// ProductHandler renders product detail page
func ProductHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var p models.Product
	err = database.DB.QueryRow(`
		SELECT id, c1_id, name, sku, slug, description, price, quantity, image, category_id, created_at 
		FROM products 
		WHERE slug = $1 AND is_active = TRUE;`, slug).
		Scan(&p.ID, &p.C1ID, &p.Name, &p.SKU, &p.Slug, &p.Description, &p.Price, &p.Quantity, &p.Image, &p.CategoryID, &p.CreatedAt)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch additional images
	imgRows, err := database.DB.Query("SELECT image_path FROM product_images WHERE product_id = $1 ORDER BY sort_order ASC;", p.ID)
	if err == nil {
		defer imgRows.Close()
		for imgRows.Next() {
			var imgPath string
			if err := imgRows.Scan(&imgPath); err == nil {
				p.Images = append(p.Images, imgPath)
			}
		}
	}

	// Fetch attributes
	attrRows, err := database.DB.Query("SELECT name, value FROM product_attributes WHERE product_id = $1 ORDER BY id ASC;", p.ID)
	if err == nil {
		defer attrRows.Close()
		for attrRows.Next() {
			var attr models.ProductAttribute
			attr.ProductID = p.ID
			if err := attrRows.Scan(&attr.Name, &attr.Value); err == nil {
				p.Attributes = append(p.Attributes, attr)
			}
		}
	}

	// Fetch reviews
	revRows, err := database.DB.Query("SELECT id, product_id, author, rating, text, created_at FROM reviews WHERE product_id = $1 AND is_approved = TRUE ORDER BY created_at DESC;", p.ID)
	var reviews []models.Review
	if err == nil {
		defer revRows.Close()
		for revRows.Next() {
			var rev models.Review
			if err := revRows.Scan(&rev.ID, &rev.ProductID, &rev.Author, &rev.Rating, &rev.Text, &rev.CreatedAt); err == nil {
				reviews = append(reviews, rev)
			}
		}
	}

	// Fetch related products (same category)
	relRows, err := database.DB.Query("SELECT id, name, price, image, slug, quantity FROM products WHERE category_id = $1 AND id <> $2 AND is_active = TRUE LIMIT 4;", p.CategoryID, p.ID)
	var related []models.Product
	if err == nil {
		defer relRows.Close()
		for relRows.Next() {
			var rp models.Product
			if err := relRows.Scan(&rp.ID, &rp.Name, &rp.Price, &rp.Image, &rp.Slug, &rp.Quantity); err == nil {
				related = append(related, rp)
			}
		}
	}

	data := PageData{
		Title:           p.Name + " - купить в магазине «Наша Игрушка»",
		MetaDescription: p.Name + ": описание, характеристики, отзывы, цена. Заказывайте прямо сейчас!",
		Categories:      cats,
		Product:         p,
		Reviews:         reviews,
		RelatedProducts: related,
		ActiveTab:       "catalog",
	}

	renderTemplate(w, "product", data)
}

// CartHandler renders cart summary page
func CartHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:           "Корзина покупателя - «Наша Игрушка»",
		MetaDescription: "Список выбранных товаров в вашей корзине. Отредактируйте количество и перейдите к оформлению заказа.",
		Categories:      cats,
		ActiveTab:       "cart",
	}
	renderTemplate(w, "cart", data)
}

// CheckoutHandler renders checkout page
func CheckoutHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:           "Оформление заказа - «Наша Игрушка»",
		MetaDescription: "Заполните контактные данные, выберите способ доставки и оплаты для подтверждения заказа.",
		Categories:      cats,
		ActiveTab:       "cart",
	}
	renderTemplate(w, "checkout", data)
}

// SubmitOrderHandler processes AJAX order creation requests
type OrderSubmission struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Delivery string `json:"delivery"`
	Payment  string `json:"payment"`
	Comment  string `json:"comment"`
	Cart     []struct {
		ID       string  `json:"id"`
		Quantity int     `json:"qty"`
		Price    float64 `json:"price"`
	} `json:"cart"`
}

func SubmitOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var submission OrderSubmission
	err := json.NewDecoder(r.Body).Decode(&submission)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if submission.Name == "" || submission.Phone == "" || len(submission.Cart) == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	orderID := fmt.Sprintf("ord-%d", time.Now().UnixNano())
	orderNumber := fmt.Sprintf("NI-%s", time.Now().Format("20060102-150405"))

	var totalPrice float64
	var items []models.OrderItem

	// Process cart items, double-checking prices from DB to avoid client tamper
	for _, cItem := range submission.Cart {
		var pName string
		var pPrice float64
		var pQty int
		err := tx.QueryRow("SELECT name, price, quantity FROM products WHERE id = $1;", cItem.ID).Scan(&pName, &pPrice, &pQty)
		if err != nil {
			http.Error(w, "Product not found: "+cItem.ID, http.StatusBadRequest)
			return
		}

		itemPrice := pPrice
		totalPrice += itemPrice * float64(cItem.Quantity)

		itemID := fmt.Sprintf("%s-item-%s", orderID, cItem.ID)
		item := models.OrderItem{
			ID:        itemID,
			OrderID:   orderID,
			ProductID: cItem.ID,
			Name:      pName,
			Quantity:  cItem.Quantity,
			Price:     itemPrice,
		}
		items = append(items, item)

		// Reduce quantity in stock
		newQty := pQty - cItem.Quantity
		if newQty < 0 {
			newQty = 0 // allow oversell but set stock to 0 or block
		}
		_, err = tx.Exec("UPDATE products SET quantity = $1 WHERE id = $2;", newQty, cItem.ID)
		if err != nil {
			http.Error(w, "Database error: updating quantity", http.StatusInternalServerError)
			return
		}
	}

	// Insert order
	_, err = tx.Exec(`
		INSERT INTO orders (id, order_number, customer_name, customer_email, customer_phone, delivery_method, payment_method, comment, total_price, status, c1_exported)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'New', FALSE);`,
		orderID, orderNumber, submission.Name, submission.Email, submission.Phone, submission.Delivery, submission.Payment, submission.Comment, totalPrice)
	if err != nil {
		log.Printf("Order creation failed: %v", err)
		http.Error(w, "Database error: order creation failed", http.StatusInternalServerError)
		return
	}

	// Insert order items
	for _, item := range items {
		_, err = tx.Exec(`
			INSERT INTO order_items (id, order_id, product_id, name, quantity, price)
			VALUES ($1, $2, $3, $4, $5, $6);`,
			item.ID, item.OrderID, item.ProductID, item.Name, item.Quantity, item.Price)
		if err != nil {
			log.Printf("Order item creation failed: %v", err)
			http.Error(w, "Database error: item creation failed", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "success",
		"order_id":     orderID,
		"order_number": orderNumber,
		"total":        fmt.Sprintf("%.2f", totalPrice),
	})
}

// SubmitReviewHandler handles AJAX product reviews
type ReviewSubmission struct {
	ProductID string `json:"product_id"`
	Author    string `json:"author"`
	Rating    int    `json:"rating"`
	Text      string `json:"text"`
}

func SubmitReviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var sub ReviewSubmission
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if sub.ProductID == "" || sub.Author == "" || sub.Text == "" || sub.Rating < 1 || sub.Rating > 5 {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	revID := fmt.Sprintf("rev-%d", time.Now().UnixNano())
	_, err := database.DB.Exec(`
		INSERT INTO reviews (id, product_id, author, rating, text, is_approved)
		VALUES ($1, $2, $3, $4, $5, TRUE);`, // Auto-approve for testing simplicity
		revID, sub.ProductID, sub.Author, sub.Rating, sub.Text)

	if err != nil {
		log.Printf("Review failed: %v", err)
		http.Error(w, "Database error: review submit failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

// BlogHandler lists articles
func BlogHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := database.DB.Query("SELECT id, title, slug, summary, image, created_at FROM blog_posts WHERE is_active = TRUE ORDER BY created_at DESC;")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var blogs []models.BlogPost
	for rows.Next() {
		var b models.BlogPost
		if err := rows.Scan(&b.ID, &b.Title, &b.Slug, &b.Summary, &b.Image, &b.CreatedAt); err == nil {
			blogs = append(blogs, b)
		}
	}

	data := PageData{
		Title:           "Блог магазина игрушек - Советы родителям",
		MetaDescription: "Полезные обзоры детских игрушек, советы по выбору подарков и статьи о развитии детей на нашем сайте.",
		Categories:      cats,
		BlogPosts:       blogs,
		ActiveTab:       "blog",
	}

	renderTemplate(w, "blog", data)
}

// BlogPostHandler displays single article
func BlogPostHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var b models.BlogPost
	err = database.DB.QueryRow("SELECT id, title, slug, summary, content, image, created_at FROM blog_posts WHERE slug = $1 AND is_active = TRUE;", slug).
		Scan(&b.ID, &b.Title, &b.Slug, &b.Summary, &b.Content, &b.Image, &b.CreatedAt)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:           b.Title + " - Блог «Наша Игрушка»",
		MetaDescription: b.Summary,
		Categories:      cats,
		BlogPost:        b,
		ActiveTab:       "blog",
	}

	renderTemplate(w, "blog_post", data)
}

// ContactsHandler renders contacts details page
func ContactsHandler(w http.ResponseWriter, r *http.Request) {
	cats, err := fetchCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:           "Контакты магазина игрушек «Наша Игрушка»",
		MetaDescription: "Адрес магазина: г. Луганск, ул. Оборонная, 9 (напротив стадиона Авангард). Телефоны менеджеров, часы работы.",
		Categories:      cats,
		ActiveTab:       "contacts",
	}

	renderTemplate(w, "contacts", data)
}
