package floxy

import (
	"strings"
	"testing"
	"time"
)

func TestParseWorkflowYAML_TaskShorthandOnFailureAndOptions(t *testing.T) {
	yaml := `
handlers:
  - name: reserve_stock
    exec: ./handlers/stock_reserve.sh
  - name: charge_payment
    exec: ./handlers/payment_charge.sh
  - name: refund_payment
    exec: ./handlers/payment_refund.sh

flows:
  - name: order_flow
    steps:
      - name: reserve_stock
        handler: reserve_stock
        on_failure: refund_payment
        max_retries: 7
        no_idempotent: true
        delay: 1500
        retry_delay: 3000
        retry_strategy: exponential
        timeout: 4500
        metadata:
          k1: v1

      - charge_payment
`

	defs, handlersExec, err := ParseWorkflowYAML([]byte(yaml), 1)
	if err != nil {
		t.Fatalf("ParseWorkflowYAML error: %v", err)
	}

	// handlers exec map
	if handlersExec["reserve_stock"] != "./handlers/stock_reserve.sh" {
		t.Fatalf("unexpected exec for reserve_stock: %v", handlersExec["reserve_stock"])
	}
	if handlersExec["refund_payment"] != "./handlers/payment_refund.sh" {
		t.Fatalf("unexpected exec for refund_payment: %v", handlersExec["refund_payment"])
	}

	def := defs["order_flow"]
	if def == nil {
		t.Fatalf("order_flow definition not found")
	}
	if def.Definition.Start != "reserve_stock" {
		t.Fatalf("unexpected start: %s", def.Definition.Start)
	}

	rs := def.Definition.Steps["reserve_stock"]
	if rs == nil {
		t.Fatalf("reserve_stock step missing")
	}
	if rs.Type != StepTypeTask {
		t.Fatalf("reserve_stock type: %v", rs.Type)
	}
	if rs.Handler != "reserve_stock" {
		t.Fatalf("reserve_stock handler: %s", rs.Handler)
	}
	if rs.MaxRetries != 7 {
		t.Fatalf("reserve_stock max_retries: %d", rs.MaxRetries)
	}
	if !rs.NoIdempotent {
		t.Fatalf("reserve_stock no_idempotent expected true")
	}
	if rs.Delay != 1500*time.Millisecond {
		t.Fatalf("reserve_stock delay: %v", rs.Delay)
	}
	if rs.RetryDelay != 3000*time.Millisecond {
		t.Fatalf("reserve_stock retry_delay: %v", rs.RetryDelay)
	}
	if rs.RetryStrategy != RetryStrategyExponential {
		t.Fatalf("reserve_stock retry_strategy: %v", rs.RetryStrategy)
	}
	if rs.Timeout != 4500*time.Millisecond {
		t.Fatalf("reserve_stock timeout: %v", rs.Timeout)
	}

	// Metadata should include exec from handler map and custom key
	if rs.Metadata == nil || rs.Metadata["exec"] != "./handlers/stock_reserve.sh" || rs.Metadata["k1"] != "v1" {
		t.Fatalf("reserve_stock metadata unexpected: %+v", rs.Metadata)
	}

	// on_failure compensation step should be created with exec metadata
	if rs.OnFailure != "refund_payment" {
		t.Fatalf("reserve_stock on_failure: %s", rs.OnFailure)
	}
	comp := def.Definition.Steps["refund_payment"]
	if comp == nil {
		t.Fatalf("compensation step missing")
	}
	if comp.Type != StepTypeTask || comp.Handler != "refund_payment" {
		t.Fatalf("compensation invalid: type=%v handler=%s", comp.Type, comp.Handler)
	}
	if comp.Metadata == nil || comp.Metadata["exec"] != "./handlers/payment_refund.sh" {
		t.Fatalf("compensation metadata exec missing: %+v", comp.Metadata)
	}

	// shorthand task
	cp := def.Definition.Steps["charge_payment"]
	if cp == nil {
		t.Fatalf("charge_payment step missing")
	}
	if cp.Handler != "charge_payment" {
		t.Fatalf("charge_payment handler: %s", cp.Handler)
	}
	if cp.Metadata == nil || cp.Metadata["exec"] != "./handlers/payment_charge.sh" {
		t.Fatalf("charge_payment exec metadata: %+v", cp.Metadata)
	}
}

func TestParseWorkflowYAML_ParallelAutoJoin(t *testing.T) {
	yaml := `
handlers:
  - name: a
    exec: ./a.sh
  - name: b
    exec: ./b.sh

flows:
  - name: f
    steps:
      - name: s0
        handler: a
      - type: parallel
        name: p
        tasks:
          - name: ta
            handler: a
          - name: tb
            handler: b
`
	defs, _, err := ParseWorkflowYAML([]byte(yaml), 1)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	def := defs["f"]
	if def == nil {
		t.Fatalf("flow f missing")
	}

	p := def.Definition.Steps["p"]
	if p == nil || p.Type != StepTypeParallel {
		t.Fatalf("parallel step invalid: %+v", p)
	}
	// p.Next should contain auto join
	if len(p.Next) != 1 || p.Next[0] != "p_join" {
		t.Fatalf("parallel next unexpected: %+v", p.Next)
	}
	// tasks should be present and prev set to p
	ta := def.Definition.Steps["ta"]
	tb := def.Definition.Steps["tb"]
	if ta == nil || tb == nil {
		t.Fatalf("parallel tasks missing: ta=%v tb=%v", ta, tb)
	}
	if ta.Prev != "p" || tb.Prev != "p" {
		t.Fatalf("tasks prev unexpected: ta.prev=%s tb.prev=%s", ta.Prev, tb.Prev)
	}
	// join auto-created
	j := def.Definition.Steps["p_join"]
	if j == nil || j.Type != StepTypeJoin {
		t.Fatalf("join invalid: %+v", j)
	}
	if j.JoinStrategy != JoinStrategyAll {
		t.Fatalf("join strategy: %v", j.JoinStrategy)
	}
	// WaitFor should include both tasks
	if len(j.WaitFor) != 2 {
		t.Fatalf("join wait_for len: %d", len(j.WaitFor))
	}
	// Order is not guaranteed; check membership
	wf := strings.Join(j.WaitFor, ",")
	if !(strings.Contains(wf, "ta") && strings.Contains(wf, "tb")) {
		t.Fatalf("join wait_for: %+v", j.WaitFor)
	}
}

func TestParseWorkflowYAML_ConditionElse(t *testing.T) {
	yaml := `
handlers:
  - name: a
    exec: ./a.sh
  - name: b
    exec: ./b.sh

flows:
  - name: f
    steps:
      - name: s1
        handler: a
      - type: condition
        name: c1
        expr: ".eq input.ok true"
        else:
          - name: s2
            handler: b
`
	defs, _, err := ParseWorkflowYAML([]byte(yaml), 1)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	def := defs["f"]
	if def == nil {
		t.Fatalf("flow f missing")
	}

	c1 := def.Definition.Steps["c1"]
	if c1 == nil || c1.Type != StepTypeCondition {
		t.Fatalf("condition step invalid: %+v", c1)
	}
	if c1.Condition != ".eq input.ok true" {
		t.Fatalf("condition expr unexpected: %s", c1.Condition)
	}
	if c1.Else == "" {
		t.Fatalf("condition else not linked")
	}
	s2 := def.Definition.Steps[c1.Else]
	if s2 == nil || s2.Prev != "c1" {
		t.Fatalf("else step invalid: %+v", s2)
	}
}

func TestValidateYAMLDocument_NoFlows(t *testing.T) {
	yaml := `
handlers:
  - name: a
    exec: ./a.sh
`
	if err := ValidateYAMLDocument([]byte(yaml)); err == nil {
		t.Fatalf("expected error for no flows")
	}
}

func TestParseWorkflowYAML_NegativeCases(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "handler missing name",
			yaml: `handlers:\n  - exec: ./a.sh\nflows:\n  - name: f\n    steps:\n      - name: s\n        handler: h\n`,
		},
		{
			name: "handler missing exec",
			yaml: `handlers:\n  - name: a\nflows:\n  - name: f\n    steps:\n      - name: s\n        handler: a\n`,
		},
		{
			name: "flow missing name",
			yaml: `handlers:\n  - name: h\n    exec: ./h.sh\nflows:\n  - steps:\n      - name: s\n        handler: h\n`,
		},
		{
			name: "flow no steps",
			yaml: `handlers:\n  - name: h\n    exec: ./h.sh\nflows:\n  - name: f\n    steps: []\n`,
		},
		{
			name: "parallel less than 2 tasks",
			yaml: `handlers:\n  - name: h\n    exec: ./h.sh\nflows:\n  - name: f\n    steps:\n      - type: parallel\n        name: p\n        tasks:\n          - name: s1\n            handler: h\n`,
		},
		{
			name: "condition missing expr",
			yaml: `handlers:\n  - name: h\n    exec: ./h.sh\nflows:\n  - name: f\n    steps:\n      - type: condition\n        name: c1\n        else:\n          - name: s\n            handler: h\n`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, _, err := ParseWorkflowYAML([]byte(c.yaml), 1); err == nil {
				t.Fatalf("expected error for case %q", c.name)
			}
		})
	}
}
