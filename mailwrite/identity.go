package mailwrite

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"

	"github.com/domainry/domainry-connector-sdk/mail"
)

const ContractVersion = "mail-write-v1"
const ContractSHA256 = "18d472fe742b5cdc77a61827fb82b36b4bd1e604b57d8826a70824582f8cbd57"

func ComputedContractSHA256() string {
	var b strings.Builder
	b.WriteString(ContractVersion + ";mail-read:" + mail.ContractSHA256 + ";send-and-reply;plain-text;explicit-to-cc-bcc;recipients-total-50;subject-2048;text-65536;strict-json-1048576;sender-headers-correlation-envelope-owned;reply-current-complete-nondraft;reply-to-precedes-from;subject-preserved;accepted-not-delivered;optional-provider-message-id;delivery-unknown;uncertain-never-replay;")
	for _, value := range []any{Message{}, SendRequest{}, ReplyRequest{}, Result{}} {
		t := reflect.TypeOf(value)
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
	if key != SendOperationKey && key != ReplyOperationKey {
		return ""
	}
	sum := sha256.Sum256([]byte(ContractSHA256 + ":" + key))
	return hex.EncodeToString(sum[:])
}
