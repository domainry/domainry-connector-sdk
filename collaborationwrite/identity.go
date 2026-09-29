package collaborationwrite

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
)

const ContractVersion = "collaboration-write-v1"

func ComputedContractSHA256() string {
	var builder strings.Builder
	builder.WriteString(ContractVersion + ";single-recipient;plain-text;provider-accepted;delivery-unknown;request-ref-envelope-owned;uncertain-never-replay;")
	for _, value := range []any{SendRequest{}, Result{}} {
		typeOf := reflect.TypeOf(value)
		builder.WriteString(typeOf.Name() + "{")
		for index := 0; index < typeOf.NumField(); index++ {
			field := typeOf.Field(index)
			builder.WriteString(field.Name + ":" + field.Type.String() + ":" + field.Tag.Get("json") + ";")
		}
		builder.WriteString("}")
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func OperationSHA256(key string) string {
	if key != SendOperationKey {
		return ""
	}
	sum := sha256.Sum256([]byte(ComputedContractSHA256() + ":" + key))
	return hex.EncodeToString(sum[:])
}
