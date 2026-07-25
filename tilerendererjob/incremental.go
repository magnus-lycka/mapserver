package tilerendererjob

import (
	"mapserver/app"
	"mapserver/db"
	"time"

	"github.com/sirupsen/logrus"
)

type IncrementalRenderEvent struct {
	LastMtime int64 `json:"lastmtime"`
}

func incrementalRender(ctx *app.App, cursor *db.IncrementalCursor) {
	safetyLag, err := time.ParseDuration(ctx.Config.IncrementalRenderingSafetyLag)
	if err != nil {
		panic(err)
	}
	renderingDuration, err := time.ParseDuration(ctx.Config.IncrementalRenderingTimer)
	if err != nil {
		panic(err)
	}

	fields := logrus.Fields{
		"cursor":    cursor,
		"safetyLag": safetyLag,
	}
	logrus.WithFields(fields).Info("Starting incremental rendering job")
	watermarkMode := ""

	for {
		start := time.Now()
		watermark, err := ctx.Blockdb.GetIncrementalWatermark(safetyLag)
		if err != nil {
			panic(err)
		}
		if watermark.Mode != watermarkMode {
			watermarkMode = watermark.Mode
			logrus.WithFields(logrus.Fields{
				"mode":       watermark.Mode,
				"safetyLag":  safetyLag,
				"upperMtime": watermark.UpperMtime,
			}).Info("Incremental watermark mode")
		}

		result, err := ctx.MapBlockAccessor.FindMapBlocksByMtime(
			cursor,
			watermark.UpperMtime,
			ctx.Config.RenderingFetchLimit,
			ctx.Config.Layers,
		)

		if err != nil {
			panic(err)
		}

		if result.UnfilteredCount == 0 {
			time.Sleep(renderingDuration)
			continue
		}

		tiles := 0
		if len(result.List) > 0 {
			tiles = renderMapblocks(ctx, result.List)
		}

		cursor = result.LastCursor
		if err := saveIncrementalCursor(ctx.Settings, cursor); err != nil {
			panic(err)
		}

		t := time.Now()
		elapsed := t.Sub(start)

		ev := IncrementalRenderEvent{
			LastMtime: cursor.Mtime,
		}

		ctx.WebEventbus.Emit("incremental-render-progress", &ev)

		fields := logrus.Fields{
			"mapblocks": len(result.List),
			"tiles":     tiles,
			"elapsed":   elapsed,
			"lastMtime": cursor.Mtime,
		}
		logrus.WithFields(fields).Info("incremental rendering")
	}
}
