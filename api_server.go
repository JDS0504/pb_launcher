package main

import (
	"context"
	"log/slog"
	"pb_launcher/configs"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"go.uber.org/fx"
)

func StartApiServer(
	lc fx.Lifecycle,
	pb *pocketbase.PocketBase,
	config configs.Config,
	apiConfig *apis.ServeConfig,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var apiReadyWg sync.WaitGroup
			apiReadyWg.Add(1)

			pb.OnServe().BindFunc(func(e *core.ServeEvent) error {
				e.InstallerFunc = nil
				if err := e.Next(); err != nil {
					return err
				}
				apiReadyWg.Done()
				return nil
			})

			go func() {
				slog.Info("Starting API server", "address", apiConfig.HttpAddr)
				if err := apis.Serve(pb.App, *apiConfig); err != nil {
					slog.Error("API server encountered an error", "error", err)
					return
				}
				slog.Info("API server stopped gracefully")
			}()

			apiReadyWg.Wait()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			slog.Info("Initiating shutdown sequence for PocketBase")

			terminateEvent := new(core.TerminateEvent)
			terminateEvent.App = pb

			if err := pb.OnTerminate().Trigger(terminateEvent, func(e *core.TerminateEvent) error {
				slog.Info("Resetting PocketBase bootstrap state")
				if err := e.App.ResetBootstrapState(); err != nil {
					slog.Error("Failed to reset bootstrap state", "error", err)
					return err
				}
				slog.Info("Bootstrap state reset successfully")
				return nil
			}); err != nil {
				slog.Error("Error during termination event handling", "error", err)
				return err
			}

			slog.Info("PocketBase shutdown completed successfully")
			return nil
		},
	})
}
