package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"collab-editor/internal/db"

	"github.com/jung-kurt/gofpdf"
)

type ExportHandler struct {
	db *db.Database
}

func NewExportHandler(database *db.Database) *ExportHandler {
	return &ExportHandler{
		db: database,
	}
}

func (h *ExportHandler) ExportDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session code and format from query params
	sessionCode := r.URL.Query().Get("session")
	format := r.URL.Query().Get("format")

	if sessionCode == "" {
		http.Error(w, "Session code required", http.StatusBadRequest)
		return
	}

	// Get session content from database
	session, err := h.db.GetOrCreateSession(sessionCode)
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Generate filename
	filename := fmt.Sprintf("document-%s-%s", sessionCode, time.Now().Format("20060102-150405"))

	switch format {
	case "txt":
		h.exportTXT(w, session.Content, filename)
	case "pdf":
		h.exportPDF(w, session.Content, filename)
	case "docx":
		h.exportDOCX(w, session.Content, filename)
	default:
		http.Error(w, "Invalid format. Supported formats: txt, pdf, docx", http.StatusBadRequest)
	}
}

func (h *ExportHandler) exportTXT(w http.ResponseWriter, content, filename string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.txt\"", filename))
	w.Write([]byte(content))
}

func (h *ExportHandler) exportPDF(w http.ResponseWriter, content, filename string) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set font
	pdf.SetFont("Arial", "", 12)

	// Add title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Collaborative Document")
	pdf.Ln(15)

	// Reset font for content
	pdf.SetFont("Arial", "", 12)

	// Split content into lines and add to PDF
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if line == "" {
			pdf.Ln(5)
		} else {
			// Handle long lines by wrapping
			pdf.MultiCell(0, 5, line, "", "L", false)
		}
	}

	// Output PDF
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.pdf\"", filename))
	w.Write(buf.Bytes())
}

func (h *ExportHandler) exportDOCX(w http.ResponseWriter, content, filename string) {
	// Create a simple DOCX structure
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add required DOCX structure files
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
    <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
    <Default Extension="xml" ContentType="application/xml"/>
    <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
    <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`,
	}

	// Create document content
	documentContent := h.createDocumentXML(content)
	files["word/document.xml"] = documentContent

	// Write all files to zip
	for name, content := range files {
		writer, err := zipWriter.Create(name)
		if err != nil {
			http.Error(w, "Failed to create DOCX", http.StatusInternalServerError)
			return
		}
		_, err = writer.Write([]byte(content))
		if err != nil {
			http.Error(w, "Failed to write DOCX", http.StatusInternalServerError)
			return
		}
	}

	// Close zip writer
	err := zipWriter.Close()
	if err != nil {
		http.Error(w, "Failed to finalize DOCX", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.docx\"", filename))
	w.Write(buf.Bytes())
}

func (h *ExportHandler) createDocumentXML(content string) string {
	// Escape XML special characters
	content = strings.ReplaceAll(content, "&", "&amp;")
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	content = strings.ReplaceAll(content, "\"", "&quot;")
	content = strings.ReplaceAll(content, "'", "&apos;")

	// Split content into paragraphs
	paragraphs := strings.Split(content, "\n")

	// Build paragraph XML
	var paragraphsXML strings.Builder
	for _, para := range paragraphs {
		if para == "" {
			paragraphsXML.WriteString(`<w:p><w:r><w:t></w:t></w:r></w:p>`)
		} else {
			paragraphsXML.WriteString(fmt.Sprintf(`<w:p><w:r><w:t>%s</w:t></w:r></w:p>`, para))
		}
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
    <w:body>
        <w:p>
            <w:pPr>
                <w:pStyle w:val="Title"/>
            </w:pPr>
            <w:r>
                <w:t>Collaborative Document</w:t>
            </w:r>
        </w:p>
        %s
    </w:body>
</w:document>`, paragraphsXML.String())
}