package main

import (
	"context"
	"csi-azure-fake-provider/fake"
	"flag"
	"log"
	"os"
	"os/signal"
)

var (
	socketPath = flag.String("socket", "/etc/kubernetes/secrets-store-csi-providers/azure.sock", "The socket address")
	valuesPath = flag.String("valuesDir", "/values/", "The values directory path")
)

func main() {
	flag.Parse()
	s, err := fake.NewMocKCSIProviderServer(*socketPath, *valuesPath)
	if err != nil {
		log.Fatalln(err)
	}

	s.SetObjects(map[string]string{"fake": "1"})
	err = s.Start()
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Everything Started :)")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()
	log.Println("Shutdown request")

	s.Stop()
	log.Println("Shutdown working")
}
