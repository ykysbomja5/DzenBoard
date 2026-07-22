package handlers

import (
	"archive/zip"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"nasha_igrushka/database"
	"nasha_igrushka/exchange"
)

// Helper to check basic auth credentials from .env
func checkC1Auth(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	expectedUser := os.Getenv("C1_USER")
	expectedPass := os.Getenv("C1_PASS")
	if expectedUser == "" {
		expectedUser = "exchange" // fallback
	}
	if expectedPass == "" {
		expectedPass = "password" // fallback
	}
	return user == expectedUser && pass == expectedPass
}

// Exchange1cHandler acts as the endpoint /export/exchange1c.php
func Exchange1cHandler(w http.ResponseWriter, r *http.Request) {
	// Standard OpenCart PHP integration expected by 1C uses BASIC AUTH
	if !checkC1Auth(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="1C Exchange"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mode := r.URL.Query().Get("mode")
	exType := r.URL.Query().Get("type")

	log.Printf("1C Sync Request: type=%s, mode=%s", exType, mode)

	switch mode {
	case "checkauth":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("success\nkey_session\n1c_session_token_123456\n"))

	case "init":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("zip=yes\nfile_limit=10485760\n")) // 10MB limit, support ZIP

	case "file":
		filename := r.URL.Query().Get("filename")
		if filename == "" {
			http.Error(w, "failure\nno filename provided", http.StatusBadRequest)
			return
		}

		// Prevent directory traversal
		filename = filepath.Base(filename)

		tempDir := os.Getenv("TEMP_DIR")
		if tempDir == "" {
			tempDir = "./temp"
		}
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			log.Printf("Error creating temp dir: %v", err)
			http.Error(w, "failure\ncould not create temp dir", http.StatusInternalServerError)
			return
		}

		filePath := filepath.Join(tempDir, filename)
		out, err := os.Create(filePath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			http.Error(w, "failure\ncould not create file", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, r.Body)
		if err != nil {
			log.Printf("Error saving file: %v", err)
			http.Error(w, "failure\ncould not save file contents", http.StatusInternalServerError)
			return
		}

		log.Printf("File %s uploaded successfully", filename)

		// If it's a zip file, unzip it
		if strings.HasSuffix(strings.ToLower(filename), ".zip") {
			uploadDir := os.Getenv("UPLOAD_DIR")
			if uploadDir == "" {
				uploadDir = "./uploads"
			}
			if err := unzip(filePath, uploadDir); err != nil {
				log.Printf("Error unzipping file %s: %v", filename, err)
				http.Error(w, "failure\nunzip failed", http.StatusInternalServerError)
				return
			}
			log.Printf("ZIP file %s extracted to %s", filename, uploadDir)
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("success\n"))

	case "import":
		filename := r.URL.Query().Get("filename")
		if filename == "" {
			http.Error(w, "failure\nno filename provided", http.StatusBadRequest)
			return
		}
		filename = filepath.Base(filename)

		// 1C uploads ZIP, which gets extracted to uploadDir.
		// If it was a zip file like import.zip, the XML inside is usually import.xml.
		// If 1C sends direct XML, it is in tempDir.
		var targetPath string
		tempDir := os.Getenv("TEMP_DIR")
		if tempDir == "" {
			tempDir = "./temp"
		}
		uploadDir := os.Getenv("UPLOAD_DIR")
		if uploadDir == "" {
			uploadDir = "./uploads"
		}

		if strings.HasSuffix(strings.ToLower(filename), ".zip") {
			// Find XML file extracted inside uploadDir
			xmlName := strings.Replace(strings.ToLower(filename), ".zip", ".xml", 1)
			targetPath = filepath.Join(uploadDir, xmlName)
		} else {
			targetPath = filepath.Join(tempDir, filename)
		}

		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			// Try checking upload dir as fallback
			targetPath = filepath.Join(uploadDir, filename)
		}

		log.Printf("Parsing XML file: %s", targetPath)

		var err error
		if strings.Contains(strings.ToLower(filename), "import") {
			err = exchange.ParseImportXML(targetPath)
		} else if strings.Contains(strings.ToLower(filename), "offers") {
			err = exchange.ParseOffersXML(targetPath)
		} else {
			log.Printf("Unknown file import: %s", filename)
		}

		if err != nil {
			log.Printf("XML import error for %s: %v", filename, err)
			http.Error(w, "failure\nimport failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("success\n"))

	case "query":
		// Sales query (exporting orders from site to 1C)
		if exType == "sale" {
			xmlData, err := exchange.OrdersToCommerceMLXML()
			if err != nil {
				log.Printf("Error generating orders XML: %v", err)
				http.Error(w, "failure\ncould not generate orders XML", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			w.Write([]byte(xmlData))
			return
		}
		http.Error(w, "failure\nunknown query type", http.StatusBadRequest)

	case "success":
		// Confirms that 1C successfully loaded orders from the site
		if exType == "sale" {
			_, err := database.DB.Exec("UPDATE orders SET c1_exported = TRUE WHERE c1_exported = FALSE;")
			if err != nil {
				log.Printf("Error marking orders as exported: %v", err)
				http.Error(w, "failure\ncould not update order status", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte("success\n"))
			return
		}
		http.Error(w, "failure\nunknown success type", http.StatusBadRequest)

	default:
		http.Error(w, "failure\nunknown mode", http.StatusBadRequest)
	}
}

// Unzip helper to extract ZIP archives
func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Secure zip extraction: prevent zip slip vulnerability
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
