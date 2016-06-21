package main

import (
	"fmt"
	"net"
	"os"

	"github.com/codegangsta/cli"
	"github.com/ernado/stun"
	"github.com/pkg/errors"
)

const (
	version = "0.2"
)

func wrapWithLogger(f func(c *cli.Context) error) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		err := f(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return err
	}
}

func discover(c *cli.Context) error {
	m := stun.AcquireFields(stun.Message{
		TransactionID: stun.NewTransactionID(),
		Type: stun.MessageType{
			Method: stun.MethodBinding,
			Class:  stun.ClassRequest,
		},
	})
	m.AddSoftware(fmt.Sprintf("cydev/stun %s", version))
	m.WriteHeader()

	request := stun.Request{
		Message: m,
		Target:  stun.Normalize(c.String("server")),
	}

	return stun.DefaultClient.Do(request, func(r stun.Response) error {
		var (
			ip   net.IP
			port int
			err  error
		)
		ip, port, err = r.Message.GetXORMappedAddress()
		if err != nil {
			return errors.Wrap(err, "failed to get ip")
		}
		fmt.Println(ip, port)
		return nil
	})
}

func main() {
	app := cli.NewApp()
	app.Name = "stun"
	app.Usage = "command line client for STUN"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "server",
			Value: "ci.cydev.ru",
			Usage: "STUN server address",
		},
	}
	app.Action = wrapWithLogger(discover)
	app.Version = version
	app.Run(os.Args)
}
