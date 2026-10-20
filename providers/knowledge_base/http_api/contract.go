// Package httpapi defines the typed operation contract for the official
// Knowledge Base HTTP provider. The network adapter implementation lives in
// domainry-connectors and is selected by an outer composition root.
package httpapi

import (
	"encoding/json"
	"path/filepath"
	"strings"

	connector "github.com/domainry/domainry-connector-sdk"
)

const (
	ConnectorKey                       = "knowledge_base"
	ProviderKey                        = "http_api"
	MaximumInlineDocumentBytes         = 10 << 20
	BuilderAssetMaximumBytes           = 20 << 20
	BuilderMediaMaximumBytes           = 20 << 20
	BuilderImageEnrichmentMaximumBytes = 8 << 20
	BuilderAnalysisOutputMaximumBytes  = 16 << 20
)

// DocumentFormatCapability mirrors kb-builder's deployed type-detection
// routes. MaximumInlineDocumentBytes is the current HTTP ingress ceiling and
// is intentionally separate from Builder's route-specific processing budgets.
// A zero BuilderWholeFileMaximumBytes means Builder does not declare a
// whole-file byte ceiling for that route; it is not permission to upload an
// unbounded file. AssetMaximumBytes, EnrichmentMaxBytes and AnalysisMaxBytes
// are derived-work budgets, not source-document upload limits.
type DocumentFormatCapability struct {
	Extension                    string
	ContentType                  string
	ParserRoute                  string
	BuilderWholeFileMaximumBytes int64
	AssetMaximumBytes            int64
	EnrichmentMaxBytes           int64
	AnalysisMaxBytes             int64
}

var documentFormats = []DocumentFormatCapability{
	{Extension: ".pdf", ContentType: "application/pdf", ParserRoute: "pdf", AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".docx", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", ParserRoute: "office", AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".pptx", ContentType: "application/vnd.openxmlformats-officedocument.presentationml.presentation", ParserRoute: "office", AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ParserRoute: "structured", AssetMaximumBytes: BuilderAssetMaximumBytes, AnalysisMaxBytes: BuilderAnalysisOutputMaximumBytes},
	{Extension: ".xlsm", ContentType: "application/vnd.ms-excel.sheet.macroEnabled.12", ParserRoute: "structured", AssetMaximumBytes: BuilderAssetMaximumBytes, AnalysisMaxBytes: BuilderAnalysisOutputMaximumBytes},
	{Extension: ".txt", ContentType: "text/plain", ParserRoute: "text"},
	{Extension: ".md", ContentType: "text/markdown", ParserRoute: "text"},
	{Extension: ".rst", ContentType: "text/plain", ParserRoute: "text"},
	{Extension: ".html", ContentType: "text/html", ParserRoute: "html"},
	{Extension: ".htm", ContentType: "text/html", ParserRoute: "html"},
	{Extension: ".csv", ContentType: "text/csv", ParserRoute: "structured", AnalysisMaxBytes: BuilderAnalysisOutputMaximumBytes},
	{Extension: ".json", ContentType: "application/json", ParserRoute: "structured", AnalysisMaxBytes: BuilderAnalysisOutputMaximumBytes},
	{Extension: ".jsonl", ContentType: "application/x-ndjson", ParserRoute: "structured", AnalysisMaxBytes: BuilderAnalysisOutputMaximumBytes},
	{Extension: ".py", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".js", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".ts", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".tsx", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".jsx", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".java", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".go", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".rs", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".c", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".h", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".cpp", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".hpp", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".cs", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".rb", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".php", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".kt", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".swift", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".scala", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".sh", ContentType: "text/plain", ParserRoute: "code"},
	{Extension: ".png", ContentType: "image/png", ParserRoute: "media", AssetMaximumBytes: BuilderAssetMaximumBytes, EnrichmentMaxBytes: BuilderImageEnrichmentMaximumBytes},
	{Extension: ".jpg", ContentType: "image/jpeg", ParserRoute: "media", AssetMaximumBytes: BuilderAssetMaximumBytes, EnrichmentMaxBytes: BuilderImageEnrichmentMaximumBytes},
	{Extension: ".jpeg", ContentType: "image/jpeg", ParserRoute: "media", AssetMaximumBytes: BuilderAssetMaximumBytes, EnrichmentMaxBytes: BuilderImageEnrichmentMaximumBytes},
	{Extension: ".gif", ContentType: "image/gif", ParserRoute: "media", AssetMaximumBytes: BuilderAssetMaximumBytes, EnrichmentMaxBytes: BuilderImageEnrichmentMaximumBytes},
	{Extension: ".bmp", ContentType: "image/bmp", ParserRoute: "media", AssetMaximumBytes: BuilderAssetMaximumBytes, EnrichmentMaxBytes: BuilderImageEnrichmentMaximumBytes},
	{Extension: ".webp", ContentType: "image/webp", ParserRoute: "media", AssetMaximumBytes: BuilderAssetMaximumBytes, EnrichmentMaxBytes: BuilderImageEnrichmentMaximumBytes},
	{Extension: ".mp3", ContentType: "audio/mpeg", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".wav", ContentType: "audio/wav", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".ogg", ContentType: "audio/ogg", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".flac", ContentType: "audio/flac", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".m4a", ContentType: "audio/mp4", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".aac", ContentType: "audio/aac", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".mp4", ContentType: "video/mp4", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".mov", ContentType: "video/quicktime", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".webm", ContentType: "video/webm", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
	{Extension: ".avi", ContentType: "video/x-msvideo", ParserRoute: "media", BuilderWholeFileMaximumBytes: BuilderMediaMaximumBytes, AssetMaximumBytes: BuilderAssetMaximumBytes},
}

func DocumentFormatCapabilities() []DocumentFormatCapability {
	return append([]DocumentFormatCapability(nil), documentFormats...)
}

func DocumentFormat(filename string) (DocumentFormatCapability, bool) {
	extension := strings.ToLower(filepath.Ext(filename))
	for _, capability := range documentFormats {
		if capability.Extension == extension {
			return capability, true
		}
	}
	return DocumentFormatCapability{}, false
}

// MaximumInlineDocumentBytesForFilename returns the maximum body the current
// kb-search-api document endpoint can actually receive. It includes Builder's
// whole-file ceiling when that ceiling is stricter than the HTTP ingress.
func MaximumInlineDocumentBytesForFilename(filename string) int64 {
	capability, ok := DocumentFormat(filename)
	if !ok {
		return 0
	}
	maximum := int64(MaximumInlineDocumentBytes)
	if capability.BuilderWholeFileMaximumBytes > 0 && capability.BuilderWholeFileMaximumBytes < maximum {
		maximum = capability.BuilderWholeFileMaximumBytes
	}
	return maximum
}

// BuilderWholeFileMaximumBytesForFilename returns only Builder's declared
// source-file ceiling. Zero means the selected Builder route has no explicit
// whole-file ceiling; it does not include the upstream HTTP ingress ceiling.
func BuilderWholeFileMaximumBytesForFilename(filename string) int64 {
	capability, ok := DocumentFormat(filename)
	if !ok {
		return 0
	}
	return capability.BuilderWholeFileMaximumBytes
}

type SearchInput struct {
	Query         string   `json:"query"`
	TopK          int      `json:"top_k,omitempty"`
	RecordKinds   []string `json:"record_kinds,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	Rerank        *bool    `json:"rerank,omitempty"`
	ResultContent string   `json:"result_content,omitempty"`
}

type FetchInput struct {
	DocID         string `json:"doc_id"`
	ResultContent string `json:"result_content,omitempty"`
}

type Output struct {
	Provider string          `json:"provider"`
	KBID     string          `json:"kb_id"`
	Result   json.RawMessage `json:"result"`
}

type DocumentInput struct {
	DocID string `json:"doc_id"`
}

type PutDocumentInput struct {
	DocID    string `json:"doc_id"`
	Filename string `json:"filename"`
	Bytes    int64  `json:"bytes"`
	SHA256   string `json:"sha256"`
}

type DocumentStatusOutput struct {
	Provider    string `json:"provider"`
	KBID        string `json:"kb_id"`
	DocID       string `json:"doc_id"`
	Exists      bool   `json:"exists"`
	IndexStatus string `json:"index_status,omitempty"`
}

type AnalysisTableCatalogInput struct{}

type AnalysisTableColumn struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Unit      string `json:"unit"`
	Precision int    `json:"precision,omitempty"`
	Scale     int    `json:"scale,omitempty"`
}

type AnalysisTableCatalogItem struct {
	DatasetKey        string                `json:"dataset_key"`
	DocID             string                `json:"doc_id"`
	TableRef          string                `json:"table_ref"`
	Name              string                `json:"name"`
	Sheet             string                `json:"sheet"`
	DefinitionVersion string                `json:"definition_version"`
	DataVersion       string                `json:"data_version"`
	Generation        string                `json:"generation"`
	DocVersion        int                   `json:"doc_version"`
	RowCount          int64                 `json:"row_count"`
	Complete          bool                  `json:"complete"`
	Columns           []AnalysisTableColumn `json:"columns"`
}

type AnalysisTableCatalogOutput struct {
	Provider string                     `json:"provider"`
	KBID     string                     `json:"kb_id"`
	Tables   []AnalysisTableCatalogItem `json:"tables"`
}

type AnalysisTableReadInput struct {
	DocID             string   `json:"doc_id"`
	DatasetKey        string   `json:"dataset_key"`
	Generation        string   `json:"generation"`
	DefinitionVersion string   `json:"definition_version"`
	DataVersion       string   `json:"data_version"`
	Fields            []string `json:"fields"`
	Offset            int      `json:"offset"`
	Limit             int      `json:"limit"`
}

type AnalysisTableReadOutput struct {
	Provider string `json:"provider"`
	KBID     string `json:"kb_id"`
	AnalysisTableCatalogItem
	Fields        []string    `json:"fields"`
	ContentSHA256 string      `json:"content_sha256"`
	Offset        int         `json:"offset"`
	Rows          [][]*string `json:"rows"`
	NextOffset    *int        `json:"next_offset"`
}

var Search = connector.CallOperation[SearchInput, Output]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "search", ContractSHA256: "634b756d886ac5bb29d4bcd38c767f9765a41d75f5bfa1a14f1e139d626c09d7", Reliability: readReliability()}
var Fetch = connector.CallOperation[FetchInput, Output]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "fetch", ContractSHA256: "31256a5720fc482b2fd0cd0b259936a58f8bc903c712a66f567c934f42a6b80e", Reliability: readReliability()}
var PutDocument = connector.CallOperation[PutDocumentInput, Output]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "put_document", ContractSHA256: "f6f49d46f100ed2a9685de45e3046a47dc91e1760eb1f779858126abe9c6d772", Reliability: documentWriteReliability()}
var DeleteDocument = connector.CallOperation[DocumentInput, Output]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "delete_document", ContractSHA256: "e4bec5d6d992035b91bf019b55a609307dc56f401c89ef78709f74f57e8e8aa8", Reliability: documentDeleteReliability()}
var DocumentStatus = connector.CallOperation[DocumentInput, DocumentStatusOutput]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "document_status", ContractSHA256: "7b5608d5de92711c8069d7f0df8e41c789175584f5cd94dbe6e1e714331eb9cb", Reliability: readReliability()}
var CatalogAnalysisTables = connector.CallOperation[AnalysisTableCatalogInput, AnalysisTableCatalogOutput]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "analysis_table_catalog", ContractSHA256: "a4cc8ab7e1c7f78a2c17995dece68922d90dfb19e6b1dc5017199074be29b266", Reliability: readReliability()}
var ReadAnalysisTable = connector.CallOperation[AnalysisTableReadInput, AnalysisTableReadOutput]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "analysis_table_read", ContractSHA256: "292b1dbe23ad36bdbaad40fb4e25af395f7f7ba4aee9630503849d428e09efa0", Reliability: readReliability()}

func readReliability() connector.ReliabilityContract {
	return connector.ReliabilityContract{Effect: connector.EffectRead, Idempotency: connector.IdempotencyContract{Strategy: connector.IdempotencyNatural}, Reconciliation: connector.ReconciliationNone, Compensation: connector.CompensationContract{Mode: connector.CompensationNone}}
}

func documentWriteReliability() connector.ReliabilityContract {
	return connector.ReliabilityContract{Effect: connector.EffectWrite, Idempotency: connector.IdempotencyContract{Strategy: connector.IdempotencyNone}, Reconciliation: connector.ReconciliationNone, Compensation: connector.CompensationContract{Mode: connector.CompensationNone}}
}

func documentDeleteReliability() connector.ReliabilityContract {
	result := documentWriteReliability()
	result.Idempotency.Strategy = connector.IdempotencyNatural
	return result
}
