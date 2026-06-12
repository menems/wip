package app

import (
	"context"
	"os"
	"syscall"

	"github.com/oklog/run"
	// "go.uber.org/zap"
)

type Engine interface {
	Start() error
	Stop()
}

type Option func(*App)

type App struct {
	// logger  *zap.Logger
	engines []Engine
}

func WithEngine(srv Engine) Option {
	return func(a *App) {
		a.engines = append(a.engines, srv)
	}
}

func New(opts ...Option) *App {

	app := &App{
		// logger: zap.NewNop(),
	}

	for _, option := range opts {
		option(app)
	}
	return app
}

// func ZapLogger(logger *zap.Logger) Option {
// return func(a *App) {
// a.logger = logger
// }
// }

// Start application serving any server set.
// returns an error in case of SIGTERM/SIGKLL during execution
// In case one of the server raising an error, it trigger it own Shutdown()
func (a *App) Start() error {

	var g run.Group

	// a.logger.Info("starting app")
	// defer a.logger.Info("app stopped")

	g.Add(run.SignalHandler(context.Background(), os.Interrupt, syscall.SIGTERM))

	for _, srv := range a.engines {
		func(srv Engine) {
			g.Add(func() error {
				return srv.Start()
			}, func(error) {
				srv.Stop()
			})
		}(srv)
	}

	if err := g.Run(); err != nil {
		if _, ok := err.(run.SignalError); !ok {
			return err
		}
	}
	return nil
}
