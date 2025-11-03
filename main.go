package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	signal.Notify(ch, syscall.SIGTERM)
	fmt.Println("server started")
	go func() {
		oscall := <-ch
		log.Warn().Msgf("system call:%+v", oscall)
		cancel()
	}()

	r := mux.NewRouter()

	r.HandleFunc("/", handler)

	// start: set up any of your logger configuration here if necessary

	logConfig()
	// end: set up any of your logger configuration here

	server := &http.Server{
		Addr:    ":8080",
		Handler: logMiddleware(r),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to listen and serve http server")
		}
	}()
	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		log.Error().Err(err).Msg("failed to shutdown http server gracefully")
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fmt.Println("request received")
	name := r.URL.Query().Get("name")
	res, err := greeting(ctx, name)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).
			Str("name", name).
			Msg("greeting failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(res))
}

func greeting(ctx context.Context, name string) (string, error) {
	log.Ctx(ctx).Info().Str("name", name).Msg("greeting")

	funcA(ctx)

	if name == "" {
		return "", fmt.Errorf("name is empty")
	}

	if len(name) < 5 {
		return fmt.Sprintf("Hello %s! Your name is to short\n", name), nil
	}
	return fmt.Sprintf("Hi %s", name), nil
}

func logConfig() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	lf, err := os.OpenFile("logs/app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open log file")
		panic(err)
	}

	multiWriter := zerolog.MultiLevelWriter(os.Stdout, lf)
	log.Logger = zerolog.New(multiWriter).With().Timestamp().Logger()
	log.Info().Msg("log configured")

}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log := log.With().
			Str("request_id", uuid.New().String()).
			Logger()

		ctx := log.WithContext(r.Context())

		log.Debug().
			Ctx(ctx).
			Str("uri", r.URL.String()).
			Str("method", r.Method).
			Str("IP", r.RemoteAddr).
			Msg("request received")

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func funcA(ctx context.Context) {
	log.Ctx(ctx).Info().Msg("funcA")
	funcB(ctx)

}

func funcB(ctx context.Context) {
	log.Ctx(ctx).Info().Msg("funcB")
}
