package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func InitDB() error {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "nasha_igrushka")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// 1. Connect to default 'postgres' database to ensure our database exists
	serverConnStr := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=%s", user, pass, host, port, sslmode)
	sysDB, err := sql.Open("pgx", serverConnStr)
	if err != nil {
		log.Printf("Warning: Failed to connect to postgres system DB: %v", err)
	} else {
		defer sysDB.Close()
		var exists bool
		query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s')", dbname)
		err = sysDB.QueryRow(query).Scan(&exists)
		if err != nil {
			log.Printf("Warning: Failed to check if database exists: %v", err)
		} else if !exists {
			log.Printf("Database '%s' does not exist. Creating it...", dbname)
			_, err = sysDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
			if err != nil {
				return fmt.Errorf("failed to create database '%s': %w", dbname, err)
			}
			log.Printf("Database '%s' created successfully.", dbname)
		}
	}

	// 2. Connect to the application database
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbname, sslmode)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", dbname, err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Ping database
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	log.Println("Database connection established successfully.")

	// Run Migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Seed Data
	if err := seedData(); err != nil {
		log.Printf("Warning: Failed to seed data: %v", err)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func runMigrations() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id VARCHAR(100) PRIMARY KEY,
			parent_id VARCHAR(100),
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			sort_order INT DEFAULT 0,
			is_active BOOLEAN DEFAULT TRUE
		);`,
		`CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(100) PRIMARY KEY,
			c1_id VARCHAR(100) UNIQUE,
			name VARCHAR(255) NOT NULL,
			sku VARCHAR(100) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			price NUMERIC(10, 2) DEFAULT 0.00,
			quantity INT DEFAULT 0,
			image VARCHAR(255),
			category_id VARCHAR(100) REFERENCES categories(id) ON DELETE SET NULL,
			is_active BOOLEAN DEFAULT TRUE,
			is_featured BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS product_images (
			id VARCHAR(100) PRIMARY KEY,
			product_id VARCHAR(100) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			image_path VARCHAR(255) NOT NULL,
			sort_order INT DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS product_attributes (
			id SERIAL PRIMARY KEY,
			product_id VARCHAR(100) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			value TEXT NOT NULL,
			UNIQUE(product_id, name)
		);`,
		`CREATE TABLE IF NOT EXISTS orders (
			id VARCHAR(100) PRIMARY KEY,
			order_number VARCHAR(100) UNIQUE NOT NULL,
			customer_name VARCHAR(255) NOT NULL,
			customer_email VARCHAR(255) NOT NULL,
			customer_phone VARCHAR(100) NOT NULL,
			delivery_method VARCHAR(150) NOT NULL,
			payment_method VARCHAR(150) NOT NULL,
			comment TEXT,
			total_price NUMERIC(10, 2) NOT NULL,
			status VARCHAR(50) DEFAULT 'New',
			c1_exported BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id VARCHAR(100) PRIMARY KEY,
			order_id VARCHAR(100) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			product_id VARCHAR(100) REFERENCES products(id) ON DELETE SET NULL,
			name VARCHAR(255) NOT NULL,
			quantity INT NOT NULL,
			price NUMERIC(10, 2) NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS blog_posts (
			id VARCHAR(100) PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			summary TEXT,
			content TEXT,
			image VARCHAR(255),
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS reviews (
			id VARCHAR(100) PRIMARY KEY,
			product_id VARCHAR(100) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			author VARCHAR(255) NOT NULL,
			rating INT NOT NULL CHECK(rating >= 1 AND rating <= 5),
			text TEXT NOT NULL,
			is_approved BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for i, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			return fmt.Errorf("migration query %d failed: %w", i, err)
		}
	}
	log.Println("Database migrations applied successfully.")
	return nil
}

func seedData() error {
	// 1. Seed categories
	categories := []struct {
		ID       string
		ParentID string
		Name     string
		Slug     string
	}{
		{"cat-constructors", "", "Конструкторы", "constructors"},
		{"cat-soft-toys", "", "Мягкие игрушки", "soft-toys"},
		{"cat-transport", "", "Детский транспорт", "transport"},
		{"cat-creativity", "", "Творчество", "creativity"},
	}

	for _, c := range categories {
		_, err := DB.Exec(`
			INSERT INTO categories (id, parent_id, name, slug, sort_order, is_active)
			VALUES ($1, $2, $3, $4, 0, TRUE)
			ON CONFLICT (id) DO NOTHING;`, c.ID, c.ParentID, c.Name, c.Slug)
		if err != nil {
			return err
		}
	}

	// 2. Seed products
	products := []struct {
		ID         string
		C1ID       string
		Name       string
		SKU        string
		Slug       string
		Desc       string
		Price      float64
		Qty        int
		Image      string
		CategoryID string
		Featured   bool
	}{
		{
			"prod-lego-castle",
			"c1-lego-castle",
			"Конструктор Замок Рыцарей",
			"LEGO-001",
			"lego-knight-castle",
			"Великолепный конструктор для моделирования средневекового замка. В комплекте 1200 деталей, включая 5 мини-фигурок рыцарей, коней и катапульту.",
			2499.00,
			15,
			"/static/images/lego_castle.jpg",
			"cat-constructors",
			true,
		},
		{
			"prod-teddy-bear",
			"c1-teddy-bear",
			"Медвежонок Барни 50см",
			"SOFT-001",
			"teddy-bear-barney",
			"Мягкий плюшевый мишка Барни. Высота 50 см, гипоаллергенный наполнитель, очень приятный на ощупь. Отличный подарок для ребенка любого возраста.",
			990.00,
			25,
			"/static/images/teddy_bear.jpg",
			"cat-soft-toys",
			true,
		},
		{
			"prod-scooter",
			"c1-scooter",
			"Самокат трехколесный Scooter Pro",
			"TR-001",
			"scooter-pro-3wheel",
			"Безопасный трехколесный самокат со светящимися колесами и регулируемой высотой руля. Выдерживает нагрузку до 50 кг.",
			1850.00,
			8,
			"/static/images/scooter.jpg",
			"cat-transport",
			true,
		},
		{
			"prod-paint-kit",
			"c1-paint-kit",
			"Набор юного художника (86 предметов)",
			"CRT-001",
			"young-artist-paint-kit",
			"Большой художественный набор в чемоданчике: фломастеры, карандаши, краски, пастель, восковые мелки, точилка, клей. Все, что нужно для творчества.",
			750.00,
			30,
			"/static/images/paint_kit.jpg",
			"cat-creativity",
			false,
		},
		{
			"prod-wooden-train",
			"c1-wooden-train",
			"Деревянная железная дорога Classic",
			"WOOD-001",
			"wooden-train-classic",
			"Экологичная деревянная железная дорога. Способствует развитию моторики и пространственного мышления. Содержит рельсы, поезд и декорации.",
			1600.00,
			12,
			"/static/images/wooden_train.jpg",
			"cat-constructors",
			true,
		},
	}

	for _, p := range products {
		_, err := DB.Exec(`
			INSERT INTO products (id, c1_id, name, sku, slug, description, price, quantity, image, category_id, is_active, is_featured)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, TRUE, $11)
			ON CONFLICT (id) DO UPDATE SET 
				price = EXCLUDED.price, 
				quantity = EXCLUDED.quantity, 
				image = EXCLUDED.image,
				is_featured = EXCLUDED.is_featured;`,
			p.ID, p.C1ID, p.Name, p.SKU, p.Slug, p.Desc, p.Price, p.Qty, p.Image, p.CategoryID, p.Featured)
		if err != nil {
			return err
		}

		// Seed some default product attributes
		attributes := map[string]string{
			"Возраст":  "3+",
			"Материал": "Пластик / Дерево",
			"Бренд":    "ToyLand",
		}
		for name, value := range attributes {
			_, err = DB.Exec(`
				INSERT INTO product_attributes (product_id, name, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (product_id, name) DO NOTHING;`, p.ID, name, value)
			if err != nil {
				return err
			}
		}
	}

	// 3. Seed blog posts
	blogs := []struct {
		ID      string
		Title   string
		Slug    string
		Summary string
		Content string
		Image   string
	}{
		{
			"blog-1",
			"Как выбрать развивающую игрушку для ребенка?",
			"how-to-choose-developing-toy",
			"В этой статье мы расскажем, на что обратить внимание при выборе развивающих игрушек для детей разного возраста.",
			"<p>Выбор игрушки — ответственный шаг для родителей. До 1 года ребенку важны текстуры и цвета (погремушки, мобили). От 1 до 3 лет развиваются моторика и речь (пирамидки, сортеры, кубики). От 3 лет и старше актуальны конструкторы, ролевые игры и наборы для творчества.</p><p>Главное правило: игрушка должна быть безопасной, сделанной из нетоксичных материалов и соответствующей возрастной категории ребенка.</p>",
			"/static/images/blog_1.jpg",
		},
		{
			"blog-2",
			"Польза конструкторов для детского развития",
			"benefits-of-constructors",
			"Почему конструкторы считаются одними из самых лучших развивающих игрушек? Разбираем влияние на логику и моторику.",
			"<p>Конструкторы развивают не только мелкую моторику пальцев, но и логическое мышление, воображение и усидчивость. Собирая замки, машины или роботов, ребенок учится пространственному моделированию, анализирует пропорции и сочетаемость элементов.</p><p>Более того, совместная сборка с родителями или сверстниками помогает развивать коммуникативные навыки и командную работу.</p>",
			"/static/images/blog_2.jpg",
		},
	}

	for _, b := range blogs {
		_, err := DB.Exec(`
			INSERT INTO blog_posts (id, title, slug, summary, content, image, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE)
			ON CONFLICT (id) DO NOTHING;`, b.ID, b.Title, b.Slug, b.Summary, b.Content, b.Image)
		if err != nil {
			return err
		}
	}

	return nil
}
