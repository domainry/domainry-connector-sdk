// Package httpapi defines the typed operation contract for the official
// Knowledge Base HTTP provider. The network adapter implementation lives in
// domainry-connectors and is selected by an outer composition root.
package httpapi

import (
	"encoding/json"

	connector "github.com/domainry/domainry-connector-sdk"
)

const (
	ConnectorKey     = "knowledge_base"
	ProviderKey      = "http_api"
	MaxDocumentBytes = 10 << 20
)

type SearchInput struct {
	Query         string `json:"query"`
	TopK          int    `json:"top_k,omitempty"`
	ResultContent string `json:"result_content,omitempty"`
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
	Content  []byte `json:"content"`
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

var Search = connector.CallOperation[SearchInput, Output]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "search", ContractSHA256: "573e693a2b63cd71a5f0937f5d1be49124e36d312de707160a5524ec00ab6c6a", Reliability: readReliability()}
var Fetch = connector.CallOperation[FetchInput, Output]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "fetch", ContractSHA256: "31256a5720fc482b2fd0cd0b259936a58f8bc903c712a66f567c934f42a6b80e", Reliability: readReliability()}
var PutDocument = connector.CallOperation[PutDocumentInput, Output]{ConnectorKey: ConnectorKey, ProviderKey: ProviderKey, Key: "put_document", ContractSHA256: "ada514ca5026f94fbe6191208dcd185567e2124d0fa04caba9d0ef8795d0f3c1", Reliability: documentWriteReliability()}
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
