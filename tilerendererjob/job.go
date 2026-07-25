package tilerendererjob

import (
	"mapserver/app"
	"mapserver/db"
	"mapserver/settings"
)

func Job(ctx *app.App) {
	cursor, err := loadIncrementalCursor(ctx.Settings)
	if err != nil {
		panic(err)
	}
	if cursor.Mtime == 0 {
		//mark db time as last incremental render point
		lastMtime, err := ctx.Blockdb.GetTimestamp()

		if err != nil {
			panic(err)
		}

		cursor = db.NewIncrementalCursor(lastMtime, nil)
		if err := saveIncrementalCursor(ctx.Settings, cursor); err != nil {
			panic(err)
		}
	}

	if ctx.Config.EnableInitialRendering {
		if ctx.Settings.GetBool(settings.SETTING_INITIAL_RUN, true) {
			initialRender(ctx)
		}
	}

	incrementalRender(ctx, cursor)

	panic("render job interrupted!")

}
