package mail

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
)

const ContractVersion = "mail-read-v1"
const ContractSHA256 = "8557d94966acb87191b43fa6489c64ada908244771f8bd57afe0cddd33e76511"
const (
	ListOperationKey   = "mail_list"
	SearchOperationKey = "mail_search"
	ReadOperationKey   = "mail_read"
)

func ComputedContractSHA256() string {
	var b strings.Builder
	b.WriteString(ContractVersion + ";read-only;provider-query-syntax-gmail-graph-kql-feishu-mail;metadata-list;page-default-10-max-25;cursor-8192;query-1024;body-default-16384-max-65536-bytes;attachments-excluded;unknown-never-complete;source-account-owned;addresses-50;subject-2048;name-512;")
	for _, v := range []any{PageRequest{}, SearchRequest{}, ReadRequest{}, Address{}, Summary{}, MessagesPage{}, Body{}, Message{}} {
		t := reflect.TypeOf(v)
		b.WriteString(t.Name() + "{")
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			b.WriteString(f.Name + ":" + f.Type.String() + ":" + f.Tag.Get("json") + ";")
		}
		b.WriteString("}")
	}
	h := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(h[:])
}

func OperationSHA256(key string) string {
	switch key {
	case ListOperationKey, SearchOperationKey, ReadOperationKey:
	default:
		return ""
	}
	h := sha256.Sum256([]byte(ContractSHA256 + ":" + key))
	return hex.EncodeToString(h[:])
}
