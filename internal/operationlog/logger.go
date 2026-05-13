package operationlog

import (
	"context"
	"log/slog"
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type Logger struct {
	app *pocketbase.PocketBase
}

func New(app *pocketbase.PocketBase) *Logger {
	return &Logger{app: app}
}

func (l *Logger) Success(ctx context.Context, serviceID, operation, message string, metadata map[string]any) {
	l.write(ctx, serviceID, operation, "success", message, metadata)
}

func (l *Logger) Error(ctx context.Context, serviceID, operation, message string, metadata map[string]any) {
	l.write(ctx, serviceID, operation, "error", message, metadata)
}

func (l *Logger) write(ctx context.Context, serviceID, operation, status, message string, metadata map[string]any) {
	_ = ctx
	collection, err := l.app.FindCachedCollectionByNameOrId(collections.OperationLogs)
	if err != nil {
		slog.Warn("failed to find operation logs collection", "error", err)
		return
	}

	record := core.NewRecord(collection)
	if serviceID != "" {
		record.Set("service", serviceID)
	}
	record.Set("operation", operation)
	record.Set("status", status)
	record.Set("message", message)
	record.Set("metadata", metadata)

	if err := l.app.Save(record); err != nil {
		slog.Warn("failed to save operation log", "error", err, "operation", operation)
	}
}
