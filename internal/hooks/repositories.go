package hooks

import (
	"errors"
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const DefaultRepositoryID = "pb91u2l315h29a5"

func AddRepositoryHooks(app *pocketbase.PocketBase) {
	app.OnRecordUpdateRequest(collections.Repositories).BindFunc(func(e *core.RecordRequestEvent) error {
		if e.Record.Id != DefaultRepositoryID {
			return e.Next()
		}

		currentRecord, err := e.App.FindRecordById(e.Collection, e.Record.Id)
		if err != nil {
			return err
		}

		currentRecord.Set("retention", e.Record.GetInt("retention"))
		currentRecord.Set("disabled", false)
		e.Record = currentRecord
		return e.Next()
	})

	app.OnRecordDeleteRequest(collections.Repositories).BindFunc(func(e *core.RecordRequestEvent) error {
		if e.Record.Id == DefaultRepositoryID {
			return errors.New("default PocketBase repository cannot be deleted")
		}
		return e.Next()
	})
}
