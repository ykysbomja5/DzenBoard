package exchange

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nasha_igrushka/database"
	"nasha_igrushka/models"
)

// Transliteration map for Russian to English slugs
var cyrillicToLatin = map[string]string{
	"а": "a", "б": "b", "в": "v", "г": "g", "д": "d", "е": "e", "ё": "yo", "ж": "zh",
	"з": "z", "и": "i", "й": "y", "к": "k", "л": "l", "м": "m", "н": "n", "о": "o",
	"п": "p", "р": "r", "с": "s", "т": "t", "у": "u", "ф": "f", "х": "h", "ц": "c",
	"ч": "ch", "ш": "sh", "щ": "shh", "ъ": "", "ы": "y", "ь": "", "э": "e", "ю": "yu",
	"я": "ya",
}

// Slugify transliterates and formats strings to be URL-friendly
func Slugify(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	for _, r := range s {
		char := string(r)
		if val, ok := cyrillicToLatin[char]; ok {
			sb.WriteString(val)
		} else {
			sb.WriteString(char)
		}
	}
	res := sb.String()
	reg := regexp.MustCompile("[^a-z0-9]+")
	res = reg.ReplaceAllString(res, "-")
	res = strings.Trim(res, "-")
	if len(res) == 0 {
		res = fmt.Sprintf("item-%d", rand.Intn(100000))
	}
	return res
}

// CommerceML XML Structs
type CMLImport struct {
	XMLName    xml.Name `xml:"КоммерческаяИнформация"`
	Classifier struct {
		Groups CMLGroups `xml:"Группы"`
	} `xml:"Классификатор"`
	Catalog struct {
		Products struct {
			Product []CMLProduct `xml:"Товар"`
		} `xml:"Товары"`
	} `xml:"Каталог"`
}

type CMLGroups struct {
	Group []CMLGroup `xml:"Группа"`
}

type CMLGroup struct {
	ID     string    `xml:"Ид"`
	Name   string    `xml:"Наименование"`
	Groups CMLGroups `xml:"Группы"`
}

type CMLProduct struct {
	ID          string `xml:"Ид"`
	Article     string `xml:"Артикул"`
	Name        string `xml:"Наименование"`
	Description string `xml:"Описание"`
	Groups      struct {
		ID []string `xml:"Ид"`
	} `xml:"Группы"`
	Image      []string `xml:"Картинка"`
	Requisites struct {
		Requisite []CMLRequisite `xml:"ЗначениеРеквизита"`
	} `xml:"ЗначенияРеквизитов"`
}

type CMLRequisite struct {
	Name  string `xml:"Наименование"`
	Value string `xml:"Значение"`
}

type CMLOffers struct {
	XMLName       xml.Name `xml:"КоммерческаяИнформация"`
	OffersPackage struct {
		Offers struct {
			Offer []CMLOffer `xml:"Предложение"`
		} `xml:"Предложения"`
	} `xml:"ПакетПредложений"`
}

type CMLOffer struct {
	ID       string `xml:"Ид"`
	Name     string `xml:"Наименование"`
	Quantity int    `xml:"Количество"`
	Prices   struct {
		Price []CMLPrice `xml:"Цена"`
	} `xml:"Цены"`
}

type CMLPrice struct {
	PricePerUnit string `xml:"ЦенаЗаЕдиницу"`
	Currency     string `xml:"Валюта"`
}

// ParseImportXML processes import.xml and updates categories and products in the DB
func ParseImportXML(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open XML file: %w", err)
	}
	defer file.Close()

	var data CMLImport
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode import.xml: %w", err)
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Process categories recursively
	var importGroups func(groups CMLGroups, parentID string) error
	importGroups = func(groups CMLGroups, parentID string) error {
		for _, g := range groups.Group {
			slug := Slugify(g.Name)

			var parentVal sql.NullString
			if parentID != "" {
				parentVal.String = parentID
				parentVal.Valid = true
			}

			_, err := tx.Exec(`
				INSERT INTO categories (id, parent_id, name, slug, sort_order, is_active)
				VALUES ($1, $2, $3, $4, 0, TRUE)
				ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, parent_id = EXCLUDED.parent_id;`,
				g.ID, parentVal, g.Name, slug)
			if err != nil {
				return fmt.Errorf("failed to save category %s: %w", g.Name, err)
			}

			if len(g.Groups.Group) > 0 {
				if err := importGroups(g.Groups, g.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := importGroups(data.Classifier.Groups, ""); err != nil {
		return err
	}

	// 2. Process products
	for _, p := range data.Catalog.Products.Product {
		slug := Slugify(p.Name)

		var catID sql.NullString
		if len(p.Groups.ID) > 0 {
			catID.String = p.Groups.ID[0]
			catID.Valid = true
		}

		mainImg := ""
		if len(p.Image) > 0 {
			// CommerceML image path, normalize separators to slash
			mainImg = "/" + filepath.ToSlash(filepath.Join("uploads", p.Image[0]))
		}

		// Insert or update product
		_, err := tx.Exec(`
			INSERT INTO products (id, c1_id, name, sku, slug, description, image, category_id, is_active, is_featured)
			VALUES ($1, $1, $2, $3, $4, $5, $6, $7, TRUE, FALSE)
			ON CONFLICT (id) DO UPDATE SET 
				name = EXCLUDED.name, 
				sku = EXCLUDED.sku, 
				description = EXCLUDED.description, 
				image = CASE WHEN EXCLUDED.image <> '' THEN EXCLUDED.image ELSE products.image END, 
				category_id = EXCLUDED.category_id;`,
			p.ID, p.Name, p.Article, slug, p.Description, mainImg, catID)
		if err != nil {
			return fmt.Errorf("failed to save product %s: %w", p.Name, err)
		}

		// Save product additional images
		if len(p.Image) > 1 {
			for idx, img := range p.Image {
				imgUrl := "/" + filepath.ToSlash(filepath.Join("uploads", img))
				imgID := fmt.Sprintf("%s-img-%d", p.ID, idx)
				_, err = tx.Exec(`
					INSERT INTO product_images (id, product_id, image_path, sort_order)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (id) DO NOTHING;`,
					imgID, p.ID, imgUrl, idx)
				if err != nil {
					log.Printf("Warning: failed to save additional image %s for product %s: %v", img, p.Name, err)
				}
			}
		}

		// Process requisites / attributes
		for _, req := range p.Requisites.Requisite {
			if req.Name == "" || req.Value == "" {
				continue
			}
			_, err = tx.Exec(`
				INSERT INTO product_attributes (product_id, name, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (product_id, name) DO UPDATE SET value = EXCLUDED.value;`,
				p.ID, req.Name, req.Value)
			if err != nil {
				log.Printf("Warning: failed to save attribute %s for product %s: %v", req.Name, p.Name, err)
			}
		}
	}

	return tx.Commit()
}

// ParseOffersXML processes offers.xml and updates prices and stock balances in the DB
func ParseOffersXML(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open XML file: %w", err)
	}
	defer file.Close()

	var data CMLOffers
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode offers.xml: %w", err)
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, o := range data.OffersPackage.Offers.Offer {
		var price float64
		if len(o.Prices.Price) > 0 {
			pStr := strings.TrimSpace(o.Prices.Price[0].PricePerUnit)
			pStr = strings.ReplaceAll(pStr, ",", ".")
			if parsedPrice, err := strconv.ParseFloat(pStr, 64); err == nil {
				price = parsedPrice
			} else {
				log.Printf("Warning: failed to parse price '%s' for offer ID %s: %v", pStr, o.ID, err)
			}
		}

		_, err = tx.Exec(`
			UPDATE products 
			SET price = $1, quantity = $2 
			WHERE id = $3;`,
			price, o.Quantity, o.ID)
		if err != nil {
			return fmt.Errorf("failed to update offers for product %s: %w", o.ID, err)
		}
	}

	return tx.Commit()
}

// OrdersToCommerceMLXML converts new orders into CommerceML XML for 1C querying
func OrdersToCommerceMLXML() (string, error) {
	rows, err := database.DB.Query(`
		SELECT id, order_number, customer_name, customer_email, customer_phone, 
		       delivery_method, payment_method, comment, total_price, created_at 
		FROM orders 
		WHERE c1_exported = FALSE;`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		err := rows.Scan(&o.ID, &o.OrderNumber, &o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
			&o.DeliveryMethod, &o.PaymentMethod, &o.Comment, &o.TotalPrice, &o.CreatedAt)
		if err != nil {
			return "", err
		}

		// Fetch items
		itemRows, err := database.DB.Query(`
			SELECT id, product_id, name, quantity, price 
			FROM order_items 
			WHERE order_id = $1;`, o.ID)
		if err != nil {
			return "", err
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var item models.OrderItem
			if err := itemRows.Scan(&item.ID, &item.ProductID, &item.Name, &item.Quantity, &item.Price); err == nil {
				o.Items = append(o.Items, item)
			}
		}
		orders = append(orders, o)
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	sb.WriteString(`<КоммерческаяИнформация ВерсияСхемы="2.05" ДатаФормирования="` + time.Now().Format("2006-01-02") + `">` + "\n")
	sb.WriteString("  <Документ>\n")

	for _, o := range orders {
		sb.WriteString("    <Заказ>\n")
		sb.WriteString("      <Ид>" + o.ID + "</Ид>\n")
		sb.WriteString("      <Номер>" + o.OrderNumber + "</Номер>\n")
		sb.WriteString("      <Дата>" + o.CreatedAt.Format("2006-01-02") + "</Дата>\n")
		sb.WriteString("      <Время>" + o.CreatedAt.Format("15:04:05") + "</Время>\n")
		sb.WriteString("      <ХозОперация>Заказ товара</ХозОперация>\n")
		sb.WriteString("      <Роль>Продавец</Роль>\n")
		sb.WriteString("      <Валюта>RUB</Валюта>\n")
		sb.WriteString("      <Курс>1</Курс>\n")
		sb.WriteString("      <Сумма>" + fmt.Sprintf("%.2f", o.TotalPrice) + "</Сумма>\n")
		sb.WriteString("      <Контрагенты>\n")
		sb.WriteString("        <Контрагент>\n")
		sb.WriteString("          <Наименование>" + o.CustomerName + "</Наименование>\n")
		sb.WriteString("          <Роль>Покупатель</Роль>\n")
		sb.WriteString("          <ПолноеНаименование>" + o.CustomerName + "</ПолноеНаименование>\n")
		sb.WriteString("          <Контакты>\n")
		sb.WriteString("            <Контакт>\n")
		sb.WriteString("              <Тип>ТелефонРабочий</Тип>\n")
		sb.WriteString("              <Значение>" + o.CustomerPhone + "</Значение>\n")
		sb.WriteString("            </Контакт>\n")
		sb.WriteString("            <Контакт>\n")
		sb.WriteString("              <Тип>Почта</Тип>\n")
		sb.WriteString("              <Значение>" + o.CustomerEmail + "</Значение>\n")
		sb.WriteString("            </Контакт>\n")
		sb.WriteString("          </Контакты>\n")
		sb.WriteString("        </Контрагент>\n")
		sb.WriteString("      </Контрагенты>\n")
		sb.WriteString("      <Товары>\n")

		for _, item := range o.Items {
			sb.WriteString("        <Товар>\n")
			sb.WriteString("          <Ид>" + item.ProductID + "</Ид>\n")
			sb.WriteString("          <Наименование>" + item.Name + "</Наименование>\n")
			sb.WriteString("          <ЦенаЗаЕдиницу>" + fmt.Sprintf("%.2f", item.Price) + "</ЦенаЗаЕдиницу>\n")
			sb.WriteString("          <Количество>" + fmt.Sprintf("%d", item.Quantity) + "</Количество>\n")
			sb.WriteString("          <Сумма>" + fmt.Sprintf("%.2f", float64(item.Quantity)*item.Price) + "</Сумма>\n")
			sb.WriteString("        </Товар>\n")
		}

		sb.WriteString("      </Товары>\n")
		sb.WriteString("      <Комментарий>" + o.Comment + " | Доставка: " + o.DeliveryMethod + " | Оплата: " + o.PaymentMethod + "</Комментарий>\n")
		sb.WriteString("    </Заказ>\n")
	}

	sb.WriteString("  </Документ>\n")
	sb.WriteString("</КоммерческаяИнформация>")

	return sb.String(), nil
}
