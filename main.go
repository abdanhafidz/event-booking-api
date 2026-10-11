package main

import (
	"log/slog"
	"os"

	"abdanhafidz.com/event-booking-api/provider"
	"abdanhafidz.com/event-booking-api/router"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	appProvider := provider.NewAppProvider()
	router.RunRouter(appProvider)
}
