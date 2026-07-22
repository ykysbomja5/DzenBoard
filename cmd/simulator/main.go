package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

const (
	baseURL  = "http://localhost:8085/export/exchange1c.php"
	username = "exchange"
	password = "password"
)

// Sample CommerceML import.xml contents
const importXML = `<?xml version="1.0" encoding="utf-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.05" ДатаФормирования="2026-07-13">
  <Классификатор>
    <Ид>c1-classifier</Ид>
    <Наименование>Классификатор товаров</Наименование>
    <Группы>
      <Группа>
        <Ид>c1-cat-interactive</Ид>
        <Наименование>Интерактивные игрушки</Наименование>
        <Группы>
          <Группа>
            <Ид>c1-cat-robots</Ид>
            <Наименование>Роботы на радиоуправлении</Наименование>
          </Группа>
        </Группы>
      </Группа>
      <Группа>
        <Ид>c1-cat-boardgames</Ид>
        <Наименование>Настольные игры</Наименование>
      </Группа>
    </Группы>
  </Классификатор>
  <Каталог>
    <Ид>c1-catalog</Ид>
    <Наименование>Основной каталог игрушек</Наименование>
    <Товары>
      <Товар>
        <Ид>c1-prod-monopoly</Ид>
        <Артикул>BG-002</Артикул>
        <Наименование>Настольная игра Монополия Классическая</Наименование>
        <Описание>Классическая настольная игра для всей семьи. Учит основам экономики, торговли и планирования. В комплекте металлическое игровое поле, фишки, карточки и игровые банкноты.</Описание>
        <Группы>
          <Ид>c1-cat-boardgames</Ид>
        </Группы>
        <ЗначенияРеквизитов>
          <ЗначениеРеквизита>
            <Наименование>Бренд</Наименование>
            <Значение>Hasbro</Значение>
          </ЗначениеРеквизита>
          <ЗначениеРеквизита>
            <Наименование>Возраст</Наименование>
            <Значение>8+</Значение>
          </ЗначениеРеквизита>
          <ЗначениеРеквизита>
            <Наименование>Материал</Наименование>
            <Значение>Картон / Пластик / Металл</Значение>
          </ЗначениеРеквизита>
        </ЗначенияРеквизитов>
      </Товар>
      <Товар>
        <Ид>c1-prod-robot-dog</Ид>
        <Артикул>RC-005</Артикул>
        <Наименование>Робот-собака на радиоуправлении SmartDog</Наименование>
        <Описание>Умная интерактивная собака-робот, которая выполняет команды, лает, танцует под музыку и реагирует на прикосновения. Пульт управления работает на частоте 2.4 ГГц.</Описание>
        <Группы>
          <Ид>c1-cat-robots</Ид>
        </Группы>
        <ЗначенияРеквизитов>
          <ЗначениеРеквизита>
            <Наименование>Бренд</Наименование>
            <Значение>SmartToys</Значение>
          </ЗначениеРеквизита>
          <ЗначениеРеквизита>
            <Наименование>Возраст</Наименование>
            <Значение>5+</Значение>
          </ЗначениеРеквизита>
          <ЗначениеРеквизита>
            <Наименование>Питание</Наименование>
            <Значение>Аккумулятор Li-Ion (в комплекте)</Значение>
          </ЗначениеРеквизита>
        </ЗначенияРеквизитов>
      </Товар>
    </Товары>
  </Каталог>
</КоммерческаяИнформация>
`

// Sample CommerceML offers.xml contents (price and stock levels)
const offersXML = `<?xml version="1.0" encoding="utf-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.05" ДатаФормирования="2026-07-13">
  <ПакетПредложений>
    <Ид>c1-offers-pkg</Ид>
    <Наименование>Пакет предложений (Цены и остатки)</Наименование>
    <Предложения>
      <Предложение>
        <Ид>c1-prod-monopoly</Ид>
        <Наименование>Настольная игра Монополия Классическая</Наименование>
        <Количество>14</Количество>
        <Цены>
          <Цена>
            <ЦенаЗаЕдиницу>1950.00</ЦенаЗаЕдиницу>
            <Валюта>RUB</Валюта>
          </Цена>
        </Цены>
      </Предложение>
      <Предложение>
        <Ид>c1-prod-robot-dog</Ид>
        <Наименование>Робот-собака на радиоуправлении SmartDog</Наименование>
        <Количество>6</Количество>
        <Цены>
          <Цена>
            <ЦенаЗаЕдиницу>3200.00</ЦенаЗаЕдиницу>
            <Валюта>RUB</Валюта>
          </Цена>
        </Цены>
      </Предложение>
    </Предложения>
  </ПакетПредложений>
</КоммерческаяИнформация>
`

func main() {
	log.Println("=== Запуск симулятора обмена 1С ===")

	// 1. Check Auth
	sessionToken := stepCheckAuth()
	if sessionToken == "" {
		log.Fatal("Ошибка: не удалось авторизоваться в шлюзе сайта.")
	}

	// 2. Initialize
	stepInit()

	// 3. Upload import.xml
	stepUploadFile("import.xml", importXML)

	// 4. Upload offers.xml
	stepUploadFile("offers.xml", offersXML)

	// 5. Trigger import processing for import.xml
	stepProcessImport("import.xml")

	// 6. Trigger import processing for offers.xml
	stepProcessImport("offers.xml")

	log.Println("=== Симуляция успешно завершена! ===")
}

func stepCheckAuth() string {
	log.Println("[1C Sim] 1. Запрос checkauth...")
	req, err := http.NewRequest("GET", baseURL+"?type=catalog&mode=checkauth", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[1C Sim] Ответ сервера:\n%s", string(body))

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Некорректный статус ответа: %d", resp.StatusCode)
	}
	return "ok"
}

func stepInit() {
	log.Println("[1C Sim] 2. Запрос init...")
	req, err := http.NewRequest("GET", baseURL+"?type=catalog&mode=init", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[1C Sim] Ответ сервера:\n%s", string(body))
}

func stepUploadFile(filename string, content string) {
	log.Printf("[1C Sim] Загрузка файла %s...", filename)
	
	buf := bytes.NewBufferString(content)
	req, err := http.NewRequest("POST", baseURL+"?type=catalog&mode=file&filename="+filename, buf)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[1C Sim] Ответ сервера:\n%s", string(body))
}

func stepProcessImport(filename string) {
	log.Printf("[1C Sim] Запуск импорта файла %s...", filename)
	req, err := http.NewRequest("GET", baseURL+"?type=catalog&mode=import&filename="+filename, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[1C Sim] Ответ сервера:\n%s", string(body))
}
