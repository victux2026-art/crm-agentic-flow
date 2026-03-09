package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	geminiModel = "gemini-2.5-flash" // Usamos el modelo probado
)

// Simplified CRMSchema representa una parte de nuestro Elastic Schema para mapeo
type CRMSchema struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone_number"`
	Company   string `json:"company_name"`
	JobTitle  string `json:"job_title"`
	// Podemos añadir más campos aquí para simular el esquema completo
	Metadata map[string]string `json:"metadata"` // Para campos personalizados que Gemini pueda sugerir
}

// MigrationMapping representa la estructura de mapeo que esperamos de Gemini
type MigrationMapping struct {
	SourceToCRMMapping      map[string]string `json:"source_to_crm_mapping"`       // Ej: {"Old Name": "first_name", "Email Address": "email"}
	SuggestedMetadataFields map[string]string `json:"suggested_metadata_fields"` // Ej: {"Lead Score": "lead_score", "Industry": "industry"}
	Notes                   string            `json:"notes"`                       // Notas adicionales de Gemini sobre el mapeo
}

// La variable global geminiClient ya NO es necesaria aquí.
// var geminiClient *genai.Client
var geminiAPIKey string // Esta sí se sigue cargando en main y se pasa a la función

func main() {
	fmt.Println("DEBUG: Iniciando migration_agent para prueba de mapeo con Gemini.")
	ctx := context.Background() // Contexto para el main

	// Cargar la clave de API de Gemini desde una variable de entorno
	geminiAPIKey = os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY no está configurada en las variables de entorno para el Migration Agent.")
	}
	// Ya no inicializamos geminiClient aquí, lo haremos en callGeminiForMapping
	// fmt.Println("Migration Agent Connected to Gemini!") // <-- ELIMINADO

	// 1. Crear un archivo CSV de ejemplo
	csvFileName := "sample_crm_data.csv"
	createSampleCSV(csvFileName)
	fmt.Printf("DEBUG: Archivo CSV de ejemplo creado: %s\n", csvFileName)

	// 2. Leer el CSV y obtener las cabeceras y las primeras filas
	headers, sampleRows, err := readCSVForMapping(csvFileName, 5) // Leer 5 filas de ejemplo
	if err != nil {
		log.Fatalf("Error al leer el archivo CSV: %v\n", err)
	}
	fmt.Println("DEBUG: Cabeceras del CSV:", headers)
	fmt.Println("DEBUG: Filas de ejemplo del CSV:", sampleRows)

	// 3. Construir el prompt para Gemini
	prompt := buildMappingPrompt(headers, sampleRows)
	fmt.Printf("DEBUG: Prompt enviado a Gemini para mapeo: %s\n", prompt)

	// 4. Llamar a Gemini para obtener el mapeo
	// callGeminiForMapping ahora manejará su propia inicialización del cliente
	geminiResponseText, err := callGeminiForMapping(ctx, prompt) // Pasamos el contexto del main
	if err != nil {
		log.Fatalf("Error al obtener el mapeo de Gemini: %v\n", err)
	}
	fmt.Printf("DEBUG: Respuesta RAW de Gemini para mapeo: %s\n", geminiResponseText)

	// 5. Limpiar y parsear la respuesta de Gemini
	cleanedResponseText := cleanGeminiResponse(geminiResponseText)
	fmt.Printf("DEBUG: Respuesta limpia de Gemini para mapeo: %s\n", cleanedResponseText)

	var migrationMapping MigrationMapping
	if err := json.Unmarshal([]byte(cleanedResponseText), &migrationMapping); err != nil {
		log.Fatalf("Error al parsear el JSON de mapeo de Gemini: %v\nRespuesta limpia: %s\n", err, cleanedResponseText)
	}

	fmt.Println("\n--- Mapeo de Migración Sugerido por Gemini ---")
	fmt.Printf("Mapeo Fuente a CRM: %v\n", migrationMapping.SourceToCRMMapping)
	fmt.Printf("Campos de Metadatos Sugeridos: %v\n", migrationMapping.SuggestedMetadataFields)
	fmt.Printf("Notas de Gemini: %s\n", migrationMapping.Notes)
	fmt.Println("----------------------------------------------")

	// Simular aplicación del mapeo
	fmt.Println("Simulando aplicación del mapeo para los datos del CSV...")
	// En un escenario real, aquí se procesarían las filas del CSV usando el migrationMapping
	fmt.Println("DEBUG: Ejecución de migration_agent finalizada.")
}

// createSampleCSV crea un archivo CSV de ejemplo
func createSampleCSV(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("No se pudo crear el archivo CSV de ejemplo: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"First Name", "Last Name", "Email Address", "Phone #", "Company Name", "Job Title", "Lead Score", "Industry"}
	data := [][]string{
		{"Alice", "Smith", "alice@example.com", "111-222-3333", "Acme Corp", "Sales Manager", "85", "Software"},
		{"Bob", "Johnson", "bob@example.com", "444-555-6666", "Globex Inc", "Marketing Lead", "70", "Marketing"},
		{"Charlie", "Brown", "charlie@example.com", "777-888-9999", "Pied Piper", "CTO", "92", "Software"},
	}

	writer.Write(headers)
	writer.WriteAll(data)
}

// readCSVForMapping lee las cabeceras y algunas filas de ejemplo de un CSV
func readCSVForMapping(filename string, numSampleRows int) ([]string, [][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo abrir el archivo CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Leer cabeceras
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("error al leer cabeceras del CSV: %w", err)
	}

	// Leer filas de ejemplo
	var sampleRows [][]string
	for i := 0; i < numSampleRows; i++ {
		record, err := reader.Read()
		if err == nil {
			sampleRows = append(sampleRows, record)
		} else if err == io.EOF { // <-- CORREGIDO a io.EOF
			break
		} else {
			return nil, nil, fmt.Errorf("error al leer fila de ejemplo del CSV: %w", err)
		}
	}
	return headers, sampleRows, nil
}

// buildMappingPrompt construye el prompt para Gemini
func buildMappingPrompt(headers []string, sampleRows [][]string) string {
	schemaJSON, _ := json.MarshalIndent(CRMSchema{}, "", "  ") // Mostrar nuestro esquema
	
	csvSample := fmt.Sprintf("Cabeceras del CSV: %s\nPrimeras filas de datos:\n", strings.Join(headers, ", "))
	for _, row := range sampleRows {
		csvSample += strings.Join(row, ", ") + "\n"
	}

	prompt := fmt.Sprintf(`Eres un asistente experto en migración de datos CRM. Te proporcionaré las cabeceras y algunas filas de ejemplo de un archivo CSV de un CRM antiguo, y el esquema de nuestro nuevo CRM. Tu tarea es sugerir el mejor mapeo de los campos del CSV a nuestro esquema.

Nuestro esquema CRM (formato JSON simplificado) es:
%s

Datos de ejemplo del CSV a mapear:
%s

Por favor, proporciona el mapeo sugerido en un objeto JSON válido con las siguientes claves:
- "source_to_crm_mapping": Un mapa donde la clave es el nombre de la cabecera del CSV y el valor es el nombre del campo en nuestro esquema CRM (ej. "First Name": "first_name", "Email Address": "email"). Si un campo del CSV no encaja directamente, o si crees que debería ir en un campo de metadatos, no lo incluyas aquí.
- "suggested_metadata_fields": Un mapa donde la clave es el nombre de la cabecera del CSV y el valor es un nombre de clave sugerido para el campo 'metadata' en nuestro CRM (ej. "Lead Score": "lead_score", "Industry": "industry"). Esto es para campos personalizados que no tienen una correspondencia directa en nuestro esquema principal.
- "notes": Cualquier nota importante sobre el mapeo o campos que no pudiste mapear.

Asegúrate de que tu respuesta sea ÚNICAMENTE el objeto JSON. No incluyas texto conversacional antes o después.`, string(schemaJSON), csvSample)

	return prompt
}

// callGeminiForMapping ahora inicializa y cierra su propio cliente Gemini
func callGeminiForMapping(ctx context.Context, prompt string) (string, error) {
    // Inicializar cliente Gemini dentro de la función
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiAPIKey))
	if err != nil {
		return "", fmt.Errorf("error al crear el cliente Gemini para el mapeo: %w", err)
	}
	defer client.Close() // Aseguramos que se cierre al salir de la función

	fmt.Println("DEBUG: callGeminiForMapping: Cliente Gemini creado para esta llamada.")

	model := client.GenerativeModel(geminiModel)
	if model == nil { // Comprobación defensiva, aunque no debería ser nil por la firma
		return "", fmt.Errorf("callGeminiForMapping: GenerativeModel returned nil")
	}

	// Generamos un contexto con timeout para esta llamada a Gemini
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // Timeout de 30 segundos
	defer cancel()

	resp, err := model.GenerateContent(callCtx, genai.Text(prompt)) // Usamos el callCtx
	if err != nil {
		return "", fmt.Errorf("error al llamar a Gemini para el mapeo: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no se recibió contenido de Gemini para el mapeo")
	}

	geminiResponseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			geminiResponseText += string(text)
		}
	}
	return geminiResponseText, nil
}

// cleanGeminiResponse limpia la respuesta de Gemini de bloques de código Markdown y texto extra
func cleanGeminiResponse(geminiResponseText string) string {
	cleanedResponseText := geminiResponseText
	jsonStart := strings.Index(cleanedResponseText, "```json\n")
	jsonEnd := strings.LastIndex(cleanedResponseText, "\n```")

	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		cleanedResponseText = cleanedResponseText[jsonStart+len("```json\n"):jsonEnd]
	} else if jsonStart == -1 && jsonEnd == -1 {
        // Si no hay marcadores, intentar limpiar solo el texto conversacional si empieza con "El correo electrónico..."
        // Esta heurística es más específica para correos y podría no ser ideal para mapeo
        if strings.HasPrefix(cleanedResponseText, "El correo electrónico") || strings.HasPrefix(cleanedResponseText, "El cuerpo del correo") {
            parts := strings.SplitN(cleanedResponseText, "\n\n", 2)
            if len(parts) > 1 {
                cleanedResponseText = parts[1]
            }
        }
    }
    return strings.TrimSpace(cleanedResponseText)
}
