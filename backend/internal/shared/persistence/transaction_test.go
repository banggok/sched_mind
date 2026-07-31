package persistence

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

func TestSerializeScheduleMutationMarksContextForNestedTransactionCallbacks_AC35_AC36(t *testing.T) {
	if ScheduleMutationSerialized(context.Background()) {
		t.Fatal("plain context unexpectedly reports schedule-mutation serialization")
	}

	ctx, release := SerializeScheduleMutation(context.Background())
	defer release()
	if !ScheduleMutationSerialized(ctx) {
		t.Fatal("serialized context does not carry the re-entrant lock marker")
	}

	nested := WithTransaction(ctx, &gorm.DB{})
	if !ScheduleMutationSerialized(nested) {
		t.Fatal("transaction context lost the schedule-mutation serialization marker")
	}
}
