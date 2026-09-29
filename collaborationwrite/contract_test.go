package collaborationwrite_test

import (
	"encoding/json"
	"strings"
	"testing"

	write "github.com/domainry/domainry-connector-sdk/collaborationwrite"
)

func TestContractDeclaresOnlyBoundedDirectSend(t *testing.T) {
	hash := write.OperationSHA256(write.SendOperationKey)
	if len(hash) != 64 || hash != write.OperationSHA256(write.SendOperationKey) || write.OperationSHA256("send_message") != "" {
		t.Fatalf("operation hash=%q", hash)
	}
	request := write.SendRequest{Recipient: "buyer@example.test", Text: "已确认，下周见。"}
	raw, err := json.Marshal(request)
	var decoded write.SendRequest
	if err != nil || json.Unmarshal(raw, &decoded) != nil || decoded != request {
		t.Fatalf("round trip request=%+v raw=%s err=%v", decoded, raw, err)
	}
	for _, invalid := range []string{
		`{"recipient":"buyer@example.test","text":"ok","access_token":"secret"}`,
		`{"recipient":"Buyer <buyer@example.test>","text":"ok"}`,
		`{"recipient":"one@example.test,two@example.test","text":"ok"}`,
		`{"recipient":"buyer@example.test","text":""}`,
		`{"recipient":"buyer@example.test","text":"ok"} {}`,
	} {
		if json.Unmarshal([]byte(invalid), &decoded) == nil {
			t.Fatalf("invalid direct send accepted: %.100s", invalid)
		}
	}
	tooLong := write.SendRequest{Recipient: "buyer@example.test", Text: strings.Repeat("x", 8001)}
	if tooLong.Validate() == nil {
		t.Fatal("message text limit was not enforced")
	}
}

func TestReceiptBindsRequestAndCannotClaimDelivery(t *testing.T) {
	receipt := write.Result{RequestRef: "account-write:1", Status: "accepted", MessageID: "message-1", AcceptedAt: "2026-09-28T12:00:00+08:00", Delivery: "unknown"}
	if err := receipt.Validate("account-write:1"); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*write.Result){
		"another request": func(r *write.Result) { r.RequestRef = "account-write:2" },
		"no message":      func(r *write.Result) { r.MessageID = "" },
		"claimed delivery": func(r *write.Result) {
			r.Delivery = "delivered"
		},
		"ambiguous time": func(r *write.Result) { r.AcceptedAt = "2026-09-28T12:00:00" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := receipt
			change(&changed)
			if changed.Validate("account-write:1") == nil {
				t.Fatal("invalid collaboration receipt accepted")
			}
		})
	}
}
