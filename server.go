package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gramework/gramework"
	"github.com/gramework/gramework/gqlhandler"
)

// handleShutdown executes custom action at shutdown
func handleShutdown(sdAction func()) {
	var signalChan = make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGTERM)
	signal.Notify(signalChan, syscall.SIGINT)

	go func() {
		<-signalChan
		sdAction()
	}()
}

func main() {
	conf := getConfig()
	rh := newRepoHandler(conf)
	schema, err := getSchema(conf.SchemaGQL, rh)
	if err != nil {
		log.Fatal(err)
	}

	gqlState, err := gqlhandler.New(schema)
	if err != nil {
		log.Fatal(err)
	}

	app := gramework.New()

	handleShutdown(func() {
		rh.shutdown()
		app.Shutdown()
	})

	app.POST("/graphql", gqlState.Handler)

	app.ListenAndServe(fmt.Sprintf("%s:%s", conf.Host.Address, conf.Host.Port))
}
