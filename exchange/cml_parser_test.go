package exchange

import (
	"encoding/xml"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Конструктор LEGO", "konstruktor-lego"},
		{"Мягкая игрушка \"Медвежонок\"", "myagkaya-igrushka-medvezhonok"},
		{"Самокат 3-х колесный", "samokat-3-h-kolesnyy"},
		{"  Детский транспорт! ", "detskiy-transport"},
		{"Hello World 123", "hello-world-123"},
	}

	for _, tt := range tests {
		result := Slugify(tt.input)
		if result != tt.expected {
			t.Errorf("Slugify(%q) = %q; expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestCommerceMLXMLParsing(t *testing.T) {
	testImportXML := `<?xml version="1.0" encoding="utf-8"?>
	<КоммерческаяИнформация ВерсияСхемы="2.05" ДатаФормирования="2026-07-13">
	  <Классификатор>
		<Группы>
		  <Группа>
			<Ид>cat-test</Ид>
			<Наименование>Тестовая категория</Наименование>
		  </Группа>
		</Группы>
	  </Классификатор>
	  <Каталог>
		<Товары>
		  <Товар>
			<Ид>prod-test</Ид>
			<Артикул>TEST-001</Артикул>
			<Наименование>Тестовый товар</Наименование>
			<Описание>Описание тестового товара</Описание>
			<Группы>
			  <Ид>cat-test</Ид>
			</Группы>
		  </Товар>
		</Товары>
	  </Каталог>
	</КоммерческаяИнформация>`

	var data CMLImport
	err := xml.Unmarshal([]byte(testImportXML), &data)
	if err != nil {
		t.Fatalf("Failed to parse import XML: %v", err)
	}

	if len(data.Classifier.Groups.Group) != 1 || data.Classifier.Groups.Group[0].ID != "cat-test" {
		t.Errorf("Category not parsed correctly: %+v", data.Classifier.Groups.Group)
	}

	if len(data.Catalog.Products.Product) != 1 || data.Catalog.Products.Product[0].ID != "prod-test" {
		t.Errorf("Product not parsed correctly: %+v", data.Catalog.Products.Product)
	}
}
