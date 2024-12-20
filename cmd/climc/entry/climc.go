// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package entry

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/cheggaaa/pb/v3"
	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/cmd/climc/promputils"
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type BaseOptions struct {
	Debug    bool `help:"Show debug information"`
	Version  bool `help:"Show version"`
	Timeout  int  `default:"600" help:"Number of seconds to wait for a response"`
	Insecure bool `default:"$YUNION_INSECURE|false" help:"Allow skip server cert verification if URL is https" short-token:"k"`

	CertFile string `default:"$YUNION_CERT_FILE" help:"certificate file"`
	KeyFile  string `default:"$YUNION_KEY_FILE" help:"private key file"`

	Completion     string `default:"" help:"Generate climc auto complete script" choices:"bash|zsh"`
	UseCachedToken bool   `default:"$YUNION_USE_CACHED_TOKEN|false" help:"Use cached token"`

	OsUsername string `default:"$OS_USERNAME" help:"Username, defaults to env[OS_USERNAME]"`
	OsPassword string `default:"$OS_PASSWORD" help:"Password, defaults to env[OS_PASSWORD]"`
	// OsProjectId string `default:"$OS_PROJECT_ID" help:"Proejct ID, defaults to env[OS_PROJECT_ID]"`
	OsProjectName   string `default:"$OS_PROJECT_NAME" help:"Project name, defaults to env[OS_PROJECT_NAME]"`
	OsProjectDomain string `default:"$OS_PROJECT_DOMAIN" help:"Domain name of project, defaults to env[OS_PROJECT_DOMAIN]"`
	OsDomainName    string `default:"$OS_DOMAIN_NAME" help:"Domain name, defaults to env[OS_DOMAIN_NAME]"`

	OsAccessKey string `default:"$OS_ACCESS_KEY" help:"ak/sk access key, defaults to env[OS_ACCESS_KEY]"`
	OsSecretKey string `default:"$OS_SECRET_KEY" help:"ak/s secret, defaults to env[OS_SECRET_KEY]"`

	OsAuthToken string `default:"$OS_AUTH_TOKEN" help:"token authenticate, defaults to env[OS_AUTH_TOKEN]"`

	OsAuthURL string `default:"$OS_AUTH_URL" help:"Defaults to env[OS_AUTH_URL]"`

	OsRegionName   string `default:"$OS_REGION_NAME" help:"Defaults to env[OS_REGION_NAME]"`
	OsZoneName     string `default:"$OS_ZONE_NAME" help:"Defaults to env[OS_ZONE_NAME]"`
	OsEndpointType string `default:"$OS_ENDPOINT_TYPE|internalURL" help:"Defaults to env[OS_ENDPOINT_TYPE] or internalURL" choices:"publicURL|internalURL|adminURL"`
	// ApiVersion     string `default:"$API_VERSION" help:"override default modules service api version"`
	OutputFormat string `default:"$CLIMC_OUTPUT_FORMAT|table" choices:"table|kv|json|yaml|flatten-table|flatten-kv" help:"output format"`

	ParallelRun int `help:"run in parallel to stess test the performance of server"`

	SUBCOMMAND string `help:"climc subcommand" subcommand:"true"`
}

func getSubcommandsParser() (*structarg.ArgumentParser, error) {
	var (
		prog = "climc"
		desc = `Command-line interface to the API server.`
	)
	parse, e := structarg.NewArgumentParserWithHelp(&BaseOptions{},
		prog,
		desc,
		`See "climc help COMMAND" for help on a specific command.`)
	if e != nil {
		return nil, e
	}
	subcmd := parse.GetSubcommand()
	if subcmd == nil {
		return nil, fmt.Errorf("No subcommand argument")
	}

	promptRootCmd := promputils.InitRootCmd(prog, desc, parse.GetOptArgs(), parse.GetPosArgs())
	var errs []error
	for _, v := range shell.CommandTable {
		_par, e := subcmd.AddSubParserWithHelp(v.Options, v.Command, v.Desc, v.Callback)

		if e != nil {
			errs = append(errs, e)
			continue
		}
		promputils.AppendCommand(promptRootCmd, v.Command, v.Desc)
		cmd := v.Command

		for _, v := range _par.GetOptArgs() {
			text := v.String()
			text = strings.TrimLeft(text, "[<")
			text = strings.TrimRight(text, "]>")
			promputils.AppendOpt(cmd, text, v.HelpString(""), v)
		}
		for _, v := range _par.GetPosArgs() {
			text := v.String()
			text = strings.TrimLeft(text, "[<")
			text = strings.TrimRight(text, "]>")
			promputils.AppendPos(cmd, text, v.HelpString(""), v)
		}
	}
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}
	return parse, nil
}

func showErrorAndExit(e error) {
	fmt.Fprintf(os.Stderr, "%s", e)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func newClientSession(options *BaseOptions) (*mcclient.ClientSession, error) {
	if len(options.OsAuthURL) == 0 {
		return nil, fmt.Errorf("Missing OS_AUTH_URL")
	}
	if len(options.OsUsername) == 0 && len(options.OsAccessKey) == 0 && len(options.OsAuthToken) == 0 {
		return nil, fmt.Errorf("Missing OS_USERNAME or OS_ACCESS_KEY or OS_AUTH_TOKEN")
	}
	if len(options.OsUsername) > 0 && len(options.OsPassword) == 0 {
		return nil, fmt.Errorf("Missing OS_PASSWORD")
	}
	if len(options.OsAccessKey) > 0 && len(options.OsSecretKey) == 0 {
		return nil, fmt.Errorf("Missing OS_SECRET_KEY")
	}

	logLevel := "info"
	if options.Debug {
		logLevel = "debug"
	}
	log.SetLogLevelByString(log.Logger(), logLevel)

	client := mcclient.NewClient(options.OsAuthURL,
		options.Timeout,
		options.Debug,
		options.Insecure,
		options.CertFile,
		options.KeyFile)

	cfg := &httpproxy.Config{
		HTTPProxy:  os.Getenv("HTTP_PROXY"),
		HTTPSProxy: os.Getenv("HTTPS_PROXY"),
		NoProxy:    os.Getenv("NO_PROXY"),
	}

	cfgProxyFunc := cfg.ProxyFunc()
	proxyFunc := func(req *http.Request) (*url.URL, error) {
		return cfgProxyFunc(req.URL)
	}
	httputils.SetClientProxyFunc(client.GetClient(), proxyFunc)

	var cacheToken mcclient.TokenCredential
	authUrlAlter := strings.Replace(options.OsAuthURL, "/", "", -1)
	authUrlAlter = strings.Replace(authUrlAlter, ":", "", -1)
	tokenCachePath := filepath.Join(os.TempDir(), fmt.Sprintf("OS_AUTH_CACHE_TOKEN-%s-%s-%s-%s", authUrlAlter, options.OsUsername, options.OsDomainName, options.OsProjectName))
	if options.UseCachedToken {
		cacheFile, err := os.Open(tokenCachePath)
		if err == nil && cacheFile != nil {
			fileInfo, _ := cacheFile.Stat()
			dur, err := time.ParseDuration("-24h")
			if fileInfo != nil && err == nil && fileInfo.ModTime().After(time.Now().Add(dur)) {
				bytesToken, err := ioutil.ReadAll(cacheFile)
				if err == nil {
					token := client.NewAuthTokenCredential()
					err := json.Unmarshal(bytesToken, token)
					if err != nil {
						fmt.Printf("Unmarshal token error:%s", err)
					} else if token.IsValid() {
						cacheToken = token
					}
				}
				cacheFile.Close()
			}
		}
	}

	if cacheToken == nil {
		var token mcclient.TokenCredential
		var err error
		if len(options.OsAuthToken) > 0 {
			token, err = client.AuthenticateToken(options.OsAuthToken, options.OsProjectName,
				options.OsProjectDomain,
				mcclient.AuthSourceCli)
		} else if len(options.OsAccessKey) > 0 {
			token, err = client.AuthenticateByAccessKey(options.OsAccessKey,
				options.OsSecretKey, mcclient.AuthSourceCli)
		} else {
			token, err = client.AuthenticateWithSource(options.OsUsername,
				options.OsPassword,
				options.OsDomainName,
				options.OsProjectName,
				options.OsProjectDomain,
				mcclient.AuthSourceCli)
		}
		if err != nil {
			return nil, err
		}
		cacheToken = token
		bytesCacheToken, err := json.Marshal(cacheToken)
		if err != nil {
			fmt.Printf("Marshal token error:%s", err)
		} else {
			fo, err := os.Create(tokenCachePath)
			if err != nil {
				fmt.Printf("Save token cache fail: %s", err)
			} else {
				fo.Write(bytesCacheToken)
				fo.Close()
			}
		}
	}

	session := client.NewSession(
		context.Background(),
		options.OsRegionName,
		options.OsZoneName,
		options.OsEndpointType,
		cacheToken,
	)
	return session, nil
}

func enterInteractiveMode(
	parser *structarg.ArgumentParser,
	sessionFactory func() *mcclient.ClientSession,
) {
	promputils.InitEnv(parser, sessionFactory())
	defer fmt.Println("Bye!")
	p := prompt.New(
		promputils.Executor,
		promputils.Completer,
		prompt.OptionPrefix("climc> "),
		prompt.OptionTitle("Climc, a Command Line Interface to Manage Clouds"),
		prompt.OptionMaxSuggestion(16),
	)
	p.Run()
}

func executeSubcommand(
	subcmd *structarg.SubcommandArgument,
	subparser *structarg.ArgumentParser,
	options *BaseOptions,
	sessionFactory func() *mcclient.ClientSession,
	parallel int,
) {
	suboptions := subparser.Options()
	if subparser.IsHelpSet() {
		helpStr, err := subcmd.SubHelpString(options.SUBCOMMAND)
		if err != nil {
			showErrorAndExit(err)
			return
		}
		fmt.Println(helpStr)
		return
	}
	if parallel <= 1 {
		sess := sessionFactory()
		err := subcmd.Invoke(sess, suboptions)
		if err != nil {
			showErrorAndExit(err)
			return
		}
	} else {
		fmt.Println("Authenticating...")
		bar := pb.StartNew(parallel)
		sess := make([]*mcclient.ClientSession, parallel)
		for i := 0; i < parallel; i++ {
			sess[i] = sessionFactory()
			bar.Increment()
		}
		bar.Finish()
		fmt.Println("Tokens are ready, start to request ...")
		start := time.Now()
		var wg sync.WaitGroup
		var errs []error
		for i := 0; i < parallel; i++ {
			wg.Add(1)
			s := sess[i]
			go func() {
				defer wg.Done()
				err := subcmd.Invoke(s, suboptions)
				if err != nil {
					errs = append(errs, err)
				}
			}()
		}
		wg.Wait()
		if len(errs) > 0 {
			showErrorAndExit(errors.NewAggregate(errs))
			return
		}
		diff := time.Now().Sub(start)
		fmt.Printf("cost: %f seconds %f qps\n", diff.Seconds(), float64(parallel)/diff.Seconds())
	}
}

func ClimcMain() {
	parser, e := getSubcommandsParser()
	if e != nil {
		showErrorAndExit(e)
		return
	}
	e = parser.ParseArgs(os.Args[1:], false)
	options := parser.Options().(*BaseOptions)

	if len(options.Completion) > 0 {
		completeScript := promputils.GenerateAutoCompleteCmds(promputils.GetRootCmd(), options.Completion)
		if len(completeScript) > 0 {
			fmt.Printf("%s", completeScript)
		}
		return
	}

	if parser.IsHelpSet() {
		return
	}

	if options.Version {
		fmt.Printf("Yunion API client version:\n %s\n", version.GetJsonString())
		return
	}

	shell.OutputFormat(options.OutputFormat)
	ensureSessionFactory := func() *mcclient.ClientSession {
		session, err := newClientSession(options)
		if err != nil {
			showErrorAndExit(err)
			return nil
		}
		return session
	}

	// enter interactive mode when not enough argument and SUBCOMMAND is empty
	if _, ok := e.(*structarg.NotEnoughArgumentsError); ok && options.SUBCOMMAND == "" {
		enterInteractiveMode(parser, ensureSessionFactory)
		return
	}

	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if e != nil {
		if subparser != nil {
			fmt.Print(subparser.Usage())
		} else {
			fmt.Print(parser.Usage())
		}
		showErrorAndExit(e)
		return
	}

	// execute subcommand in non-interactive mode
	executeSubcommand(subcmd, subparser, options, ensureSessionFactory, options.ParallelRun)
}
