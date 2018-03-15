package main

import (
	"github.com/kowala-tech/kUSD/console"
	ethn "github.com/kowala-tech/kUSD/node"
	"github.com/kowala-tech/kUSD/params"
	"github.com/sudachen/misc/out"
	cht "github.com/kowala-tech/kUSD/chat/v1"
	_ "github.com/kowala-tech/kUSD/chat/log"
)

func runConsole(listenAddr string) (err error) {
	ncfg := ethn.DefaultConfig
	ncfg.Name = "test-cht"
	ncfg.Version = params.VersionWithCommit("")
	ncfg.HTTPModules = append(ncfg.HTTPModules, "cht")
	ncfg.WSModules = append(ncfg.WSModules, "cht")
	ncfg.P2P.ListenAddr = listenAddr
	//ncfg.IPCPath = fmt.Sprintf("cht.%s.ipc",strings.Replace(listenAddr,":",".",0))

	stk, err := ethn.New(&ncfg)
	if err != nil {
		return
	}

	c := cht.New(nil)
	err = stk.Register(func(n *ethn.ServiceContext) (ethn.Service, error) {
		return c, nil
	})
	if err != nil {
		return
	}

	err = stk.Start()
	if err != nil {
		return
	}
	defer stk.Stop()

	clnt, err := stk.Attach()
	if err != nil {
		return err
	}

	cons, err := console.New(console.Config{
		DataDir: stk.InstanceDir(),
		DocRoot: "testdata",
		Client:  clnt,
	})

	if err != nil {
		return err
	}

	defer cons.Stop(true)

	cons.Evaluate(cht.Console_JS)

	cons.Welcome()
	cons.Interactive()

	return
}

func init() {
	out.Trace.SetCurrent()
}

func main() {
	err := runConsole("127.0.0.1:29999")
	if err != nil {
		out.StdErr.Print(err)
	}
}
