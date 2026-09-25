package usage

import (
	"context"
	"testing"
)

func TestRecordByStage(t *testing.T) {
	Reset()
	ctx := WithStage(context.Background(), "clasificacion")
	Record(ctx, 0.01, 100, 50)
	Record(ctx, 0.02, 100, 50)
	Record(context.Background(), 0.005, 10, 5)

	by := ByStage()
	if got := by["clasificacion"]; got.Requests != 2 || got.PromptTokens != 200 || got.CostUSD < 0.0299 || got.CostUSD > 0.0301 {
		t.Errorf("clasificacion = %+v", got)
	}
	if by["otros"].Requests != 1 {
		t.Errorf("otros = %+v", by["otros"])
	}
	if tot := Total(); tot < 0.0349 || tot > 0.0351 {
		t.Errorf("Total = %v", tot)
	}
	if ReasoningDisabled(ctx) || !ReasoningDisabled(WithoutReasoning(ctx)) {
		t.Error("WithoutReasoning no se propaga")
	}
}
