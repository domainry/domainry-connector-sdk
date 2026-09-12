package web

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
)

const ContractVersion = "public-web-read-v1"
const ContractSHA256 = "4a23e7257e9896ea361a13e981fd4fcfd9a41b3989d91bef7cbd8826f7f26024"
const (
	SearchOperationKey = "web_search"
	FetchOperationKey  = "web_fetch"
)

func ComputedContractSHA256() string {
	var b strings.Builder
	b.WriteString(ContractVersion + ";read-only;query-4096-bytes;ranked-results-not-complete;limit-default-5-max-10;excerpt-default-1500-max-8000-bytes;content-default-16384-max-65536-bytes;public-url-syntax-not-network-authorization;url-4096-standard-ports-no-credentials-discard-fragments;source-completeness-unknown-or-partial;title-1024-description-4096;warnings-8x512;source-account-owned;")
	for _, v := range []any{SearchRequest{}, SearchItem{}, SearchResult{}, FetchRequest{}, Page{}} {
		t := reflect.TypeOf(v)
		b.WriteString(t.Name() + "{")
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			b.WriteString(f.Name + ":" + f.Type.String() + ":" + f.Tag.Get("json") + ";")
		}
		b.WriteString("}")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
func OperationSHA256(key string) string {
	if key != SearchOperationKey && key != FetchOperationKey {
		return ""
	}
	sum := sha256.Sum256([]byte(ContractSHA256 + ":" + key))
	return hex.EncodeToString(sum[:])
}
